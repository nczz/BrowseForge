package browser

import (
	"browseforge/internal/config"
	"browseforge/internal/groups"
	"browseforge/internal/profile"
	bfruntime "browseforge/internal/runtime"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/mxschmitt/playwright-go"
)

// Session represents a running browser profile
type Session struct {
	ID                  string
	LaunchID            string
	ProfileID           string
	RuntimeID           string
	Browser             playwright.Browser
	Context             playwright.BrowserContext
	Page                playwright.Page
	relay               *SOCKS5Relay
	ConnectURL          string // Internal Playwright Bind endpoint
	ProfileDir          string
	UserDataDir         string
	ExecutablePath      string
	CreatedAt           time.Time
	EndpointBoundAt     time.Time
	HealthStatus        string
	HealthAttempts      int
	LastHealthAt        time.Time
	LastHealthLatencyMS int64
}

// Manager handles browser instances (multi-instance: one process per profile)
type Manager struct {
	cfg                  *config.Config
	groupStore           *groups.Store
	runtimes             *bfruntime.Registry
	pw                   *playwright.Playwright
	mu                   sync.RWMutex
	eventsMu             sync.RWMutex
	sessions             map[string]*Session // sessionID → Session
	endpointHealthPolicy EndpointHealthPolicy
	bindSessionEndpoint  func(*Session) (string, error)
	endpointHealthCheck  func(string, float64, bool) error
	sessionEvents        []SessionLifecycleEvent
	sessionEventStart    int
	nextEventSequence    uint64
	nextLaunchSequence   uint64
}

// InstallPlaywrightDriver installs the Playwright control driver without
// downloading Playwright-managed browsers. BrowseForge manages browser binaries
// separately, but the Go client still needs playwright-core and Node.js to
// connect to those browsers.
func InstallPlaywrightDriver() error {
	return playwright.Install(&playwright.RunOptions{SkipInstallBrowsers: true})
}

func NewManager(cfg *config.Config, groupStores ...*groups.Store) (*Manager, error) {
	if err := InstallPlaywrightDriver(); err != nil {
		return nil, fmt.Errorf("playwright.Install: %w", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("playwright.Run: %w", err)
	}

	var groupStore *groups.Store
	if len(groupStores) > 0 {
		groupStore = groupStores[0]
	}
	m := &Manager{
		cfg:                  cfg,
		groupStore:           groupStore,
		runtimes:             bfruntime.NewRegistry(cfg),
		pw:                   pw,
		sessions:             make(map[string]*Session),
		endpointHealthPolicy: endpointHealthPolicyFromConfig(cfg.EndpointHealth),
	}
	m.recoverOrphanSessions()
	return m, nil
}

// Playwright returns the underlying Playwright driver instance.
// Used by integration components that need to reuse the same Playwright process.
func (m *Manager) Playwright() *playwright.Playwright {
	return m.pw
}

func (m *Manager) RuntimeRegistry() *bfruntime.Registry {
	return m.runtimes
}

// recoverOrphanSessions cleans up stale session state on startup
func (m *Manager) recoverOrphanSessions() {
	// On startup, all previous sessions are dead (processes gone)
	// Just ensure clean state
	slog.Info("session recovery: starting clean")
}

func (m *Manager) LaunchSession(p *profile.Profile) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, s := range m.sessions {
		if s.ProfileID != p.ID {
			continue
		}
		if err := m.validateSessionEndpoint(p, s, "existing"); err == nil {
			return s, nil
		} else {
			slog.Warn("session endpoint unhealthy; closing stale session", "session", s.ID, "profile", p.ID, "runtime_id", s.RuntimeID, "connectURL", s.ConnectURL, "error", err)
			m.closeSessionResources(s, "stale endpoint")
			delete(m.sessions, id)
		}
	}

	var lastErr error
	for attempt := 1; attempt <= 2; attempt++ {
		session, err := m.launchProfile(p)
		if session != nil {
			m.assignLaunchIDLocked(session)
			m.recordSessionEventLocked(session, "browser_launched", sessionEventDetails{attempt: attempt})
		}
		if err == nil {
			err = m.prepareSessionEndpoint(p, session, attempt)
		}
		if err == nil {
			m.observeBrowserLifecycle(session)
			m.sessions[session.ID] = session
			m.recordSessionEventLocked(session, "session_registered", sessionEventDetails{attempt: attempt})
			slog.Info("session launched", "session", session.ID, "launch_id", session.LaunchID, "profile", p.ID, "runtime_id", session.RuntimeID, "connectURL", session.ConnectURL, "profileDir", session.ProfileDir, "userDataDir", session.UserDataDir, "executablePath", session.ExecutablePath)
			return session, nil
		}

		lastErr = err
		if session != nil {
			m.recordSessionEventLocked(session, "session_launch_failed", sessionEventDetails{attempt: attempt, code: ErrorCode(err), message: err.Error()})
			m.closeSessionResources(session, "failed launch attempt")
		}
		if session != nil && isEndpointStabilityFailure(err) {
			policy := m.cfg.ChromiumRuntimeSettings(session.RuntimeID)
			if policy != nil && policy.RepairTransientCacheOnLaunchFailure {
				m.recordSessionEventLocked(session, "profile_transient_cache_repaired", sessionEventDetails{
					code:    ErrorCode(err),
					reason:  "endpoint stability failure",
					message: err.Error(),
				})
				slog.Warn("repairing transient chromium profile data after endpoint stability failure", "profile", p.ID, "runtime_id", session.RuntimeID, "userDataDir", session.UserDataDir, "code", ErrorCode(err), "error", err)
				repairTransientChromiumData(session.UserDataDir)
			}
		}
		if attempt == 1 && shouldRetryLaunchAttempt(err) {
			slog.Warn("browser launch endpoint failed; restarting Playwright and retrying", "profile", p.ID, "runtime_id", p.RuntimeID, "attempt", attempt, "code", ErrorCode(err), "error", err)
			if session != nil {
				m.recordSessionEventLocked(session, "playwright_restart_started", sessionEventDetails{attempt: attempt, code: ErrorCode(err), reason: "launch endpoint failure"})
			}
			if len(m.sessions) > 0 {
				m.dropSessionsLocked("playwright driver restart after launch endpoint failure")
			}
			if restartErr := m.restartPlaywright(); restartErr != nil {
				if session != nil {
					m.recordSessionEventLocked(session, "playwright_restart_failed", sessionEventDetails{attempt: attempt, message: restartErr.Error()})
				}
				return nil, fmt.Errorf("%w; playwright restart failed: %v", err, restartErr)
			}
			if session != nil {
				m.recordSessionEventLocked(session, "playwright_restarted", sessionEventDetails{attempt: attempt})
			}
			continue
		}
		break
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("launch session failed")
}

func (m *Manager) launchProfile(p *profile.Profile) (*Session, error) {
	desc, err := m.runtimes.ResolveProfile(p)
	if err != nil {
		return nil, err
	}
	if !desc.Enabled {
		return nil, fmt.Errorf("runtime %q is disabled", desc.ID)
	}
	switch desc.ID {
	case bfruntime.CloakBrowser, bfruntime.BrowseForgeChromium:
		return m.launchChromium(p)
	case bfruntime.Camoufox:
		return m.launchFirefox(p)
	default:
		return nil, fmt.Errorf("runtime %q is registered but has no launcher", desc.ID)
	}
}

func (m *Manager) prepareSessionEndpoint(p *profile.Profile, session *Session, attempt int) error {
	if session == nil {
		return codedError{code: "LAUNCH_FAILED", err: fmt.Errorf("browser launcher returned nil session")}
	}
	endpoint, err := m.bindExternalEndpoint(session)
	if err != nil {
		wrapped := codedError{code: "BROWSER_BIND_FAILED", err: fmt.Errorf("bind Playwright endpoint: %w", err)}
		m.recordSessionEventLocked(session, "endpoint_bind_failed", sessionEventDetails{attempt: attempt, code: wrapped.code, message: wrapped.Error()})
		slog.Warn("session endpoint bind failed", "session", session.ID, "launch_id", session.LaunchID, "profile", p.ID, "runtime_id", session.RuntimeID, "attempt", attempt, "profileDir", session.ProfileDir, "userDataDir", session.UserDataDir, "executablePath", session.ExecutablePath, "error", wrapped)
		return wrapped
	}
	session.ConnectURL = endpoint
	session.EndpointBoundAt = time.Now().UTC()
	m.recordSessionEventLocked(session, "endpoint_bound", sessionEventDetails{attempt: attempt})
	slog.Info("session endpoint bound", "session", session.ID, "launch_id", session.LaunchID, "profile", p.ID, "runtime_id", session.RuntimeID, "attempt", attempt, "connectURL", endpoint, "profileDir", session.ProfileDir, "userDataDir", session.UserDataDir, "executablePath", session.ExecutablePath)
	return m.validateSessionEndpoint(p, session, "launch")
}

func (m *Manager) observeBrowserLifecycle(session *Session) {
	if session == nil || session.Context == nil {
		return
	}
	launchedBrowser := session.Context.Browser()
	if launchedBrowser == nil {
		return
	}
	launchedBrowser.OnDisconnected(func(playwright.Browser) {
		m.recordSessionEventLocked(session, "browser_disconnected", sessionEventDetails{message: "browser process closed or Playwright disconnected"})
		slog.Warn("session browser disconnected", "session", session.ID, "launch_id", session.LaunchID, "profile", session.ProfileID, "runtime_id", session.RuntimeID, "connectURL", session.ConnectURL)
	})
}

func (m *Manager) bindExternalEndpoint(session *Session) (string, error) {
	if m.bindSessionEndpoint != nil {
		return m.bindSessionEndpoint(session)
	}
	if session.Context == nil {
		return "", fmt.Errorf("browser context is nil")
	}
	browser := session.Context.Browser()
	if browser == nil {
		return "", fmt.Errorf("browser is nil")
	}
	result, err := browser.Bind("browseforge-"+session.ID, playwright.BrowserBindOptions{
		Host: playwright.String(m.cfg.Host),
		Port: playwright.Int(0),
	})
	if err != nil {
		return "", err
	}
	return result.Endpoint, nil
}

func (m *Manager) validateSessionEndpoint(p *profile.Profile, session *Session, stage string) error {
	if session.ConnectURL == "" {
		err := codedError{code: "ENDPOINT_UNHEALTHY", err: fmt.Errorf("session %s has no Playwright endpoint", session.ID)}
		m.recordSessionEventLocked(session, "endpoint_health_failed", sessionEventDetails{stage: stage, code: err.code, message: err.Error()})
		return err
	}

	policy := m.endpointHealthPolicy
	var lastErr error
	for attempt := 1; attempt <= policy.Attempts; attempt++ {
		started := time.Now()
		session.HealthAttempts = attempt
		m.recordSessionEventLocked(session, "endpoint_health_started", sessionEventDetails{stage: stage, attempt: attempt})
		err := m.checkEndpointHealthThroughBindGrace(session, float64(policy.TimeoutMS), policy.ProbePage)
		durationMS := time.Since(started).Milliseconds()
		session.LastHealthAt = time.Now().UTC()
		session.LastHealthLatencyMS = durationMS
		if err == nil {
			session.HealthStatus = "healthy"
			m.recordSessionEventLocked(session, "endpoint_healthy", sessionEventDetails{stage: stage, attempt: attempt, durationMS: durationMS})
			slog.Info("session endpoint healthy", "session", session.ID, "launch_id", session.LaunchID, "profile", p.ID, "runtime_id", session.RuntimeID, "stage", stage, "attempt", attempt, "connectURL", session.ConnectURL, "timeoutMS", policy.TimeoutMS, "durationMS", durationMS, "probePage", policy.ProbePage)
			return nil
		}

		lastErr = err
		code := classifyEndpointHealthError(err)
		session.HealthStatus = "unhealthy"
		m.recordSessionEventLocked(session, "endpoint_health_failed", sessionEventDetails{stage: stage, attempt: attempt, code: code, durationMS: durationMS, message: err.Error()})
		slog.Warn("session endpoint health failed", "session", session.ID, "launch_id", session.LaunchID, "profile", p.ID, "runtime_id", session.RuntimeID, "stage", stage, "attempt", attempt, "attempts", policy.Attempts, "connectURL", session.ConnectURL, "timeoutMS", policy.TimeoutMS, "durationMS", durationMS, "probePage", policy.ProbePage, "code", code, "willRetry", attempt < policy.Attempts, "profileDir", session.ProfileDir, "userDataDir", session.UserDataDir, "executablePath", session.ExecutablePath, "error", err)
		if attempt < policy.Attempts {
			time.Sleep(time.Duration(policy.RetryDelayMS) * time.Millisecond)
		}
	}

	code := classifyEndpointHealthError(lastErr)
	return codedError{code: code, err: fmt.Errorf("endpoint health check failed after %d attempts: %w", policy.Attempts, lastErr)}
}

func (m *Manager) checkEndpointHealth(session *Session, timeoutMS float64, probePage bool) error {
	if m.endpointHealthCheck != nil {
		return m.endpointHealthCheck(session.ConnectURL, timeoutMS, probePage)
	}
	if session.Context == nil {
		return fmt.Errorf("verify bound browser: session context is nil")
	}
	browser := session.Context.Browser()
	if browser == nil || !browser.IsConnected() {
		return fmt.Errorf("verify bound browser: browser is disconnected")
	}
	if !probePage {
		return nil
	}
	if session.Page == nil {
		return fmt.Errorf("probe existing page: session page is nil")
	}
	if _, err := session.Page.Title(); err != nil {
		return fmt.Errorf("probe existing page: %w", err)
	}
	return nil
}

func (m *Manager) checkEndpointHealthThroughBindGrace(session *Session, timeoutMS float64, probePage bool) error {
	if err := m.checkEndpointHealth(session, timeoutMS, probePage); err != nil {
		return err
	}
	wait := endpointBindGraceRemaining(session.EndpointBoundAt, m.endpointHealthPolicy.BindGraceMS, time.Now())
	if wait <= 0 {
		return nil
	}
	time.Sleep(wait)
	if err := m.checkEndpointHealth(session, timeoutMS, probePage); err != nil {
		return fmt.Errorf("verify bound browser after %dms bind grace: %w", m.endpointHealthPolicy.BindGraceMS, err)
	}
	return nil
}

func (m *Manager) closeSessionResources(s *Session, reason string) {
	if s == nil {
		return
	}
	m.recordSessionEventLocked(s, "session_close_started", sessionEventDetails{reason: reason})
	if s.Context != nil {
		if err := s.Context.Close(); err != nil {
			m.recordSessionEventLocked(s, "session_context_close_failed", sessionEventDetails{reason: reason, message: err.Error()})
			slog.Warn("session context close failed", "session", s.ID, "launch_id", s.LaunchID, "reason", reason, "error", err)
		}
	}
	if s.Browser != nil {
		if err := s.Browser.Close(); err != nil {
			m.recordSessionEventLocked(s, "session_browser_close_failed", sessionEventDetails{reason: reason, message: err.Error()})
			slog.Warn("session browser close failed", "session", s.ID, "launch_id", s.LaunchID, "reason", reason, "error", err)
		}
	}
	if s.relay != nil {
		s.relay.Close()
	}
	m.recordSessionEventLocked(s, "session_closed", sessionEventDetails{reason: reason})
	slog.Info("session resources closed", "session", s.ID, "launch_id", s.LaunchID, "reason", reason, "profile", s.ProfileID, "runtime_id", s.RuntimeID, "connectURL", s.ConnectURL)
}

func (m *Manager) effectiveProxy(p *profile.Profile) groups.EffectiveProxy {
	if m.groupStore != nil {
		return m.groupStore.EffectiveProxy(p)
	}
	if p != nil && p.Proxy != nil && p.Proxy.Host != "" {
		return groups.EffectiveProxy{Proxy: p.Proxy, Source: "profile", Mode: groups.ProxyModeDefault}
	}
	return groups.EffectiveProxy{Source: "none", Mode: groups.ProxyModeDefault}
}

func (m *Manager) restartPlaywright() error {
	if m.pw != nil {
		if err := m.pw.Stop(); err != nil {
			slog.Warn("playwright stop during restart failed", "error", err)
		}
		m.pw = nil
	}
	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("playwright.Run: %w", err)
	}
	m.pw = pw
	return nil
}

func (m *Manager) dropSessionsLocked(reason string) {
	for id, s := range m.sessions {
		m.closeSessionResources(s, reason)
		delete(m.sessions, id)
		slog.Warn("session dropped", "session", id, "launch_id", s.LaunchID, "reason", reason)
	}
}

func (m *Manager) GetSession(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	return s, ok
}

func (m *Manager) ListSessions() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		result = append(result, s)
	}
	return result
}

func (m *Manager) CloseSession(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return fmt.Errorf("session not found: %s", id)
	}
	m.closeSessionResources(s, "client_request")
	delete(m.sessions, id)
	slog.Info("session closed", "session", id, "launch_id", s.LaunchID)
	return nil
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, s := range m.sessions {
		m.closeSessionResources(s, "manager_shutdown")
		delete(m.sessions, id)
	}
	if m.pw != nil {
		m.pw.Stop()
	}
}
