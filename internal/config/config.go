package config

import (
	"encoding/json"
	"log/slog"
	"os"
)

type Config struct {
	Host             string                `json:"host,omitempty"`
	Port             string                `json:"port"`
	NoSandbox        bool                  `json:"no_sandbox,omitempty"`
	ProfilesDir      string                `json:"profiles_dir"`
	DataDir          string                `json:"data_dir"`
	LogFile          string                `json:"log_file"`
	CamoufoxPath     string                `json:"camoufox_path"`
	CloakBrowserPath string                `json:"cloakbrowser_path"`
	CloakBrowser     *CloakBrowserConfig   `json:"cloakbrowser,omitempty"`
	FingerprintDir   string                `json:"fingerprint_dir"`
	EndpointHealth   *EndpointHealthConfig `json:"endpoint_health,omitempty"`
	Humanize         *HumanizeConfig       `json:"humanize,omitempty"`
	APIToken         string                `json:"-"` // generated at runtime
	Version          string                `json:"-"` // set from main
}

// EndpointHealthConfig controls Playwright Bind endpoint validation and the
// bounded in-memory lifecycle history exposed to operators.
type EndpointHealthConfig struct {
	TimeoutMS    int  `json:"timeout_ms,omitempty"`
	Attempts     int  `json:"attempts,omitempty"`
	RetryDelayMS int  `json:"retry_delay_ms,omitempty"`
	BindGraceMS  int  `json:"bind_grace_ms,omitempty"`
	ProbePage    bool `json:"probe_page,omitempty"`
	HistoryLimit int  `json:"history_limit,omitempty"`
}

// CloakBrowserConfig controls Chromium/CloakBrowser launch compatibility.
// These settings are host runtime policy, not profile identity.
type CloakBrowserConfig struct {
	SafeGPU                             bool     `json:"safe_gpu"`
	AutoSafeGPUFallback                 bool     `json:"auto_safe_gpu_fallback"`
	IsolatedRuntimeCache                bool     `json:"isolated_runtime_cache"`
	RepairTransientCacheOnLaunchFailure bool     `json:"repair_transient_cache_on_launch_failure"`
	ExtraArgs                           []string `json:"extra_args"`
}

// HumanizeConfig controls human-like behavior simulation.
type HumanizeConfig struct {
	Enabled     *bool    `json:"enabled,omitempty"`
	MouseSpeed  string   `json:"mouse_speed,omitempty"`
	TypingCPM   int      `json:"typing_cpm,omitempty"`
	TypoRate    *float64 `json:"typo_rate,omitempty"`
	ScrollStyle string   `json:"scroll_style,omitempty"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		Host:           "127.0.0.1",
		Port:           "19280",
		ProfilesDir:    "profiles",
		DataDir:        "data",
		LogFile:        "logs/server.log",
		FingerprintDir: "data",
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func SetupLogger(logFile string) *slog.Logger {
	os.MkdirAll("logs", 0755)
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}
	return slog.New(slog.NewJSONHandler(f, nil))
}
