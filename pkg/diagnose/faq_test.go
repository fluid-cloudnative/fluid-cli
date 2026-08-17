package diagnose

import (
	"testing"

	fluidv1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
)

func TestMatchFAQs_DatasetNotBound(t *testing.T) {
	t.Parallel()

	ctx := &DiagnosticContext{
		Dataset: DatasetContext{Phase: string(fluidv1alpha1.NotBoundDatasetPhase)},
		Summary: SummaryContext{RuntimeCount: 1},
	}
	catalog, err := LoadFAQCatalog("")
	if err != nil {
		t.Fatalf("LoadFAQCatalog: %v", err)
	}
	matches := matchFAQs(ctx, catalog)
	if len(matches) == 0 {
		t.Fatal("expected at least one FAQ match")
	}
	found := false
	for _, m := range matches {
		if m.ID == "faq-dataset-not-bound" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected faq-dataset-not-bound, got %+v", matches)
	}
}

func TestMatchFAQs_NoMounts(t *testing.T) {
	t.Parallel()

	ctx := &DiagnosticContext{
		Dataset: DatasetContext{Phase: string(fluidv1alpha1.BoundDatasetPhase), Mounts: nil},
		Summary: SummaryContext{RuntimeCount: 1},
	}
	catalog, err := LoadFAQCatalog("")
	if err != nil {
		t.Fatalf("LoadFAQCatalog: %v", err)
	}
	matches := matchFAQs(ctx, catalog)
	for _, m := range matches {
		if m.ID == "faq-dataset-no-mounts" {
			return
		}
	}
	t.Fatalf("expected faq-dataset-no-mounts, got %+v", matches)
}

func TestBuildContext_AttachesFAQMatches(t *testing.T) {
	t.Parallel()

	ctx, err := BuildContext(BuildContextInput{
		Dataset: &fluidv1alpha1.Dataset{
			Status: fluidv1alpha1.DatasetStatus{
				Phase: fluidv1alpha1.NotBoundDatasetPhase,
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}
	if len(ctx.MatchedFAQs) == 0 {
		t.Fatal("expected MatchedFAQs on context")
	}
}
