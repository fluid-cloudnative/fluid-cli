package diagnose

import (
	"bytes"
	"os"
	"testing"

	diagpkg "github.com/fluid-cloudnative/fluid-cli/pkg/diagnose"
)

func TestDiagnoseConfig_SetAndGet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	root := bytes.NewBuffer(nil)
	setCmd := newConfigSetCommand()
	setCmd.SetOut(root)
	setCmd.SetErr(root)
	setCmd.SetArgs([]string{"llm-endpoint", "https://api.example.com/v1"})
	if err := setCmd.Execute(); err != nil {
		t.Fatalf("config set: %v", err)
	}

	got, err := diagpkg.GetLLMEndpoint()
	if err != nil {
		t.Fatalf("GetLLMEndpoint: %v", err)
	}
	if got != "https://api.example.com/v1" {
		t.Fatalf("got %q", got)
	}

	out := &bytes.Buffer{}
	getCmd := newConfigGetCommand()
	getCmd.SetOut(out)
	getCmd.SetArgs([]string{"llm-endpoint"})
	if err := getCmd.Execute(); err != nil {
		t.Fatalf("config get: %v", err)
	}
	if out.String() != "https://api.example.com/v1\n" {
		t.Fatalf("get output: %q", out.String())
	}

	path, err := diagpkg.ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file missing: %v", err)
	}
}
