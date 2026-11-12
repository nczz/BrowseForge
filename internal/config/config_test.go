package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCloakBrowserConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{
  "port": "19280",
  "profiles_dir": "profiles",
  "data_dir": "data",
  "log_file": "logs/server.log",
  "camoufox_path": "",
  "cloakbrowser_path": "",
  "fingerprint_dir": "data",
  "cloakbrowser": {
    "safe_gpu": true,
    "auto_safe_gpu_fallback": true,
    "isolated_runtime_cache": true,
    "repair_transient_cache_on_launch_failure": true,
    "extra_args": ["--disable-features=Translate"]
  }
}`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.CloakBrowser == nil {
		t.Fatal("cloakbrowser config missing")
	}
	if !cfg.CloakBrowser.SafeGPU ||
		!cfg.CloakBrowser.AutoSafeGPUFallback ||
		!cfg.CloakBrowser.IsolatedRuntimeCache ||
		!cfg.CloakBrowser.RepairTransientCacheOnLaunchFailure {
		t.Fatalf("cloakbrowser config = %#v", cfg.CloakBrowser)
	}
	if len(cfg.CloakBrowser.ExtraArgs) != 1 || cfg.CloakBrowser.ExtraArgs[0] != "--disable-features=Translate" {
		t.Fatalf("extra args = %#v", cfg.CloakBrowser.ExtraArgs)
	}
}

func TestLoadEndpointHealthConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{
  "endpoint_health": {
    "timeout_ms": 15000,
    "attempts": 2,
    "retry_delay_ms": 500,
    "bind_grace_ms": 100,
    "probe_page": false,
    "history_limit": 256
  }
}`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.EndpointHealth == nil {
		t.Fatal("EndpointHealth = nil")
	}
	if cfg.EndpointHealth.TimeoutMS != 15000 ||
		cfg.EndpointHealth.Attempts != 2 ||
		cfg.EndpointHealth.RetryDelayMS != 500 ||
		cfg.EndpointHealth.BindGraceMS != 100 ||
		cfg.EndpointHealth.ProbePage ||
		cfg.EndpointHealth.HistoryLimit != 256 {
		t.Fatalf("EndpointHealth = %#v", cfg.EndpointHealth)
	}
}

func TestLoadDefaultLeavesCloakBrowserPolicyUnset(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("load missing config: %v", err)
	}
	if cfg.CloakBrowser != nil {
		t.Fatalf("default cloakbrowser config = %#v, want nil", cfg.CloakBrowser)
	}
}
