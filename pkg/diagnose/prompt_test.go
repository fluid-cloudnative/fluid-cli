package diagnose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	fluidv1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFormatPrompt_Golden(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	ctx, err := BuildContext(BuildContextInput{
		GeneratedAt: now,
		Dataset: &fluidv1alpha1.Dataset{
			ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
			Status:     fluidv1alpha1.DatasetStatus{Phase: fluidv1alpha1.BoundDatasetPhase},
		},
		Pods: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "fuse", Namespace: "default"},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{Name: "fuse", Ready: true, RestartCount: 0},
					},
				},
			},
		},
		Events: []corev1.Event{
			{
				Type:           corev1.EventTypeWarning,
				Reason:         "FailedMount",
				Message:        "mount failed",
				LastTimestamp:  metav1.Time{Time: now},
				InvolvedObject: corev1.ObjectReference{Kind: "Pod", Name: "fuse"},
			},
		},
		FAQ: FAQOptions{Skip: true},
	})
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}

	got := FormatPrompt(ctx)
	golden := filepath.Join("testdata", "prompt_golden.txt")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}

	wantBytes, err := os.ReadFile(golden)
	if err != nil {
		// First run without golden file: assert key sections instead
		for _, section := range []string{
			"Version: fluid-diagnose-prompt/v1",
			"## Instructions",
			"## Dataset",
			"## Pods",
			"## Events",
			"## Summary",
			"Do not provide remediation",
		} {
			if !strings.Contains(got, section) {
				t.Fatalf("prompt missing %q:\n%s", section, got)
			}
		}
		return
	}
	if got != string(wantBytes) {
		t.Fatalf("prompt mismatch with golden file %s (run UPDATE_GOLDEN=1 go test to refresh)", golden)
	}
}
