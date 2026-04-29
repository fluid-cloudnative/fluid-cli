package diagnose

import (
	"os"
	"path/filepath"
	"testing"

	diagpkg "github.com/fluid-cloudnative/fluid-cli/pkg/diagnose"
)

func TestBuildTUIData(t *testing.T) {
	dir := t.TempDir()
	summary := "Dataset: default/demo\nRecent warning events (up to 10):\n- WarningA: pod failed\n"
	manifest := `{"artifacts":[{"path":"summary.txt","status":"collected"},{"path":"pods/p0/logs.txt","status":"failed","reason":"stream error"}]}`

	if err := os.WriteFile(filepath.Join(dir, "summary.txt"), []byte(summary), 0o644); err != nil {
		t.Fatalf("write summary: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	got, err := buildTUIData("demo", "default", &diagpkg.Result{
		OutputPath:          dir,
		ArchivePath:         dir + ".tar.gz",
		PartialFailureCount: 1,
	})
	if err != nil {
		t.Fatalf("buildTUIData returned error: %v", err)
	}

	if got.Dataset != "demo" || got.Namespace != "default" {
		t.Fatalf("unexpected identity: %#v", got)
	}
	if len(got.Artifacts) != 2 {
		t.Fatalf("expected 2 artifact rows, got %d", len(got.Artifacts))
	}
	if len(got.WarningEvents) != 1 {
		t.Fatalf("expected 1 warning event parsed from summary, got %d", len(got.WarningEvents))
	}
}

func TestBuildTUIData_MissingSummary(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(`{"artifacts":[]}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	_, err := buildTUIData("demo", "default", &diagpkg.Result{OutputPath: dir})
	if err == nil {
		t.Fatalf("expected error when summary.txt is missing")
	}
}
