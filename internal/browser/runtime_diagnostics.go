package browser

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxRuntimeDebugBytes       int64 = 64 * 1024
	runtimeDiagnosticFreshness       = 15 * time.Second
)

type runtimeDiagnostic struct {
	Code    string
	Message string
}

func recentChromiumRuntimeDiagnostic(executablePath string, launchedAt time.Time) (runtimeDiagnostic, bool) {
	if executablePath == "" {
		return runtimeDiagnostic{}, false
	}

	debugPath := filepath.Join(filepath.Dir(executablePath), "debug.log")
	info, err := os.Stat(debugPath)
	if err != nil ||
		info.ModTime().Before(launchedAt) ||
		time.Since(info.ModTime()) > runtimeDiagnosticFreshness {
		return runtimeDiagnostic{}, false
	}

	file, err := os.Open(debugPath)
	if err != nil {
		return runtimeDiagnostic{}, false
	}
	defer file.Close()

	start := info.Size() - maxRuntimeDebugBytes
	if start < 0 {
		start = 0
	}
	if _, err := file.Seek(start, io.SeekStart); err != nil {
		return runtimeDiagnostic{}, false
	}
	data, err := io.ReadAll(io.LimitReader(file, maxRuntimeDebugBytes))
	if err != nil {
		return runtimeDiagnostic{}, false
	}

	lines := strings.Split(string(data), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		lower := strings.ToLower(line)
		if strings.Contains(lower, "0x5af") ||
			strings.Contains(lower, "paging file is too small") ||
			strings.Contains(line, "分頁檔太小") {
			return runtimeDiagnostic{
				Code:    "WINDOWS_COMMIT_LIMIT",
				Message: "CloakBrowser failed to load Chrome DLL because the Windows paging file was too small (0x5AF)",
			}, true
		}
		if strings.Contains(lower, "failed to load chrome dll") {
			return runtimeDiagnostic{
				Code:    "CHROME_DLL_LOAD_FAILED",
				Message: "CloakBrowser failed to load Chrome DLL; inspect the runtime debug log",
			}, true
		}
	}

	return runtimeDiagnostic{}, false
}
