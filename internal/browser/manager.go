package browser

import (
	"fmt"
	"log/slog"
	"sync"

	"browseforge/internal/config"
	"browseforge/internal/groups"
	"browseforge/internal/profile"

	"github.com/playwright-community/playwright-go"
)

// Session represents a running browser profile
type Session struct {
	ID             string
	ProfileID      string
	Engine         string
	Browser        playwright.Browser
	Context        playwright.BrowserContext
	Page           playwright.Page
	relay          *SOCKS5Relay
	ConnectURL     string // Bind endpoint for external Playwright clients
	ProfileDir     string
	UserDataDir    string
	ExecutablePath string
}

// Manager handles browser instances (multi-instance: one process per profile)
type Manager struct {
	cfg                     *config.Config
	groupStore              *groups.Store
	pw                      *playwright.Playwright
	mu                      sync.RWMutex
	sessions                map[string]*Session // sessionID → Session
	endpointHealthTimeoutMS float64
	bindSessionEndpoint     func(*Session) (string, error)
	endpointHealthCheck     func(string, float64) error
}

func NewManager(cfg *config.Config, groupStores ...*groups.Store) (*Manager, error) {
	if err := playwright.Install(&playwright.RunOptions{SkipInstallBrowsers: true}); err != nil {
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
		cfg:                     cfg,
		groupStore:              groupStore,
		pw:                      pw,
		sessions:                make(map[string]*Session),
		endpointHealthTimeoutMS: defaultEndpointHealthTimeoutMS,
	}
	m.recoverOrphanSessions()
	return m, nil
}

// Playwright returns the underlying Playwright driver instance.
// Used by integration components that need to reuse the same Playwright process.
func (m *Manager) Playwright() *playwright.Playwright {
	return m.pw
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
			slog.Warn("session endpoint unhealthy; closing stale session", "session", s.ID, "profile", p.ID, "engine", s.Engine, "connectURL", s.ConnectURL, "error", err)
			m.closeSessionResources(s, "stale endpoint")
			delete(m.sessions, id)
		}
	}

	var lastErr error
	for attempt := 1; attempt <= 2; attempt++ {
		session, err := m.launchProfile(p)
		if err == nil {
			err = m.prepareSessionEndpoint(p, session, attempt)
		}
		if err == nil {
			m.sessions[session.ID] = session
			slog.Info("session launched", "session", session.ID, "profile", p.ID, "engine", session.Engine, "connectURL", session.ConnectURL, "profileDir", session.ProfileDir, "userDataDir", session.UserDataDir, "executablePath", session.ExecutablePath)
			return session, nil
		}

		lastErr = err
		if session != nil {
			m.closeSessionResources(session, "failed launch attempt")
		}
		if attempt == 1 && shouldRetryLaunchAttempt(err) {
			slog.Warn("browser launch endpoint failed; restarting Playwright and retrying", "profile", p.ID, "engine", p.Engine, "attempt", attempt, "code", ErrorCode(err), "error", err)
			if len(m.sessions) > 0 {
				m.dropSessionsLocked("playwright driver restart after launch endpoint failure")
			}
			if restartErr := m.restartPlaywright(); restartErr != nil {
				return nil, fmt.Errorf("%w; playwright restart failed: %v", err, restartErr)
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
	switch p.Engine {
	case "chromium":
		return m.launchChromium(p)
	default:
		return m.launchFirefox(p)
	}
}

const defaultEndpointHealthTimeoutMS = 5000

func (m *Manager) prepareSessionEndpoint(p *profile.Profile, session *Session, attempt int) error {
	if session == nil {
		return codedError{code: "LAUNCH_FAILED", err: fmt.Errorf("browser launcher returned nil session")}
	}
	endpoint, err := m.bindExternalEndpoint(session)
	if err != nil {
		wrapped := codedError{code: "BROWSER_BIND_FAILED", err: fmt.Errorf("bind Playwright endpoint: %w", err)}
		slog.Warn("session endpoint bind failed", "session", session.ID, "profile", p.ID, "engine", session.Engine, "attempt", attempt, "profileDir", session.ProfileDir, "userDataDir", session.UserDataDir, "executablePath", session.ExecutablePath, "error", wrapped)
		return wrapped
	}
	session.ConnectURL = endpoint
	slog.Info("session endpoint bound", "session", session.ID, "profile", p.ID, "engine", session.Engine, "attempt", attempt, "connectURL", endpoint, "profileDir", session.ProfileDir, "userDataDir", session.UserDataDir, "executablePath", session.ExecutablePath)
	return m.validateSessionEndpoint(p, session, "launch")
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
		return codedError{code: "ENDPOINT_UNHEALTHY", err: fmt.Errorf("session %s has no Playwright endpoint", session.ID)}
	}
	timeoutMS := m.endpointHealthTimeoutMS
	if timeoutMS <= 0 {
		timeoutMS = defaultEndpointHealthTimeoutMS
	}
	if err := m.checkEndpointHealth(session, timeoutMS); err != nil {
		code := classifyEndpointHealthError(err)
		wrapped := codedError{code: code, err: fmt.Errorf("Playwright endpoint health check failed: %w", err)}
		slog.Warn("session endpoint health failed", "session", session.ID, "profile", p.ID, "engine", session.Engine, "stage", stage, "connectURL", session.ConnectURL, "timeoutMS", timeoutMS, "code", code, "profileDir", session.ProfileDir, "userDataDir", session.UserDataDir, "executablePath", session.ExecutablePath, "error", wrapped)
		return wrapped
	}
	slog.Info("session endpoint healthy", "session", session.ID, "profile", p.ID, "engine", session.Engine, "stage", stage, "connectURL", session.ConnectURL, "timeoutMS", timeoutMS)
	return nil
}

func (m *Manager) checkEndpointHealth(session *Session, timeoutMS float64) error {
	if m.endpointHealthCheck != nil {
		return m.endpointHealthCheck(session.ConnectURL, timeoutMS)
	}
	if m.pw == nil {
		return fmt.Errorf("playwright driver is not running")
	}
	var (
		connected playwright.Browser
		err       error
	)
	options := playwright.BrowserTypeConnectOptions{Timeout: playwright.Float(timeoutMS)}
	if session.Engine == "firefox" {
		connected, err = m.pw.Firefox.Connect(session.ConnectURL, options)
	} else {
		connected, err = m.pw.Chromium.Connect(session.ConnectURL, options)
	}
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer connected.Close()
	page, err := connected.NewPage()
	if err != nil {
		return fmt.Errorf("new page: %w", err)
	}
	defer page.Close()
	return nil
}

func (m *Manager) closeSessionResources(s *Session, reason string) {
	if s == nil {
		return
	}
	if s.Context != nil {
		if err := s.Context.Close(); err != nil {
			slog.Warn("session context close failed", "session", s.ID, "reason", reason, "error", err)
		}
	}
	if s.Browser != nil {
		if err := s.Browser.Close(); err != nil {
			slog.Warn("session browser close failed", "session", s.ID, "reason", reason, "error", err)
		}
	}
	if s.relay != nil {
		s.relay.Close()
	}
	slog.Info("session resources closed", "session", s.ID, "reason", reason, "profile", s.ProfileID, "engine", s.Engine, "connectURL", s.ConnectURL)
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
		if s.relay != nil {
			s.relay.Close()
		}
		delete(m.sessions, id)
		slog.Warn("session dropped", "session", id, "reason", reason)
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
	var result []*Session
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
	if s.Context != nil {
		s.Context.Close()
	}
	if s.Browser != nil {
		s.Browser.Close()
	}
	if s.relay != nil {
		s.relay.Close()
	}
	delete(m.sessions, id)
	slog.Info("session closed", "session", id)
	return nil
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, s := range m.sessions {
		if s.Context != nil {
			s.Context.Close()
		}
		if s.relay != nil {
			s.relay.Close()
		}
		delete(m.sessions, id)
	}
	if m.pw != nil {
		m.pw.Stop()
	}
}
