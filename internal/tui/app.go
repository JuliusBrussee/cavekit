// Package tui implements the bubbletea-based terminal user interface.
package tui

import (
	"context"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/JuliusBrussee/cavekit/internal/backend"
	"github.com/JuliusBrussee/cavekit/internal/exec"
	"github.com/JuliusBrussee/cavekit/internal/session"
	"github.com/JuliusBrussee/cavekit/internal/site"
	"github.com/JuliusBrussee/cavekit/internal/worktree"
)

// Tab represents the active content tab.
type Tab int

const (
	TabPreview Tab = iota
	TabDiff
	TabTerminal
)

func (t Tab) String() string {
	switch t {
	case TabPreview:
		return "Preview"
	case TabDiff:
		return "Diff"
	case TabTerminal:
		return "Terminal"
	default:
		return "Unknown"
	}
}

// tickMsg triggers periodic updates (metadata, capture).
type tickMsg time.Time

// animTickMsg triggers visual-only updates (spinner, toast fade).
type animTickMsg time.Time

// instanceCreatedMsg is sent when a new instance has been started in the background.
type instanceCreatedMsg struct {
	inst *session.Instance
	err  error
}

// attachFinishedMsg is sent when external attach returns.
type attachFinishedMsg struct{ err error }

// App is the main bubbletea model.
type App struct {
	width  int
	height int

	activeTab     Tab
	selectedIndex int

	// Components
	header       *Header
	statusLine   *StatusLine
	instanceList *InstanceList
	tabContent   *TabContent
	bottomMenu   *BottomMenu
	overlay      *Overlay
	toasts       *ToastManager
	dashboard    *Dashboard

	// Tab data sources
	previewTab  *PreviewTab
	diffTab     *DiffTab
	terminalTab *TerminalTab

	// Site picker for new-instance flow
	sitePicker *SitePicker

	// Pending confirmation action (to distinguish kill vs push)
	pendingAction KeyAction

	// Data
	instances []*session.Instance

	// Session management
	sessionMgr     *session.Manager
	store          *session.Store
	autoYes        *session.AutoYes
	statusDetector *backend.StatusDetector
	sessionBackend backend.SessionBackend
	caps           backend.Capabilities
	projectRoot    string
	program        string

	// Input mode: keystrokes forwarded to the selected runtime session.
	inputMode bool

	// Tick counter for staggering
	tickCount int

	// Set to true when we need to quit
	quitting bool
}

// NewApp creates a new TUI application model.
func NewApp(projectRoot, program string, autoYesEnabled bool) App {
	executor := exec.NewRealExecutor()
	sessionBackend, caps := backend.New(executor)
	wtMgr := worktree.NewManager(executor)
	sessMgr := session.NewManager(sessionBackend, wtMgr)
	store := session.NewStore("")

	return App{
		activeTab:    TabPreview,
		header:       NewHeader(projectRoot),
		statusLine:   NewStatusLine(),
		instanceList: NewInstanceList(),
		tabContent:   NewTabContent(),
		bottomMenu:   NewBottomMenu(),
		overlay:      NewOverlay(),
		toasts:       NewToastManager(),
		dashboard:    NewDashboard(),

		// Tab data sources
		previewTab:  NewPreviewTab(sessionBackend),
		diffTab:     NewDiffTab(wtMgr),
		terminalTab: NewTerminalTab(sessionBackend, caps),

		// Site picker
		sitePicker: NewSitePicker(),

		// Session management
		sessionMgr:     sessMgr,
		store:          store,
		autoYes:        session.NewAutoYes(sessionBackend, autoYesEnabled),
		statusDetector: backend.NewStatusDetector(sessionBackend),
		sessionBackend: sessionBackend,
		caps:           caps,
		projectRoot:    projectRoot,
		program:        program,
	}
}

// Init implements tea.Model.
func (a App) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		animTickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func animTickCmd() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg {
		return animTickMsg(t)
	})
}

// Update implements tea.Model.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		action := MapKey(key, a.overlay.IsActive(), a.overlay.Active, a.inputMode)

		switch action {
		case ActionForwardKey:
			if sel := a.instanceList.Selected(); sel != nil && sel.SessionName != "" {
				a.forwardKey(key, msg)
			}
			return a, nil
		case ActionEnterInput:
			if sel := a.instanceList.Selected(); sel != nil && sel.SessionName != "" {
				a.inputMode = true
			}
		case ActionExitInput:
			a.inputMode = false
		case ActionQuit:
			a.quitting = true
			a.saveState()
			return a, tea.Quit
		case ActionSwitchTab:
			a.activeTab = (a.activeTab + 1) % 3
			a.tabContent.SetActiveTab(a.activeTab)
		case ActionNavigateDown:
			if a.overlay.Active == OverlaySitePicker {
				a.sitePicker.MoveDown()
			} else {
				a.selectedIndex++
				a.instanceList.SetSelected(a.selectedIndex)
				a.selectedIndex = a.instanceList.SelectedIndex()
			}
		case ActionNavigateUp:
			if a.overlay.Active == OverlaySitePicker {
				a.sitePicker.MoveUp()
			} else {
				if a.selectedIndex > 0 {
					a.selectedIndex--
				}
				a.instanceList.SetSelected(a.selectedIndex)
				a.selectedIndex = a.instanceList.SelectedIndex()
			}
		case ActionHelp:
			if a.overlay.Active == OverlayHelp {
				a.overlay.Hide()
			} else {
				a.overlay.Show(OverlayHelp, "", "")
			}
		case ActionToggleSelect:
			if a.overlay.Active == OverlaySitePicker {
				a.sitePicker.ToggleSelect()
			}
		case ActionCancel:
			a.overlay.Hide()
			a.sitePicker.Hide()
		case ActionNew:
			items := a.discoverSiteItems()
			if len(items) > 0 {
				a.sitePicker.SetItems(items)
				a.sitePicker.Show()
				a.overlay.Show(OverlaySitePicker, "Select Site", "")
			} else {
				a.overlay.Show(OverlayTextInput, "New Instance", "Enter site name:")
			}
		case ActionKill:
			if sel := a.instanceList.Selected(); sel != nil {
				a.pendingAction = ActionKill
				a.overlay.Show(OverlayConfirmation, "Kill Instance", "Kill '"+sel.Title+"'?")
			}
		case ActionTextInput:
			if a.overlay.Active == OverlayTextInput {
				r := msg.Runes
				if len(r) > 0 {
					a.overlay.InputValue += string(r)
				}
			}
		case ActionBackspace:
			if a.overlay.Active == OverlayTextInput && len(a.overlay.InputValue) > 0 {
				a.overlay.InputValue = a.overlay.InputValue[:len(a.overlay.InputValue)-1]
			}
		case ActionConfirmYes:
			switch a.overlay.Active {
			case OverlaySitePicker:
				selected := a.sitePicker.SelectedItems()
				a.overlay.Hide()
				a.sitePicker.Hide()
				if len(selected) > 0 {
					return a, a.launchSites(selected)
				}
			case OverlayTextInput:
				name := a.overlay.InputValue
				if name != "" && len(name) <= 32 {
					a.overlay.Hide()
					return a, a.createInstanceCmd(name)
				}
			case OverlayConfirmation:
				if sel := a.instanceList.Selected(); sel != nil {
					a.overlay.Hide()
					switch a.pendingAction {
					case ActionKill:
						a.sessionMgr.Kill(context.Background(), sel, a.projectRoot, true)
						a.removeInstance(sel)
						a.toasts.Add("Instance killed", ToastInfo)
					case ActionPush:
						wtMgr := worktree.NewManager(exec.NewRealExecutor())
						wtMgr.Push(context.Background(), sel.WorktreePath, "Cavekit: push from monitor")
						a.toasts.Add("Branch pushed", ToastSuccess)
					}
					a.pendingAction = ActionNone
				}
			}
		case ActionConfirmNo:
			a.overlay.Hide()
		case ActionOpen:
			if sel := a.instanceList.Selected(); sel != nil && sel.SessionName != "" {
				if a.caps.SupportsAttach {
					return a, a.attachCmd(sel.SessionName)
				}
				a.terminalTab.EnsureSession(context.Background(), sel.Title, sel.WorktreePath, sel.SessionName)
				a.activeTab = TabTerminal
				a.tabContent.SetActiveTab(TabTerminal)
			}
		case ActionPush:
			if sel := a.instanceList.Selected(); sel != nil && sel.WorktreePath != "" {
				a.pendingAction = ActionPush
				a.overlay.Show(OverlayConfirmation, "Push Branch", "Push '"+sel.Title+"' branch to remote?")
			}
		case ActionCheckout:
			if sel := a.instanceList.Selected(); sel != nil && sel.WorktreePath != "" {
				a.terminalTab.EnsureSession(context.Background(), sel.Title, sel.WorktreePath, sel.SessionName)
				a.activeTab = TabTerminal
				a.tabContent.SetActiveTab(TabTerminal)
			}
		case ActionResume:
			if sel := a.instanceList.Selected(); sel != nil && sel.Status == session.StatusPaused {
				a.sessionMgr.Resume(context.Background(), sel)
				a.instanceList.SetInstances(a.instances)
			}
		case ActionScrollUp:
			a.diffTab.ScrollUp(3)
			a.tabContent.SetDiff(a.diffTab.Content())
		case ActionScrollDown:
			a.diffTab.ScrollDown(3)
			a.tabContent.SetDiff(a.diffTab.Content())
		case ActionNextFile:
			a.diffTab.NextFile()
			a.tabContent.SetDiff(a.diffTab.Content())
		case ActionPrevFile:
			a.diffTab.PrevFile()
			a.tabContent.SetDiff(a.diffTab.Content())
		}

	case tea.MouseMsg:
		if !a.overlay.IsActive() {
			leftWidth := max(int(float64(a.width)*LeftPanelRatio), MinLeftWidth)
			if msg.X < leftWidth {
				row := msg.Y - a.header.Height() - 1
				if row >= 0 {
					// Account for 2-line rows
					idx := a.instanceList.scrollOffset + row/2
					if idx >= 0 && idx < len(a.instances) {
						a.selectedIndex = idx
						a.instanceList.SetSelected(a.selectedIndex)
						a.selectedIndex = a.instanceList.SelectedIndex()
					}
				}
			}
		}

	case attachFinishedMsg:
		// TUI resumes after tmux detach

	case instanceCreatedMsg:
		if msg.err == nil && msg.inst != nil {
			a.instances = append(a.instances, msg.inst)
			a.instanceList.SetInstances(a.instances)
			a.selectedIndex = len(a.instances) - 1
			a.instanceList.SetSelected(a.selectedIndex)
			a.saveState()
			a.toasts.Add("Instance created", ToastSuccess)
		} else if msg.err != nil {
			a.toasts.Add("Error: "+msg.err.Error(), ToastError)
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.updateLayout()

	case tickMsg:
		a.onTick()
		return a, tickCmd()

	case animTickMsg:
		a.toasts.Tick()
		return a, animTickCmd()
	}

	return a, nil
}

// onTick performs all periodic updates on each 500ms tick.
func (a *App) onTick() {
	ctx := context.Background()
	a.tickCount++

	wtMgr := worktree.NewManager(exec.NewRealExecutor())

	// Update progress and diff stats for all active instances
	for i, inst := range a.instances {
		if inst.IsActive() {
			session.UpdateProgress(inst)

			// Diff stats: selected instance every tick, others every 5th tick
			if inst.WorktreePath != "" {
				isSelected := i == a.selectedIndex
				if isSelected || a.tickCount%5 == 0 {
					stats, err := wtMgr.DiffStat(ctx, inst.WorktreePath)
					if err == nil {
						inst.DiffAdded = stats.Insertions
						inst.DiffRemoved = stats.Deletions
						inst.BranchName = worktree.BranchName(inst.Title)
					}
				}
			}

			// Update instance status from runtime status detection
			if inst.SessionName != "" {
				paneStatus, err := a.statusDetector.Detect(ctx, inst.SessionName)
				if err == nil {
					switch paneStatus {
					case backend.PaneActive:
						inst.Status = session.StatusRunning
					case backend.PaneIdle, backend.PanePrompt:
						inst.Status = session.StatusReady
					}
				}

				a.autoYes.Check(ctx, inst.SessionName)
			}
		}
	}

	// Update instance list display
	a.instanceList.SetInstances(a.instances)

	// Update header stats
	var running, done, tasksDone, tasksTotal int
	for _, inst := range a.instances {
		if inst.Status == session.StatusRunning {
			running++
		}
		if inst.Status == session.StatusDone {
			done++
		}
		tasksDone += inst.TasksDone
		tasksTotal += inst.TasksTotal
	}
	a.header.SetStats(len(a.instances), running, done, tasksDone, tasksTotal)

	// Update status line
	a.statusLine.SetInstance(a.instanceList.Selected())

	// Update dashboard
	a.dashboard.SetInstances(a.instances)

	// Update menu based on context
	if a.inputMode {
		a.bottomMenu.SetItems(InputModeMenu())
	} else if a.overlay.IsActive() {
		a.bottomMenu.SetItems(OverlayMenu(a.overlay.Active))
	} else if a.instanceList.Selected() == nil {
		a.bottomMenu.SetItems(NoSelectionMenu())
	} else {
		a.bottomMenu.SetItems(DefaultMenu())
	}

	// Update tab content for selected instance
	sel := a.instanceList.Selected()
	if sel != nil {
		// Update diff badge
		a.tabContent.SetDiffStats(a.diffTab.Stats())

		switch a.activeTab {
		case TabPreview:
			a.previewTab.Capture(ctx, sel.SessionName)
			a.tabContent.SetPreview(a.previewTab.Content())
		case TabDiff:
			a.diffTab.Refresh(ctx, sel.WorktreePath)
			a.tabContent.SetDiff(a.diffTab.Content())
		case TabTerminal:
			a.terminalTab.Capture(ctx, sel.Title, sel.SessionName)
			a.tabContent.SetTerminal(a.terminalTab.Content())
		}
	} else {
		a.tabContent.SetPreview("")
		a.tabContent.SetDiff("")
		a.tabContent.SetTerminal("")
		a.tabContent.SetDiffStats("")
	}
}

func (a *App) updateLayout() {
	leftWidth := max(int(float64(a.width)*LeftPanelRatio), MinLeftWidth)
	rightWidth := a.width - leftWidth - 2
	contentHeight := a.height - MenuHeight - a.header.Height() - a.statusLine.Height() - 2

	a.header.SetWidth(a.width)
	a.statusLine.SetWidth(a.width)
	a.instanceList.SetSize(leftWidth-2, contentHeight-2)
	a.tabContent.SetSize(rightWidth-2, contentHeight-2)
	a.bottomMenu.SetWidth(a.width)
	a.overlay.SetSize(a.width, a.height)
	a.dashboard.SetSize(rightWidth-2, contentHeight-2)
	a.toasts.SetSize(a.width, a.height)
}

// View implements tea.Model.
func (a App) View() string {
	if a.quitting {
		return ""
	}
	if a.width == 0 || a.height == 0 {
		return "Initializing..."
	}

	leftWidth := max(int(float64(a.width)*LeftPanelRatio), MinLeftWidth)
	rightWidth := a.width - leftWidth - 2
	contentHeight := a.height - MenuHeight - a.header.Height() - a.statusLine.Height() - 2

	// Header
	header := a.header.View()

	// Render panels
	left := LeftPanelStyle.
		Width(leftWidth).
		Height(contentHeight).
		Render(a.instanceList.View())

	// Right panel: dashboard when no selection, tab content otherwise
	var rightContent string
	if a.instanceList.Selected() == nil {
		rightContent = a.dashboard.View()
	} else {
		rightContent = a.tabContent.View()
	}

	right := RightPanelStyle.
		Width(rightWidth).
		Height(contentHeight).
		Render(rightContent)

	statusLine := a.statusLine.View()
	menu := a.bottomMenu.View()

	// Compose layout
	panels := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	base := lipgloss.JoinVertical(lipgloss.Left, header, panels, statusLine, menu)

	// Overlay on top with dimmed background
	if a.overlay.IsActive() {
		dimmed := DimView(base)
		if a.overlay.Active == OverlaySitePicker {
			pickerContent := a.sitePicker.View()
			overlayWidth := min(60, a.width-4)
			rendered := OverlayStyle.Width(overlayWidth).Render(pickerContent)
			return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, rendered,
				lipgloss.WithWhitespaceChars(" "),
				lipgloss.WithWhitespaceForeground(lipgloss.NoColor{}),
			)
		}
		_ = dimmed
		return a.overlay.View()
	}

	// Toast overlay
	if a.toasts.IsActive() {
		return a.toasts.Overlay(base, a.width, a.height)
	}

	return base
}

// SetInstances updates the displayed instances.
func (a *App) SetInstances(instances []*session.Instance) {
	a.instances = instances
	a.instanceList.SetInstances(instances)
}

// attachCmd suspends the TUI and attaches to a backend session when supported.
func (a *App) attachCmd(sessionName string) tea.Cmd {
	c, err := backend.AttachCommand(sessionName)
	if err != nil {
		return func() tea.Msg {
			return attachFinishedMsg{err: err}
		}
	}
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return attachFinishedMsg{err: err}
	})
}

func (a *App) createInstanceCmd(name string) tea.Cmd {
	return func() tea.Msg {
		prog := a.program
		if prog == "" {
			prog = "claude"
		}
		inst := a.sessionMgr.Create(name, "", name, prog)
		err := a.sessionMgr.Start(context.Background(), inst, a.projectRoot, name, 3*time.Second)
		return instanceCreatedMsg{inst: inst, err: err}
	}
}

func (a *App) removeInstance(inst *session.Instance) {
	for i, ins := range a.instances {
		if ins == inst {
			a.instances = append(a.instances[:i], a.instances[i+1:]...)
			break
		}
	}
	a.instanceList.SetInstances(a.instances)
	if a.selectedIndex >= len(a.instances) && a.selectedIndex > 0 {
		a.selectedIndex = len(a.instances) - 1
	}
	a.instanceList.SetSelected(a.selectedIndex)
	a.saveState()
}

// discoverSiteItems scans the project for available sites.
func (a *App) discoverSiteItems() []SitePickerItem {
	sites, err := site.Discover(a.projectRoot)
	if err != nil {
		return nil
	}

	var items []SitePickerItem
	for _, ss := range sites {
		item := SitePickerItem{
			Name:   ss.Name,
			Path:   ss.Path,
			Status: site.SiteAvailable,
		}

		f, err := site.Parse(ss.Path)
		if err == nil {
			implDir := filepath.Join(a.projectRoot, "context", "impl")
			statuses, err := site.TrackStatus(implDir)
			if err == nil {
				summary := site.ComputeProgress(f, statuses)
				item.TasksDone = summary.Done
				item.TasksTotal = summary.Total

				wtPath := worktree.WorktreePath(a.projectRoot, ss.Name)
				item.Status = site.ClassifySite(f, statuses, wtPath)
			}
		}

		items = append(items, item)
	}
	return items
}

// launchSites creates and starts instances for the selected sites.
func (a *App) launchSites(items []SitePickerItem) tea.Cmd {
	return func() tea.Msg {
		prog := a.program
		if prog == "" {
			prog = "claude"
		}
		item := items[0]
		inst := a.sessionMgr.Create(item.Name, item.Path, item.Name, prog)
		err := a.sessionMgr.Start(context.Background(), inst, a.projectRoot, item.Name, 3*time.Second)
		return instanceCreatedMsg{inst: inst, err: err}
	}
}

// forwardKey sends a keystroke to the selected instance's runtime session.
func (a *App) forwardKey(key string, msg tea.KeyMsg) {
	sel := a.instanceList.Selected()
	if sel == nil || sel.SessionName == "" {
		return
	}
	ctx := context.Background()

	switch key {
	case "enter":
		_ = a.sessionBackend.SendKeys(ctx, sel.SessionName, "Enter")
	case "backspace":
		_ = a.sessionBackend.SendKeys(ctx, sel.SessionName, "BSpace")
	case "tab":
		_ = a.sessionBackend.SendKeys(ctx, sel.SessionName, "Tab")
	case "up":
		_ = a.sessionBackend.SendKeys(ctx, sel.SessionName, "Up")
	case "down":
		_ = a.sessionBackend.SendKeys(ctx, sel.SessionName, "Down")
	case "left":
		_ = a.sessionBackend.SendKeys(ctx, sel.SessionName, "Left")
	case "right":
		_ = a.sessionBackend.SendKeys(ctx, sel.SessionName, "Right")
	case " ":
		_ = a.sessionBackend.SendKeys(ctx, sel.SessionName, "Space")
	case "ctrl+c":
		_ = a.sessionBackend.SendKeys(ctx, sel.SessionName, "C-c")
	case "ctrl+d":
		_ = a.sessionBackend.SendKeys(ctx, sel.SessionName, "C-d")
	default:
		if len(key) == 1 {
			_ = a.sessionBackend.SendKeys(ctx, sel.SessionName, key)
		}
	}
}

func (a *App) saveState() {
	if a.store != nil {
		a.store.Save(a.instances)
	}
}

// Run starts the TUI application.
func Run(projectRoot, program string, autoYes bool) error {
	app := NewApp(projectRoot, program, autoYes)

	// Load persisted instances and validate
	instances, _ := app.store.Load()
	if len(instances) > 0 {
		ctx := context.Background()
		for _, inst := range instances {
			if inst.SessionName != "" && !app.sessionBackend.Exists(ctx, inst.SessionName) {
				inst.Status = session.StatusDone
			}
		}
		var active []*session.Instance
		for _, inst := range instances {
			if inst.Status != session.StatusDone {
				active = append(active, inst)
			}
		}
		app.instances = active
		app.instanceList.SetInstances(active)
	}

	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}
