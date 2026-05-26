package diagnose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	fluidv1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
)

func TestLoadFAQRulesFromFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "docs", "diagnose-faq.example.yaml")
	catalog, err := LoadFAQCatalog(path)
	if err != nil {
		t.Fatalf("LoadFAQCatalog: %v", err)
	}
	if len(catalog) < 9 {
		t.Fatalf("expected built-in + file rules, got %d", len(catalog))
	}

	ctx := &DiagnosticContext{
		Dataset: DatasetContext{
			Phase:  string(fluidv1alpha1.NotBoundDatasetPhase),
			Mounts: nil,
		},
		Summary: SummaryContext{RuntimeCount: 1},
	}
	matches := matchFAQs(ctx, catalog)
	found := false
	for _, m := range matches {
		if m.ID == "faq-custom-mount-path" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected custom FAQ from file, got %+v", matches)
	}
}

func TestBuildContext_FAQSkip(t *testing.T) {
	t.Parallel()

	ctx, err := BuildContext(BuildContextInput{
		Dataset: &fluidv1alpha1.Dataset{
			Status: fluidv1alpha1.DatasetStatus{Phase: fluidv1alpha1.NotBoundDatasetPhase},
		},
		FAQ: FAQOptions{Skip: true},
	})
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}
	if len(ctx.MatchedFAQs) != 0 {
		t.Fatalf("expected no FAQs when skipped, got %+v", ctx.MatchedFAQs)
	}
}

func TestLoadFAQRulesFromFile_Invalid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("faqs: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := loadFAQRulesFromFile(path)
	if err == nil {
		t.Fatal("expected error for empty faqs")
	}
}

func TestBuildContext_LoadsMarkdownFAQReferences(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "faq.md")
	const markdown = `## 1. Why do I fail to install fluid with Helm?

**Answer**: It is recommended to follow the Fluid installation document.

This may be because Helm below version 3 will not install CRDs automatically.

## 2. Why can't I delete Runtime?

**Answer**: Please check the related Pod running status and Runtime Events.

As long as any active Pod is still using the Volume created by Fluid, Fluid will not complete the delete operation.
`
	if err := os.WriteFile(path, []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, err := BuildContext(BuildContextInput{
		Dataset: &fluidv1alpha1.Dataset{
			Spec: fluidv1alpha1.DatasetSpec{
				Mounts: []fluidv1alpha1.Mount{{MountPoint: "local:///data"}},
			},
			Status: fluidv1alpha1.DatasetStatus{
				Phase: fluidv1alpha1.BoundDatasetPhase,
				Runtimes: []fluidv1alpha1.Runtime{{
					Name:      "demo",
					Namespace: "default",
					Type:      "AlluxioRuntime",
				}},
			},
		},
		FAQ: FAQOptions{File: path},
	})
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}
	if len(ctx.ReferenceFAQs) != 2 {
		t.Fatalf("ReferenceFAQs: got %d want 2", len(ctx.ReferenceFAQs))
	}
	if len(ctx.MatchedFAQs) != 0 {
		t.Fatalf("markdown FAQ must be reference-only, got matches %+v", ctx.MatchedFAQs)
	}

	prompt := FormatPrompt(ctx)
	if !strings.Contains(prompt, "## Reference FAQs") {
		t.Fatalf("prompt missing reference FAQ section:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Why can't I delete Runtime?") {
		t.Fatalf("prompt missing parsed markdown FAQ:\n%s", prompt)
	}
}
