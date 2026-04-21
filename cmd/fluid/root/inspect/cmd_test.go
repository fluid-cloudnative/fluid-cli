package inspect

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	inspectpkg "github.com/fluid-cloudnative/fluid-cli/pkg/inspect"
	fluidscheme "github.com/fluid-cloudnative/fluid-cli/pkg/scheme"
	fluidv1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRenderInspectOutput_UsesTablePath(t *testing.T) {
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	report := &inspectpkg.DatasetReport{
		Identity: inspectpkg.DatasetIdentity{Name: "demo", Namespace: "default"},
		Status:   inspectpkg.DatasetStatusSummary{Phase: "Bound"},
	}

	if err := renderInspectOutput(cmd, report, "table", false); err != nil {
		t.Fatalf("renderInspectOutput returned error: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatalf("expected table output to be written")
	}
}

func TestRenderInspectOutput_InvalidFormat(t *testing.T) {
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err := renderInspectOutput(cmd, &inspectpkg.DatasetReport{}, "xml", false)
	if err == nil {
		t.Fatalf("expected invalid format error")
	}
}

func TestListDatasetNames_Sorted(t *testing.T) {
	c := fake.NewClientBuilder().
		WithScheme(fluidscheme.Scheme).
		WithObjects(
			&fluidv1alpha1.Dataset{ObjectMeta: metav1.ObjectMeta{Name: "zeta", Namespace: "default"}},
			&fluidv1alpha1.Dataset{ObjectMeta: metav1.ObjectMeta{Name: "alpha", Namespace: "default"}},
			&fluidv1alpha1.Dataset{ObjectMeta: metav1.ObjectMeta{Name: "beta", Namespace: "other"}},
		).
		Build()

	got, err := listDatasetNames(context.Background(), c, "default")
	if err != nil {
		t.Fatalf("listDatasetNames returned error: %v", err)
	}
	want := []string{"alpha", "zeta"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected dataset names: got=%v want=%v", got, want)
	}
}

func TestListDatasetNames_NoDatasets(t *testing.T) {
	c := fake.NewClientBuilder().
		WithScheme(fluidscheme.Scheme).
		Build()

	_, err := listDatasetNames(context.Background(), c, "default")
	if err == nil {
		t.Fatalf("expected error when no datasets exist")
	}
	if !strings.Contains(err.Error(), "no datasets found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIsClusterConnectivityError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "tls timeout", err: assertErr("Get https://127.0.0.1:6443/api: net/http: TLS handshake timeout"), want: true},
		{name: "connection refused", err: assertErr("dial tcp 127.0.0.1:6443: connect: connection refused"), want: true},
		{name: "other error", err: assertErr("dataset not found"), want: false},
	}
	for _, tt := range tests {
		if got := isClusterConnectivityError(tt.err); got != tt.want {
			t.Fatalf("%s: got=%v want=%v", tt.name, got, tt.want)
		}
	}
}

func assertErr(msg string) error { return simpleErr(msg) }

type simpleErr string

func (e simpleErr) Error() string { return string(e) }
