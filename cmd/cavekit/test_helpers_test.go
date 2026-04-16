package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func stringsContains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func stringsHasPrefix(s, prefix string) bool {
	return strings.HasPrefix(s, prefix)
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = original
	}()

	fn()

	_ = writer.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return buf.String()
}
