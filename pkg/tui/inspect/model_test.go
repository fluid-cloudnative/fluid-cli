package inspect

import (
	"strings"
	"testing"

	inspectpkg "github.com/fluid-cloudnative/fluid-cli/pkg/inspect"
)

func TestOverviewTextIncludesKeyFields(t *testing.T) {
	report := &inspectpkg.DatasetReport{
		Identity: inspectpkg.DatasetIdentity{
			Name:       "demo",
			Namespace:  "default",
			APIVersion: "data.fluid.io/v1alpha1",
			Kind:       "Dataset",
		},
		Spec: inspectpkg.DatasetSpecSummary{
			Mounts: []inspectpkg.MountSummary{
				{Name: "ufs", MountPoint: "s3://bucket"},
			},
		},
		Status: inspectpkg.DatasetStatusSummary{
			Phase:    "Bound",
			FileNum:  "10",
			UfsTotal: "1Gi",
			Conditions: []inspectpkg.ConditionSummary{
				{Type: "Ready", Status: "True", Reason: "DatasetReady", Message: "healthy"},
			},
			Runtimes: []inspectpkg.RuntimeSummary{
				{Name: "alluxio-demo", Namespace: "default", Category: "Accelerate", Type: "alluxio"},
			},
		},
	}

	text := overviewText(report)
	for _, want := range []string{"default/demo", "phase=Bound", "s3://bucket", "Ready=True", "alluxio-demo"} {
		if !strings.Contains(text, want) {
			t.Fatalf("overview text missing %q:\n%s", want, text)
		}
	}
}
