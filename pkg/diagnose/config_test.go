package diagnose

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaveUserConfig_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg := UserConfig{
		Diagnose: DiagnoseConfig{
			LLM: LLMConfig{Endpoint: "https://api.example.com/v1"},
		},
	}
	if err := SaveUserConfig(cfg); err != nil {
		t.Fatalf("SaveUserConfig: %v", err)
	}

	loaded, err := LoadUserConfig()
	if err != nil {
		t.Fatalf("LoadUserConfig: %v", err)
	}
	if loaded.Diagnose.LLM.Endpoint != cfg.Diagnose.LLM.Endpoint {
		t.Fatalf("endpoint: got %q want %q", loaded.Diagnose.LLM.Endpoint, cfg.Diagnose.LLM.Endpoint)
	}

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode: got %o want 0600", info.Mode().Perm())
	}
}

func TestResolveLLMSettings_Precedence(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	if err := SaveUserConfig(UserConfig{
		Diagnose: DiagnoseConfig{LLM: LLMConfig{Endpoint: "https://file.example/v1"}},
	}); err != nil {
		t.Fatalf("SaveUserConfig: %v", err)
	}

	t.Setenv(envLLMEndpoint, "https://env.example/v1")
	t.Setenv(envLLMAPIKey, "env-key")
	settings, err := ResolveLLMSettings("", "", false, false)
	if err != nil {
		t.Fatalf("ResolveLLMSettings env: %v", err)
	}
	if settings.Endpoint != "https://env.example/v1" {
		t.Fatalf("env endpoint: got %q", settings.Endpoint)
	}
	if settings.APIKey != "env-key" {
		t.Fatalf("env api key: got %q", settings.APIKey)
	}
	if settings.Skip {
		t.Fatal("expected skip=false when endpoint configured without --llm-skip")
	}

	t.Setenv(envLLMAPIKey, "")
	_, err = ResolveLLMSettings("https://flag.example/v1", "", false, false)
	if err == nil {
		t.Fatal("expected error when endpoint set without API key")
	}

	settings, err = ResolveLLMSettings("https://flag.example/v1", "", false, true)
	if err == nil {
		t.Fatal("expected error when skip=false without API key")
	}
	_ = settings

	settings, err = ResolveLLMSettings("https://flag.example/v1", "custom-model", true, true)
	if err != nil {
		t.Fatalf("ResolveLLMSettings explicit skip: %v", err)
	}
	if !settings.Skip {
		t.Fatal("expected skip=true when --llm-skip=true")
	}
	if settings.Model != "custom-model" {
		t.Fatalf("model: got %q", settings.Model)
	}
}

func TestValidateLLMEndpoint(t *testing.T) {
	t.Parallel()

	cases := []struct {
		endpoint string
		wantErr  bool
	}{
		{"https://api.example.com/v1", false},
		{"http://localhost:8080", false},
		{"", true},
		{"ftp://bad.example", true},
		{"not-a-url", true},
	}
	for _, tc := range cases {
		err := ValidateLLMEndpoint(tc.endpoint)
		if tc.wantErr && err == nil {
			t.Fatalf("expected error for %q", tc.endpoint)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("unexpected error for %q: %v", tc.endpoint, err)
		}
	}
}

func TestSetGetUnsetLLMEndpoint(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	if err := SetLLMEndpoint("https://api.example.com/v1"); err != nil {
		t.Fatalf("SetLLMEndpoint: %v", err)
	}
	got, err := GetLLMEndpoint()
	if err != nil {
		t.Fatalf("GetLLMEndpoint: %v", err)
	}
	if got != "https://api.example.com/v1" {
		t.Fatalf("got %q", got)
	}

	if err := UnsetLLMEndpoint(); err != nil {
		t.Fatalf("UnsetLLMEndpoint: %v", err)
	}
	got, err = GetLLMEndpoint()
	if err != nil {
		t.Fatalf("GetLLMEndpoint after unset: %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty endpoint, got %q", got)
	}

	path, _ := ConfigPath()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file should exist: %v", err)
	}
	if filepath.Base(path) != "config" {
		t.Fatalf("unexpected config basename: %s", path)
	}
}
