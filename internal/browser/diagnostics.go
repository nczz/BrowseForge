package browser

import (
	"time"

	"browseforge/internal/config"
)

const (
	defaultEndpointHealthTimeoutMS    = 15000
	defaultEndpointHealthAttempts     = 2
	defaultEndpointHealthRetryDelayMS = 500
	defaultEndpointBindGraceMS        = 100
	defaultSessionEventHistoryLimit   = 256
	maxEndpointHealthTimeoutMS        = 120000
	maxEndpointHealthAttempts         = 5
	maxEndpointHealthRetryDelayMS     = 30000
	maxEndpointBindGraceMS            = 5000
	maxSessionEventHistoryLimit       = 2000
)

// EndpointHealthPolicy controls non-consuming session validation, the minimum
// delay before attaching to a new Playwright Bind endpoint, and diagnostics.
type EndpointHealthPolicy struct {
	TimeoutMS    int  `json:"timeout_ms"`
	Attempts     int  `json:"attempts"`
	RetryDelayMS int  `json:"retry_delay_ms"`
	BindGraceMS  int  `json:"bind_grace_ms"`
	ProbePage    bool `json:"probe_page"`
	HistoryLimit int  `json:"history_limit"`
}

// SessionLifecycleEvent is a bounded, in-memory postmortem record. Persistent
// JSON logs remain the source of truth across BrowseForge restarts.
type SessionLifecycleEvent struct {
	Sequence   uint64    `json:"sequence"`
	Timestamp  time.Time `json:"timestamp"`
	Event      string    `json:"event"`
	SessionID  string    `json:"session_id,omitempty"`
	LaunchID   string    `json:"launch_id,omitempty"`
	ProfileID  string    `json:"profile_id,omitempty"`
	RuntimeID  string    `json:"runtime_id,omitempty"`
	Stage      string    `json:"stage,omitempty"`
	Attempt    int       `json:"attempt,omitempty"`
	Code       string    `json:"code,omitempty"`
	DurationMS int64     `json:"duration_ms,omitempty"`
	Endpoint   string    `json:"endpoint,omitempty"`
	Reason     string    `json:"reason,omitempty"`
	Message    string    `json:"message,omitempty"`
}

type SessionEventFilter struct {
	SessionID string
	LaunchID  string
	ProfileID string
	Limit     int
}

type sessionEventDetails struct {
	stage      string
	attempt    int
	code       string
	durationMS int64
	reason     string
	message    string
}

func endpointHealthPolicy(cfgTimeoutMS, cfgAttempts, cfgRetryDelayMS, cfgBindGraceMS int, probePage bool, cfgHistoryLimit int) EndpointHealthPolicy {
	policy := EndpointHealthPolicy{
		TimeoutMS:    cfgTimeoutMS,
		Attempts:     cfgAttempts,
		RetryDelayMS: cfgRetryDelayMS,
		BindGraceMS:  cfgBindGraceMS,
		ProbePage:    probePage,
		HistoryLimit: cfgHistoryLimit,
	}
	if policy.TimeoutMS <= 0 {
		policy.TimeoutMS = defaultEndpointHealthTimeoutMS
	}
	if policy.TimeoutMS > maxEndpointHealthTimeoutMS {
		policy.TimeoutMS = maxEndpointHealthTimeoutMS
	}
	if policy.Attempts <= 0 {
		policy.Attempts = defaultEndpointHealthAttempts
	}
	if policy.RetryDelayMS <= 0 {
		policy.RetryDelayMS = defaultEndpointHealthRetryDelayMS
	}
	if policy.Attempts > maxEndpointHealthAttempts {
		policy.Attempts = maxEndpointHealthAttempts
	}
	if policy.BindGraceMS <= 0 {
		policy.BindGraceMS = defaultEndpointBindGraceMS
	}
	if policy.RetryDelayMS > maxEndpointHealthRetryDelayMS {
		policy.RetryDelayMS = maxEndpointHealthRetryDelayMS
	}
	if policy.HistoryLimit <= 0 {
		policy.HistoryLimit = defaultSessionEventHistoryLimit
	}
	if policy.BindGraceMS > maxEndpointBindGraceMS {
		policy.BindGraceMS = maxEndpointBindGraceMS
	}
	if policy.HistoryLimit > maxSessionEventHistoryLimit {
		policy.HistoryLimit = maxSessionEventHistoryLimit
	}
	return policy
}

func endpointHealthPolicyFromConfig(raw *config.EndpointHealthConfig) EndpointHealthPolicy {
	if raw == nil {
		return endpointHealthPolicy(0, 0, 0, 0, false, 0)
	}
	return endpointHealthPolicy(raw.TimeoutMS, raw.Attempts, raw.RetryDelayMS, raw.BindGraceMS, raw.ProbePage, raw.HistoryLimit)
}

func (m *Manager) EndpointHealthPolicy() EndpointHealthPolicy {
	return m.endpointHealthPolicy
}

func endpointBindGraceRemaining(boundAt time.Time, bindGraceMS int, now time.Time) time.Duration {
	if boundAt.IsZero() || bindGraceMS <= 0 {
		return 0
	}
	remaining := boundAt.Add(time.Duration(bindGraceMS) * time.Millisecond).Sub(now)
	if remaining <= 0 {
		return 0
	}
	return remaining
}

// WaitForEndpointBindGrace prevents consumers from attaching during the
// short Playwright Bind handoff window observed on Windows.
func (m *Manager) WaitForEndpointBindGrace(session *Session, stage string) time.Duration {
	if session == nil {
		return 0
	}
	wait := endpointBindGraceRemaining(session.EndpointBoundAt, m.endpointHealthPolicy.BindGraceMS, time.Now())
	if wait <= 0 {
		return 0
	}
	time.Sleep(wait)
	m.recordSessionEventLocked(session, "endpoint_bind_grace_waited", sessionEventDetails{
		stage:      stage,
		durationMS: wait.Milliseconds(),
	})
	return wait
}

// RecordSessionProxyEvent adds proxy lifecycle evidence to the same bounded
// history used by browser launch diagnostics.
func (m *Manager) RecordSessionProxyEvent(session *Session, eventName, code, message string, duration time.Duration) {
	m.recordSessionEventLocked(session, eventName, sessionEventDetails{
		stage:      "proxy",
		code:       code,
		durationMS: duration.Milliseconds(),
		message:    message,
	})
}

// SessionEvents returns newest events first so the most recent failure is
// immediately visible to CLI and API consumers.
func (m *Manager) SessionEvents(filter SessionEventFilter) []SessionLifecycleEvent {
	m.eventsMu.RLock()
	defer m.eventsMu.RUnlock()

	limit := filter.Limit
	if limit <= 0 || limit > m.endpointHealthPolicy.HistoryLimit {
		limit = m.endpointHealthPolicy.HistoryLimit
	}
	result := make([]SessionLifecycleEvent, 0, min(limit, len(m.sessionEvents)))
	for offset := 0; offset < len(m.sessionEvents) && len(result) < limit; offset++ {
		index := (m.sessionEventStart + len(m.sessionEvents) - 1 - offset) % len(m.sessionEvents)
		event := m.sessionEvents[index]
		if filter.SessionID != "" && event.SessionID != filter.SessionID {
			continue
		}
		if filter.LaunchID != "" && event.LaunchID != filter.LaunchID {
			continue
		}
		if filter.ProfileID != "" && event.ProfileID != filter.ProfileID {
			continue
		}
		result = append(result, event)
	}
	return result
}

func (m *Manager) assignLaunchIDLocked(session *Session) {
	m.nextLaunchSequence++
	now := time.Now().UTC()
	session.LaunchID = session.ID + "-" + now.Format("20060102T150405.000000000Z") + "-" + formatSequence(m.nextLaunchSequence)
	session.CreatedAt = now
}

func formatSequence(sequence uint64) string {
	const digits = "0123456789"
	var buf [20]byte
	i := len(buf)
	for {
		i--
		buf[i] = digits[sequence%10]
		sequence /= 10
		if sequence == 0 {
			return string(buf[i:])
		}
	}
}

func (m *Manager) recordSessionEventLocked(session *Session, eventName string, details sessionEventDetails) {
	m.eventsMu.Lock()
	defer m.eventsMu.Unlock()
	m.nextEventSequence++
	event := SessionLifecycleEvent{
		Sequence:   m.nextEventSequence,
		Timestamp:  time.Now().UTC(),
		Event:      eventName,
		Stage:      details.stage,
		Attempt:    details.attempt,
		Code:       details.code,
		DurationMS: details.durationMS,
		Reason:     details.reason,
		Message:    details.message,
	}
	if session != nil {
		event.SessionID = session.ID
		event.LaunchID = session.LaunchID
		event.ProfileID = session.ProfileID
		event.RuntimeID = session.RuntimeID
		event.Endpoint = session.ConnectURL
	}

	limit := m.endpointHealthPolicy.HistoryLimit
	if len(m.sessionEvents) < limit {
		m.sessionEvents = append(m.sessionEvents, event)
		return
	}
	m.sessionEvents[m.sessionEventStart] = event
	m.sessionEventStart = (m.sessionEventStart + 1) % len(m.sessionEvents)
}
