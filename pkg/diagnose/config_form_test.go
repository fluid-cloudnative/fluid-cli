package diagnose

import (
	"os"
	"testing"
)

func TestApplyLLMFormValues(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	if err := ApplyLLMFormValues(LLMFormValues{
		Endpoint: "https://api.example.com/v1",
		APIKey:   "secret-key",
		Model:    "test-model",
	}); err != nil {
		t.Fatalf("ApplyLLMFormValues: %v", err)
	}

	endpoint, err := GetLLMEndpoint()
	if err != nil {
		t.Fatalf("GetLLMEndpoint: %v", err)
	}
	if endpoint != "https://api.example.com/v1" {
		t.Fatalf("endpoint: got %q", endpoint)
	}
	ok, err := GetLLMAPIKeyConfigured()
	if err != nil || !ok {
		t.Fatalf("api key configured: ok=%v err=%v", ok, err)
	}
	model, err := GetLLMModel()
	if err != nil {
		t.Fatalf("GetLLMModel: %v", err)
	}
	if model != "test-model" {
		t.Fatalf("model: got %q", model)
	}

	if err := ApplyLLMFormValues(LLMFormValues{
		Endpoint: "https://api.example.com/v1",
		Model:    "other-model",
	}); err != nil {
		t.Fatalf("ApplyLLMFormValues without key: %v", err)
	}
	cfg, err := LoadUserConfig()
	if err != nil {
		t.Fatalf("LoadUserConfig: %v", err)
	}
	if cfg.Diagnose.LLM.APIKey != "secret-key" {
		t.Fatalf("api key should be unchanged, got %q", cfg.Diagnose.LLM.APIKey)
	}
}

func TestLoadLLMFormDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv(envLLMAPIKey, "")

	if err := SetLLMEndpoint("https://saved.example/v1"); err != nil {
		t.Fatalf("SetLLMEndpoint: %v", err)
	}

	def, err := LoadLLMFormDefaults()
	if err != nil {
		t.Fatalf("LoadLLMFormDefaults: %v", err)
	}
	if def.Endpoint != "https://saved.example/v1" {
		t.Fatalf("endpoint: got %q", def.Endpoint)
	}
	if def.ConfigPath == "" {
		t.Fatal("expected config path")
	}
	if _, err := os.Stat(def.ConfigPath); err != nil {
		t.Fatalf("stat config: %v", err)
	}
}
