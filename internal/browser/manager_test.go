package browser

import (
	"browseforge/internal/config"
	"browseforge/internal/profile"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
		cfg:                  &config.Config{Host: "127.0.0.1"},
		endpointHealthPolicy: endpointHealthPolicy(1234, 1, 1, 1, false, 32),
		bindSessionEndpoint: func(s *Session) (string, error) {
			if s.ID != "sess_prof_health" {
				t.Fatalf("bind session = %s, want sess_prof_health", s.ID)
			}
			return wantEndpoint, nil
		},
		endpointHealthCheck: func(endpoint string, timeoutMS float64, probePage bool) error {
			if endpoint != wantEndpoint {
				t.Fatalf("endpoint = %s, want %s", endpoint, wantEndpoint)
			}
			if timeoutMS != 1234 {
				t.Fatalf("timeoutMS = %v, want 1234", timeoutMS)
			}
			if probePage {
				t.Fatal("probePage = true, want false")
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
		cfg:                  &config.Config{Host: "127.0.0.1"},
		endpointHealthPolicy: endpointHealthPolicy(0, 1, 1, 1, false, 32),
		bindSessionEndpoint: func(*Session) (string, error) {
			return wantEndpoint, nil
		},
		endpointHealthCheck: func(endpoint string, timeoutMS float64, probePage bool) error {
			healthChecked = true
			if endpoint != wantEndpoint {
				t.Fatalf("endpoint = %s, want %s", endpoint, wantEndpoint)
			}
			if timeoutMS != defaultEndpointHealthTimeoutMS {
				t.Fatalf("timeoutMS = %v, want %v", timeoutMS, defaultEndpointHealthTimeoutMS)
			}
			if probePage {
				t.Fatal("probePage = true, want false")
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
		{errors.New("verify bound browser: browser is disconnected"), "BROWSER_PROCESS_EXITED"},
		{errors.New("websocket: bad handshake"), "ENDPOINT_UNHEALTHY"},
	}
	for _, tc := range cases {
		if got := classifyEndpointHealthError(tc.err); got != tc.want {
			t.Fatalf("classifyEndpointHealthError(%q) = %q, want %q", tc.err, got, tc.want)
		}
	}
}

func TestValidateSessionEndpointRetriesSameChildBeforeSuccess(t *testing.T) {
	checks := 0
	m := &Manager{
		endpointHealthPolicy: endpointHealthPolicy(2500, 2, 1, 1, false, 32),
		endpointHealthCheck: func(endpoint string, timeoutMS float64, probePage bool) error {
			checks++
			if endpoint != "ws://127.0.0.1:12345/retry" {
				t.Fatalf("endpoint = %q", endpoint)
			}
			if timeoutMS != 2500 || probePage {
				t.Fatalf("policy = timeout %v probePage %v", timeoutMS, probePage)
			}
			if checks == 1 {
				return errors.New("browserType.connect: Timeout 2500ms exceeded")
			}
			return nil
		},
	}
	p := &profile.Profile{ID: "prof_retry", Engine: "chromium"}
	s := &Session{
		ID:         "sess_prof_retry",
		LaunchID:   "launch-retry",
		ProfileID:  p.ID,
		Engine:     p.Engine,
		ConnectURL: "ws://127.0.0.1:12345/retry",
	}

	if err := m.validateSessionEndpoint(p, s, "launch"); err != nil {
		t.Fatalf("validateSessionEndpoint: %v", err)
	}
	if checks != 2 {
		t.Fatalf("checks = %d, want 2", checks)
	}
	if s.HealthStatus != "healthy" || s.HealthAttempts != 2 {
		t.Fatalf("health = %q attempts = %d", s.HealthStatus, s.HealthAttempts)
	}
	events := m.SessionEvents(SessionEventFilter{LaunchID: s.LaunchID, Limit: 10})
	if len(events) != 4 {
		t.Fatalf("events = %#v, want start/fail/start/healthy", events)
	}
	if events[0].Event != "endpoint_healthy" || events[1].Event != "endpoint_health_started" || events[2].Event != "endpoint_health_failed" {
		t.Fatalf("events newest first = %#v", events)
	}
}

func TestValidateSessionEndpointRejectsDisconnectDuringBindGrace(t *testing.T) {
	checks := 0
	m := &Manager{
		endpointHealthPolicy: endpointHealthPolicy(2500, 1, 1, 10, false, 32),
		endpointHealthCheck: func(string, float64, bool) error {
			checks++
			if checks == 1 {
				return nil
			}
			return errors.New("verify bound browser: browser is disconnected")
		},
	}
	p := &profile.Profile{ID: "prof_unstable", Engine: "chromium"}
	s := &Session{
		ID:              "sess_prof_unstable",
		LaunchID:        "launch-unstable",
		ProfileID:       p.ID,
		Engine:          p.Engine,
		ConnectURL:      "ws://127.0.0.1:12345/unstable",
		EndpointBoundAt: time.Now(),
	}

	err := m.validateSessionEndpoint(p, s, "launch")
	if err == nil {
		t.Fatal("validateSessionEndpoint succeeded after browser disconnected during bind grace")
	}
	if got := ErrorCode(err); got != "BROWSER_PROCESS_EXITED" {
		t.Fatalf("ErrorCode = %q, want BROWSER_PROCESS_EXITED; err=%v", got, err)
	}
	if checks != 2 {
		t.Fatalf("checks = %d, want initial and post-grace checks", checks)
	}
}

func TestSessionEventsRemainAvailableAfterCloseAndAreBounded(t *testing.T) {
	m := &Manager{endpointHealthPolicy: endpointHealthPolicy(0, 0, 0, 0, false, 3)}
	s := &Session{ID: "sess_prof_history", LaunchID: "launch-history", ProfileID: "prof_history", Engine: "chromium"}
	m.recordSessionEventLocked(s, "browser_launched", sessionEventDetails{})
	m.closeSessionResources(s, "test_cleanup")

	events := m.SessionEvents(SessionEventFilter{ProfileID: s.ProfileID, Limit: 10})
	if len(events) != 3 {
		t.Fatalf("len(events) = %d, want 3", len(events))
	}
	if events[0].Event != "session_closed" ||
		events[0].Reason != "test_cleanup" ||
		events[1].Event != "session_close_started" ||
		events[2].Event != "browser_launched" {
		t.Fatalf("post-close events newest first = %#v", events)
	}
}

func TestEndpointHealthPolicyFromConfig(t *testing.T) {
	got := endpointHealthPolicyFromConfig(&config.EndpointHealthConfig{
		TimeoutMS:    maxEndpointHealthTimeoutMS + 1,
		Attempts:     maxEndpointHealthAttempts + 1,
		RetryDelayMS: maxEndpointHealthRetryDelayMS + 1,
		BindGraceMS:  maxEndpointBindGraceMS + 1,
		ProbePage:    true,
		HistoryLimit: maxSessionEventHistoryLimit + 1,
	})
	if got.TimeoutMS != maxEndpointHealthTimeoutMS ||
		got.Attempts != maxEndpointHealthAttempts ||
		got.RetryDelayMS != maxEndpointHealthRetryDelayMS ||
		got.BindGraceMS != maxEndpointBindGraceMS ||
		!got.ProbePage ||
		got.HistoryLimit != maxSessionEventHistoryLimit {
		t.Fatalf("policy = %#v", got)
	}
	defaults := endpointHealthPolicyFromConfig(nil)
	if defaults.TimeoutMS != defaultEndpointHealthTimeoutMS ||
		defaults.Attempts != defaultEndpointHealthAttempts ||
		defaults.RetryDelayMS != defaultEndpointHealthRetryDelayMS ||
		defaults.BindGraceMS != defaultEndpointBindGraceMS ||
		defaults.ProbePage ||
		defaults.HistoryLimit != defaultSessionEventHistoryLimit {
		t.Fatalf("default policy = %#v", defaults)
	}
}

func TestEndpointBindGraceRemaining(t *testing.T) {
	now := time.Date(2026, time.March, 16, 12, 0, 0, 0, time.UTC)
	boundAt := now.Add(-40 * time.Millisecond)

	if got := endpointBindGraceRemaining(boundAt, 100, now); got != 60*time.Millisecond {
		t.Fatalf("remaining grace = %s, want 60ms", got)
	}
	if got := endpointBindGraceRemaining(boundAt, 40, now); got != 0 {
		t.Fatalf("expired grace = %s, want 0", got)
	}
	if got := endpointBindGraceRemaining(time.Time{}, 100, now); got != 0 {
		t.Fatalf("zero bind timestamp grace = %s, want 0", got)
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
		filepath.Join("BrowserMetrics", "BrowserMetrics-old.pma"),
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
