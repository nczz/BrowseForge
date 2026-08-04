package browser

import (
	"browseforge/internal/config"
	"browseforge/internal/profile"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShouldRetryLaunchForProtocolEOF(t *testing.T) {
	cases := []string{
		"target closed: could not read protocol padding: EOF",
		"launch firefox: target closed",
		"unexpected EOF",
		"FATAL:content\\browser\\gpu\\gpu_data_manager_impl_private.cc:417] GPU process isn't usable. Goodbye.",
		"ERROR:net\\disk_cache\\disk_cache.cc:284] Unable to create cache",
	}

	for _, msg := range cases {
		if !shouldRetryLaunch(errors.New(msg)) {
			t.Fatalf("expected retryable launch error for %q", msg)
		}
	}
}

func TestShouldRetryLaunchRejectsRegularErrors(t *testing.T) {
	if shouldRetryLaunch(errors.New("profile appears to be in use")) {
		t.Fatal("profile lock errors should not restart Playwright")
	}
}

func TestShouldRetryLaunchRejectsNoManagerRetryError(t *testing.T) {
	err := noManagerRetryError{err: errors.New("target closed: GPU process isn't usable")}
	if shouldRetryLaunch(err) {
		t.Fatal("fallback-exhausted errors should not be retried by manager")
	}
}

func TestPrepareSessionEndpointRequiresHealthyEndpoint(t *testing.T) {
	wantEndpoint := "ws://127.0.0.1:12345/bind"
	m := &Manager{
		cfg:                     &config.Config{Host: "127.0.0.1"},
		endpointHealthTimeoutMS: 1234,
		bindSessionEndpoint: func(s *Session) (string, error) {
			if s.ID != "sess_prof_health" {
				t.Fatalf("bind session = %s, want sess_prof_health", s.ID)
			}
			return wantEndpoint, nil
		},
		endpointHealthCheck: func(endpoint string, timeoutMS float64) error {
			if endpoint != wantEndpoint {
				t.Fatalf("endpoint = %s, want %s", endpoint, wantEndpoint)
			}
			if timeoutMS != 1234 {
				t.Fatalf("timeoutMS = %v, want 1234", timeoutMS)
			}
			return errors.New("browserType.connect: Timeout 30000ms exceeded")
		},
	}
	p := &profile.Profile{ID: "prof_health", Engine: "chromium"}
	s := &Session{ID: "sess_prof_health", ProfileID: p.ID, Engine: p.Engine, ProfileDir: "profiles/prof_health", UserDataDir: "profiles/prof_health/browser-data", ExecutablePath: "browsers/cloakbrowser/chrome.exe"}

	err := m.prepareSessionEndpoint(p, s, 1)
	if err == nil {
		t.Fatal("prepareSessionEndpoint succeeded for unhealthy endpoint")
	}
	if got := ErrorCode(err); got != "BROWSER_CONNECT_TIMEOUT" {
		t.Fatalf("ErrorCode = %q, want BROWSER_CONNECT_TIMEOUT; err=%v", got, err)
	}
	if s.ConnectURL != wantEndpoint {
		t.Fatalf("ConnectURL = %q, want %q for diagnostics", s.ConnectURL, wantEndpoint)
	}
}

func TestPrepareSessionEndpointStoresHealthyEndpoint(t *testing.T) {
	wantEndpoint := "ws://127.0.0.1:12345/healthy"
	healthChecked := false
	m := &Manager{
		cfg: &config.Config{Host: "127.0.0.1"},
		bindSessionEndpoint: func(*Session) (string, error) {
			return wantEndpoint, nil
		},
		endpointHealthCheck: func(endpoint string, timeoutMS float64) error {
			healthChecked = true
			if endpoint != wantEndpoint {
				t.Fatalf("endpoint = %s, want %s", endpoint, wantEndpoint)
			}
			if timeoutMS != defaultEndpointHealthTimeoutMS {
				t.Fatalf("timeoutMS = %v, want %v", timeoutMS, defaultEndpointHealthTimeoutMS)
			}
			return nil
		},
	}
	p := &profile.Profile{ID: "prof_ok", Engine: "chromium"}
	s := &Session{ID: "sess_prof_ok", ProfileID: p.ID, Engine: p.Engine}

	if err := m.prepareSessionEndpoint(p, s, 1); err != nil {
		t.Fatalf("prepareSessionEndpoint: %v", err)
	}
	if !healthChecked {
		t.Fatal("endpoint health check was not called")
	}
	if s.ConnectURL != wantEndpoint {
		t.Fatalf("ConnectURL = %q, want %q", s.ConnectURL, wantEndpoint)
	}
}

func TestEndpointHealthErrorClassification(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{errors.New("browserType.connect: Timeout 30000ms exceeded"), "BROWSER_CONNECT_TIMEOUT"},
		{errors.New("TargetClosedException: Process exited"), "BROWSER_PROCESS_EXITED"},
		{errors.New("Target page, context or browser has been closed"), "BROWSER_PROCESS_EXITED"},
		{errors.New("websocket: bad handshake"), "ENDPOINT_UNHEALTHY"},
	}
	for _, tc := range cases {
		if got := classifyEndpointHealthError(tc.err); got != tc.want {
			t.Fatalf("classifyEndpointHealthError(%q) = %q, want %q", tc.err, got, tc.want)
		}
	}
}

func TestCleanProfileLocks(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"SingletonLock", "SingletonCookie", "SingletonSocket"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("lock"), 0644); err != nil {
			t.Fatalf("write lock %s: %v", name, err)
		}
	}

	cleanProfileLocks(dir)

	for _, name := range []string{"SingletonLock", "SingletonCookie", "SingletonSocket"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be removed, got err=%v", name, err)
		}
	}
}

func TestSanitizeExtraChromiumArgs(t *testing.T) {
	got := sanitizeExtraChromiumArgs([]string{
		" --disable-features=Translate ",
		"--user-data-dir=C:\\temp\\profile",
		"--remote-debugging-port=9222",
		"--remote-debugging-pipe",
		"--disk-cache-dir=C:\\temp\\cache",
		"--proxy-server=http://proxy.example",
		"--disable-features=Translate",
		"",
		"--disable-background-networking",
	})

	want := []string{"--disable-features=Translate", "--disable-background-networking"}
	if len(got) != len(want) {
		t.Fatalf("args len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("args[%d] = %q, want %q; all=%#v", i, got[i], want[i], got)
		}
	}
}

func TestRepairTransientChromiumDataPreservesProfileState(t *testing.T) {
	dir := t.TempDir()
	removePaths := []string{
		filepath.Join("Default", "Cache", "data.bin"),
		filepath.Join("Default", "Code Cache", "code.bin"),
		filepath.Join("Default", "GPUCache", "gpu.bin"),
		filepath.Join("BrowseForgeRuntimeCache", "cache-1", "data.bin"),
		"ShaderCache",
		"GrShaderCache",
		"component_crx_cache",
	}
	for _, rel := range removePaths {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte("cache"), 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	preservePaths := []string{
		filepath.Join("Default", "Cookies"),
		filepath.Join("Default", "Local Storage", "leveldb", "state.log"),
		filepath.Join("Default", "Preferences"),
	}
	for _, rel := range preservePaths {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte("state"), 0644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	repairTransientChromiumData(dir)

	for _, rel := range removePaths {
		if _, err := os.Stat(filepath.Join(dir, rel)); !os.IsNotExist(err) {
			t.Fatalf("expected transient path %s removed, err=%v", rel, err)
		}
	}
	for _, rel := range preservePaths {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Fatalf("expected profile state %s preserved: %v", rel, err)
		}
	}
}

func TestShouldAutoFallbackCloakBrowserLaunch(t *testing.T) {
	gpuErr := errors.New("GPU process isn't usable. Goodbye.")
	if shouldAutoFallbackCloakBrowserLaunch(nil, gpuErr) {
		t.Fatal("nil policy should not fallback")
	}
	if shouldAutoFallbackCloakBrowserLaunch(&config.CloakBrowserConfig{}, gpuErr) {
		t.Fatal("fallback should be opt-in")
	}
	if !shouldAutoFallbackCloakBrowserLaunch(&config.CloakBrowserConfig{AutoSafeGPUFallback: true}, gpuErr) {
		t.Fatal("expected opt-in GPU/cache error to fallback")
	}
	if shouldAutoFallbackCloakBrowserLaunch(&config.CloakBrowserConfig{AutoSafeGPUFallback: true}, errors.New("profile appears to be in use")) {
		t.Fatal("non GPU/cache errors should not fallback")
	}
	if shouldAutoFallbackCloakBrowserLaunch(&config.CloakBrowserConfig{
		AutoSafeGPUFallback:  true,
		SafeGPU:              true,
		IsolatedRuntimeCache: true,
	}, gpuErr) {
		t.Fatal("policy already using fallback-equivalent settings should not fallback again")
	}
}

func TestApplyCloakBrowserLaunchPolicyFallbackArgs(t *testing.T) {
	dir := t.TempDir()
	args, err := applyCloakBrowserLaunchPolicy(
		[]string{"--no-first-run"},
		dir,
		&config.CloakBrowserConfig{
			AutoSafeGPUFallback: true,
			ExtraArgs:           []string{"--disable-features=Translate", "--user-data-dir=C:\\temp"},
		},
		true,
	)
	if err != nil {
		t.Fatalf("apply fallback policy: %v", err)
	}

	joined := strings.Join(args, "\n")
	for _, want := range []string{
		"--disable-gpu",
		"--disable-gpu-compositing",
		"--disable-gpu-sandbox",
		"--disable-gpu-shader-disk-cache",
		"--in-process-gpu",
		"--disable-features=Translate",
		"--disk-cache-dir=",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("fallback args missing %q: %#v", want, args)
		}
	}
	if strings.Contains(joined, "--user-data-dir") {
		t.Fatalf("unsafe extra arg was not filtered: %#v", args)
	}
}
