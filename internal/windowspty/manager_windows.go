//go:build windows

package windowspty

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	execpkg "github.com/JuliusBrussee/cavekit/internal/exec"
	"golang.org/x/sys/windows"
)

const (
	defaultCols      = 120
	defaultRows      = 40
	maxVisibleLines  = 200
	registryFileName = "windowspty-sessions.json"
)

type sessionRecord struct {
	Name      string `json:"name"`
	ProcessID uint32 `json:"process_id"`
	WorkDir   string `json:"work_dir"`
	LogPath   string `json:"log_path"`
	Program   string `json:"program"`
}

type runtimeSession struct {
	record      sessionRecord
	console     windows.Handle
	process     windows.Handle
	thread      windows.Handle
	inputWriter *os.File
	outputReader *os.File
	logFile     *os.File
	done        chan struct{}
}

// Manager hosts agent processes inside Windows pseudo consoles.
type Manager struct {
	mu       sync.Mutex
	sessions map[string]*runtimeSession
	registry string
}

func NewManager(_ execpkg.Executor) *Manager {
	return &Manager{
		sessions: make(map[string]*runtimeSession),
		registry: defaultRegistryPath(),
	}
}

func (m *Manager) CreateSession(ctx context.Context, name, workDir, program string) error {
	m.mu.Lock()
	if rt, ok := m.sessions[name]; ok && rt != nil {
		m.mu.Unlock()
		return fmt.Errorf("session already exists: %s", name)
	}
	m.mu.Unlock()
	if m.Exists(ctx, name) {
		return fmt.Errorf("session already exists: %s", name)
	}

	if err := os.MkdirAll(filepath.Dir(m.registry), 0o755); err != nil {
		return err
	}

	logDir := filepath.Join(filepath.Dir(m.registry), "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return err
	}
	logPath := filepath.Join(logDir, sanitizeFileName(name)+".log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return err
	}

	inRead, inWrite, err := newPipePair()
	if err != nil {
		logFile.Close()
		return err
	}
	outRead, outWrite, err := newPipePair()
	if err != nil {
		logFile.Close()
		windows.CloseHandle(inRead)
		windows.CloseHandle(inWrite)
		return err
	}

	var console windows.Handle
	err = windows.CreatePseudoConsole(
		windows.Coord{X: defaultCols, Y: defaultRows},
		inRead,
		outWrite,
		0,
		&console,
	)
	windows.CloseHandle(inRead)
	windows.CloseHandle(outWrite)
	if err != nil {
		logFile.Close()
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		return fmt.Errorf("CreatePseudoConsole: %w", err)
	}

	appName, cmdLine, err := resolveProgram(program)
	if err != nil {
		windows.ClosePseudoConsole(console)
		logFile.Close()
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		return err
	}

	attrList, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		windows.ClosePseudoConsole(console)
		logFile.Close()
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		return err
	}
	defer attrList.Delete()
	if err := attrList.Update(
		windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE,
		unsafe.Pointer(console),
		unsafe.Sizeof(console),
	); err != nil {
		windows.ClosePseudoConsole(console)
		logFile.Close()
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		return err
	}

	var si windows.StartupInfoEx
	si.Cb = uint32(unsafe.Sizeof(si))
	si.Flags = windows.STARTF_USESTDHANDLES
	si.ProcThreadAttributeList = attrList.List()

	appPtr, err := windows.UTF16PtrFromString(appName)
	if err != nil {
		windows.ClosePseudoConsole(console)
		logFile.Close()
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		return err
	}
	cmdLineBuf, err := windows.UTF16FromString(cmdLine)
	if err != nil {
		windows.ClosePseudoConsole(console)
		logFile.Close()
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		return err
	}
	workDirPtr, err := windows.UTF16PtrFromString(workDir)
	if err != nil {
		windows.ClosePseudoConsole(console)
		logFile.Close()
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		return err
	}

	var pi windows.ProcessInformation
	err = windows.CreateProcess(
		appPtr,
		&cmdLineBuf[0],
		nil,
		nil,
		false,
		windows.EXTENDED_STARTUPINFO_PRESENT|windows.CREATE_UNICODE_ENVIRONMENT,
		nil,
		workDirPtr,
		&si.StartupInfo,
		&pi,
	)
	if err != nil {
		windows.ClosePseudoConsole(console)
		logFile.Close()
		windows.CloseHandle(inWrite)
		windows.CloseHandle(outRead)
		return fmt.Errorf("CreateProcess: %w", err)
	}

	rt := &runtimeSession{
		record: sessionRecord{
			Name:      name,
			ProcessID: pi.ProcessId,
			WorkDir:   workDir,
			LogPath:   logPath,
			Program:   program,
		},
		console:      console,
		process:      pi.Process,
		thread:       pi.Thread,
		inputWriter:  os.NewFile(uintptr(inWrite), name+"-stdin"),
		outputReader: os.NewFile(uintptr(outRead), name+"-stdout"),
		logFile:      logFile,
		done:         make(chan struct{}),
	}

	m.mu.Lock()
	m.sessions[name] = rt
	if err := m.writeRegistryLocked(upsertRecord(m.readRegistryLocked(), rt.record)); err != nil {
		delete(m.sessions, name)
		m.mu.Unlock()
		rt.close()
		return err
	}
	m.mu.Unlock()

	go m.captureOutput(name, rt)
	go m.waitForExit(name, rt)

	return nil
}

func (m *Manager) Exists(ctx context.Context, name string) bool {
	_ = ctx
	m.mu.Lock()
	if rt, ok := m.sessions[name]; ok && rt != nil {
		running := processRunning(rt.record.ProcessID)
		m.mu.Unlock()
		return running
	}
	records := m.readRegistryLocked()
	record, ok := records[name]
	m.mu.Unlock()
	if !ok {
		return false
	}
	if !processRunning(record.ProcessID) {
		m.mu.Lock()
		records = m.readRegistryLocked()
		delete(records, name)
		_ = m.writeRegistryLocked(records)
		m.mu.Unlock()
		return false
	}
	return true
}

func (m *Manager) Kill(ctx context.Context, name string) error {
	_ = ctx
	m.mu.Lock()
	rt, ok := m.sessions[name]
	if ok {
		delete(m.sessions, name)
		records := m.readRegistryLocked()
		delete(records, name)
		_ = m.writeRegistryLocked(records)
		m.mu.Unlock()
		rt.close()
		return terminatePID(rt.record.ProcessID)
	}
	records := m.readRegistryLocked()
	record, ok := records[name]
	if ok {
		delete(records, name)
		_ = m.writeRegistryLocked(records)
	}
	m.mu.Unlock()
	if !ok {
		return nil
	}
	return terminatePID(record.ProcessID)
}

func (m *Manager) ListSessions(ctx context.Context) ([]string, error) {
	_ = ctx
	m.mu.Lock()
	records := m.readRegistryLocked()
	var sessions []string
	changed := false
	for name, record := range records {
		if processRunning(record.ProcessID) {
			sessions = append(sessions, name)
			continue
		}
		delete(records, name)
		changed = true
	}
	if changed {
		_ = m.writeRegistryLocked(records)
	}
	m.mu.Unlock()
	return sessions, nil
}

func (m *Manager) CapturePane(ctx context.Context, name string) (string, error) {
	_ = ctx
	data, err := m.readLog(name)
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", nil
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) > maxVisibleLines {
		lines = lines[len(lines)-maxVisibleLines:]
	}
	return strings.Join(lines, "\n"), nil
}

func (m *Manager) CaptureScrollback(ctx context.Context, name string) (string, error) {
	_ = ctx
	data, err := m.readLog(name)
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n"), nil
}

func (m *Manager) SendKeys(ctx context.Context, name string, keys ...string) error {
	_ = ctx
	m.mu.Lock()
	rt, ok := m.sessions[name]
	m.mu.Unlock()
	if !ok || rt == nil || rt.inputWriter == nil {
		return fmt.Errorf("session input unavailable: %s", name)
	}

	var buf bytes.Buffer
	for _, key := range keys {
		buf.WriteString(translateKey(key))
	}
	_, err := rt.inputWriter.Write(buf.Bytes())
	return err
}

func (m *Manager) SendText(ctx context.Context, name string, text string) error {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		if err := m.SendKeys(ctx, name, line, "Enter"); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) SendCommand(ctx context.Context, name, cmd string) error {
	return m.SendKeys(ctx, name, cmd, "Enter")
}

// BackendMetadata exposes runtime metadata for persistence.
func (m *Manager) BackendMetadata(name string) (kind string, processID int, logPath string, worktreePath string, ok bool) {
	m.mu.Lock()
	if rt, exists := m.sessions[name]; exists && rt != nil {
		m.mu.Unlock()
		return "windowspty", int(rt.record.ProcessID), rt.record.LogPath, rt.record.WorkDir, true
	}
	records := m.readRegistryLocked()
	record, exists := records[name]
	m.mu.Unlock()
	if !exists {
		return "", 0, "", "", false
	}
	return "windowspty", int(record.ProcessID), record.LogPath, record.WorkDir, true
}

func (m *Manager) readLog(name string) ([]byte, error) {
	m.mu.Lock()
	records := m.readRegistryLocked()
	record, ok := records[name]
	m.mu.Unlock()
	if !ok {
		return nil, os.ErrNotExist
	}
	return os.ReadFile(record.LogPath)
}

func (m *Manager) captureOutput(name string, rt *runtimeSession) {
	if rt.outputReader == nil || rt.logFile == nil {
		return
	}
	_, _ = io.Copy(rt.logFile, rt.outputReader)
}

func (m *Manager) waitForExit(name string, rt *runtimeSession) {
	defer close(rt.done)
	_, _ = windows.WaitForSingleObject(rt.process, windows.INFINITE)

	m.mu.Lock()
	current, ok := m.sessions[name]
	if ok && current == rt {
		delete(m.sessions, name)
		records := m.readRegistryLocked()
		delete(records, name)
		_ = m.writeRegistryLocked(records)
	}
	m.mu.Unlock()
	rt.close()
}

func (m *Manager) readRegistryLocked() map[string]sessionRecord {
	data, err := os.ReadFile(m.registry)
	if err != nil {
		return map[string]sessionRecord{}
	}
	var records map[string]sessionRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return map[string]sessionRecord{}
	}
	if records == nil {
		return map[string]sessionRecord{}
	}
	return records
}

func (m *Manager) writeRegistryLocked(records map[string]sessionRecord) error {
	if err := os.MkdirAll(filepath.Dir(m.registry), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.registry, data, 0o644)
}

func (rt *runtimeSession) close() {
	if rt.inputWriter != nil {
		_ = rt.inputWriter.Close()
	}
	if rt.outputReader != nil {
		_ = rt.outputReader.Close()
	}
	if rt.logFile != nil {
		_ = rt.logFile.Close()
	}
	if rt.thread != 0 {
		_ = windows.CloseHandle(rt.thread)
	}
	if rt.process != 0 {
		_ = windows.CloseHandle(rt.process)
	}
	if rt.console != 0 {
		windows.ClosePseudoConsole(rt.console)
	}
}

func resolveProgram(program string) (string, string, error) {
	args, err := windows.DecomposeCommandLine(program)
	if err != nil || len(args) == 0 {
		args = []string{program}
	}
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return "", "", errors.New("program is required")
	}
	appName, err := exec.LookPath(args[0])
	if err != nil {
		return "", "", err
	}
	args[0] = appName
	return appName, windows.ComposeCommandLine(args), nil
}

func newPipePair() (windows.Handle, windows.Handle, error) {
	var readHandle, writeHandle windows.Handle
	if err := windows.CreatePipe(&readHandle, &writeHandle, nil, 0); err != nil {
		return 0, 0, err
	}
	return readHandle, writeHandle, nil
}

func defaultRegistryPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cavekit", registryFileName)
}

func sanitizeFileName(name string) string {
	replacer := strings.NewReplacer("\\", "_", "/", "_", ":", "_", " ", "_")
	return replacer.Replace(name)
}

func translateKey(key string) string {
	switch key {
	case "Enter":
		return "\r"
	case "BSpace":
		return "\b"
	case "Tab":
		return "\t"
	case "Up":
		return "\x1b[A"
	case "Down":
		return "\x1b[B"
	case "Left":
		return "\x1b[D"
	case "Right":
		return "\x1b[C"
	case "Space":
		return " "
	case "C-c":
		return "\x03"
	case "C-d":
		return "\x04"
	default:
		return key
	}
}

func upsertRecord(records map[string]sessionRecord, record sessionRecord) map[string]sessionRecord {
	if records == nil {
		records = map[string]sessionRecord{}
	}
	records[record.Name] = record
	return records
}

func terminatePID(pid uint32) error {
	if pid == 0 {
		return nil
	}
	process, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.SYNCHRONIZE, false, pid)
	if err != nil {
		return nil
	}
	defer windows.CloseHandle(process)
	if err := windows.TerminateProcess(process, 1); err != nil {
		return err
	}
	_, _ = windows.WaitForSingleObject(process, windows.INFINITE)
	return nil
}

func processRunning(pid uint32) bool {
	if pid == 0 {
		return false
	}
	process, err := windows.OpenProcess(windows.SYNCHRONIZE, false, pid)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(process)
	status, err := windows.WaitForSingleObject(process, 0)
	return err == nil && status == uint32(windows.WAIT_TIMEOUT)
}
