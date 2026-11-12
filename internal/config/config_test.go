package config

import (
	"os"
	"testing"
)

func TestLoadAppliesPublicBaseURLEnvWhenConfigMissing(t *testing.T) {
	t.Setenv("BROWSEFORGE_PUBLIC_BASE_URL", "https://bf.example.com/root/")
	cfg, err := Load(t.TempDir() + "/missing.json")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PublicBaseURL != "https://bf.example.com/root" {
		t.Fatalf("PublicBaseURL = %q", cfg.PublicBaseURL)
	}
}

func TestLoadTrimsPublicBaseURLFromConfig(t *testing.T) {
	path := t.TempDir() + "/config.json"
	if err := os.WriteFile(path, []byte(`{"public_base_url":" https://bf.example.com/root/ "}`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PublicBaseURL != "https://bf.example.com/root" {
		t.Fatalf("PublicBaseURL = %q", cfg.PublicBaseURL)
	}
}

func TestLoadEndpointHealthConfig(t *testing.T) {
	path := t.TempDir() + "/config.json"
	if err := os.WriteFile(path, []byte(`{
		"endpoint_health": {
			"timeout_ms": 15000,
			"attempts": 2,
			"retry_delay_ms": 500,
			"bind_grace_ms": 100,
			"probe_page": false,
			"history_limit": 256
		}
	}`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
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
