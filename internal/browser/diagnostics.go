package browser

import (
	"time"

	"browseforge/internal/config"
)

const (
	defaultEndpointHealthTimeoutMS    = 15000
	defaultEndpointHealthAttempts     = 2
	defaultEndpointHealthRetryDelayMS = 500
	defaultSessionEventHistoryLimit   = 256
	maxEndpointHealthTimeoutMS        = 120000
	maxEndpointHealthAttempts         = 5
	maxEndpointHealthRetryDelayMS     = 30000
	maxSessionEventHistoryLimit       = 2000
)

// EndpointHealthPolicy describes how BrowseForge validates a Playwright Bind
// endpoint before exposing the session to external clients.
type EndpointHealthPolicy struct {
	TimeoutMS    int  `json:"timeout_ms"`
	Attempts     int  `json:"attempts"`
	RetryDelayMS int  `json:"retry_delay_ms"`
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

func endpointHealthPolicy(cfgTimeoutMS, cfgAttempts, cfgRetryDelayMS int, probePage bool, cfgHistoryLimit int) EndpointHealthPolicy {
	policy := EndpointHealthPolicy{
		TimeoutMS:    cfgTimeoutMS,
		Attempts:     cfgAttempts,
		RetryDelayMS: cfgRetryDelayMS,
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
	if policy.RetryDelayMS > maxEndpointHealthRetryDelayMS {
		policy.RetryDelayMS = maxEndpointHealthRetryDelayMS
	}
	if policy.HistoryLimit <= 0 {
		policy.HistoryLimit = defaultSessionEventHistoryLimit
	}
	if policy.HistoryLimit > maxSessionEventHistoryLimit {
		policy.HistoryLimit = maxSessionEventHistoryLimit
	}
	return policy
}

func endpointHealthPolicyFromConfig(raw *config.EndpointHealthConfig) EndpointHealthPolicy {
	if raw == nil {
		return endpointHealthPolicy(0, 0, 0, false, 0)
	}
	return endpointHealthPolicy(raw.TimeoutMS, raw.Attempts, raw.RetryDelayMS, raw.ProbePage, raw.HistoryLimit)
}

func (m *Manager) EndpointHealthPolicy() EndpointHealthPolicy {
	return m.endpointHealthPolicy
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
