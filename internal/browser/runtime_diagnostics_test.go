package browser

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecentChromiumRuntimeDiagnosticClassifiesWindowsCommitLimit(t *testing.T) {
	runtimeDir := t.TempDir()
	executablePath := filepath.Join(runtimeDir, "chrome.exe")
	launchedAt := time.Now().Add(-time.Second)
	debugLog := "[1113/113714.642:ERROR:chrome\\app\\main_dll_loader_win.cc:141] Failed to load Chrome DLL: 分頁檔太小，無法完成操作。 (0x5AF)\n"
	if err := os.WriteFile(filepath.Join(runtimeDir, "debug.log"), []byte(debugLog), 0o600); err != nil {
		t.Fatal(err)
	}

	diagnostic, ok := recentChromiumRuntimeDiagnostic(executablePath, launchedAt)
	if !ok {
		t.Fatal("expected a runtime diagnostic")
	}
	if diagnostic.Code != "WINDOWS_COMMIT_LIMIT" {
		t.Fatalf("unexpected code %q", diagnostic.Code)
	}
}

func TestRecentChromiumRuntimeDiagnosticIgnoresStaleLog(t *testing.T) {
	runtimeDir := t.TempDir()
	debugPath := filepath.Join(runtimeDir, "debug.log")
	if err := os.WriteFile(debugPath, []byte("Failed to load Chrome DLL (0x5AF)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	staleAt := time.Now().Add(-time.Hour)
	if err := os.Chtimes(debugPath, staleAt, staleAt); err != nil {
		t.Fatal(err)
	}

	if _, ok := recentChromiumRuntimeDiagnostic(filepath.Join(runtimeDir, "chrome.exe"), time.Now()); ok {
		t.Fatal("stale runtime log must not be attributed to a new launch")
	}
}

func TestRecentChromiumRuntimeDiagnosticIgnoresOldLogFromCurrentLaunch(t *testing.T) {
	runtimeDir := t.TempDir()
	debugPath := filepath.Join(runtimeDir, "debug.log")
	if err := os.WriteFile(debugPath, []byte("Failed to load Chrome DLL (0x5AF)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldAt := time.Now().Add(-time.Minute)
	if err := os.Chtimes(debugPath, oldAt, oldAt); err != nil {
		t.Fatal(err)
	}

	if _, ok := recentChromiumRuntimeDiagnostic(filepath.Join(runtimeDir, "chrome.exe"), time.Now().Add(-time.Hour)); ok {
		t.Fatal("old runtime log must not be attributed to a later disconnect")
	}
}
