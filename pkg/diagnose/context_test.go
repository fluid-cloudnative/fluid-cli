package diagnose

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	fluidv1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildContext_DeterministicAndGuards(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
	ds := &fluidv1alpha1.Dataset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
			Labels: map[string]string{
				"token": "secret-value",
				"team":  "fluid",
			},
		},
		Status: fluidv1alpha1.DatasetStatus{
			Phase: fluidv1alpha1.BoundDatasetPhase,
			Conditions: []fluidv1alpha1.DatasetCondition{
				{Type: fluidv1alpha1.DatasetReady, Status: corev1.ConditionFalse, Reason: "X", Message: strings.Repeat("m", 600)},
			},
		},
	}

	events := make([]corev1.Event, 0, 60)
	for i := 0; i < 60; i++ {
		events = append(events, corev1.Event{
			Type:   corev1.EventTypeWarning,
			Reason: "Failed",
			Message: strings.Repeat("e", 600),
			LastTimestamp: metav1.Time{Time: now.Add(time.Duration(i) * time.Minute)},
		})
	}

	ctx := BuildContext(BuildContextInput{
		GeneratedAt: now,
		Dataset:     ds,
		Pods: []corev1.Pod{
			{ObjectMeta: metav1.ObjectMeta{Name: "b", Namespace: "default"}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
			{ObjectMeta: metav1.ObjectMeta{Name: "a", Namespace: "default"}, Status: corev1.PodStatus{Phase: corev1.PodPending}},
		},
		Events: events,
	})

	if ctx.Dataset.Labels["token"] != "<redacted>" {
		t.Fatalf("expected redacted token label")
	}
	if len(ctx.Events) != maxPromptEvents {
		t.Fatalf("events: got %d want %d", len(ctx.Events), maxPromptEvents)
	}
	if !strings.Contains(ctx.Events[0].Message, "...(truncated)") {
		t.Fatalf("expected truncated event message")
	}
	if ctx.Pods[0].Name != "a" || ctx.Pods[1].Name != "b" {
		t.Fatalf("pods not sorted: %+v", ctx.Pods)
	}

	b1, err := json.Marshal(ctx)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	b2, err := json.Marshal(BuildContext(BuildContextInput{
		GeneratedAt: now,
		Dataset:     ds,
		Pods: []corev1.Pod{
			{ObjectMeta: metav1.ObjectMeta{Name: "b", Namespace: "default"}, Status: corev1.PodStatus{Phase: corev1.PodRunning}},
			{ObjectMeta: metav1.ObjectMeta{Name: "a", Namespace: "default"}, Status: corev1.PodStatus{Phase: corev1.PodPending}},
		},
		Events: events,
	}))
	if err != nil {
		t.Fatalf("marshal 2: %v", err)
	}
	if string(b1) != string(b2) {
		t.Fatal("expected deterministic JSON output")
	}
}
