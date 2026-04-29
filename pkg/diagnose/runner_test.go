package diagnose

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	fluidv1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubefake "k8s.io/client-go/kubernetes/fake"
	ctrlclientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRun_ReturnsErrorWhenManifestWriteFails(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	if err := fluidv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add fluid scheme: %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add corev1 scheme: %v", err)
	}

	dataset := &fluidv1alpha1.Dataset{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "demo",
			Namespace: "default",
		},
	}

	c := ctrlclientfake.NewClientBuilder().WithScheme(scheme).WithObjects(dataset).Build()
	kubeClient := kubefake.NewSimpleClientset()
	runner := NewRunner(c, kubeClient)

	outputDir := t.TempDir()
	manifestPath := filepath.Join(outputDir, "manifest.json")
	if err := os.MkdirAll(manifestPath, 0o755); err != nil {
		t.Fatalf("create conflicting manifest path: %v", err)
	}

	_, err := runner.Run(context.Background(), Options{
		DatasetName: "demo",
		Namespace:   "default",
		Output:      "dir",
		OutputDir:   outputDir,
		NoLogs:      true,
	})
	if err == nil {
		t.Fatalf("expected error when manifest path is not writable")
	}
	if !strings.Contains(err.Error(), "writing manifest") {
		t.Fatalf("expected writing manifest error, got: %v", err)
	}
}

type errorReadCloser struct {
	data []byte
	err  error
	read bool
}

func (r *errorReadCloser) Read(p []byte) (int, error) {
	if !r.read {
		r.read = true
		n := copy(p, r.data)
		return n, r.err
	}
	return 0, io.EOF
}

func (r *errorReadCloser) Close() error {
	return nil
}

func TestCopyPodLog_AppendsCopyError(t *testing.T) {
	t.Parallel()

	var b strings.Builder
	stream := &errorReadCloser{
		data: []byte("partial-log"),
		err:  errors.New("stream interrupted"),
	}

	copyPodLog(&b, stream)

	got := b.String()
	if !strings.Contains(got, "partial-log") {
		t.Fatalf("expected partial log content, got: %q", got)
	}
	if !strings.Contains(got, "log copy error: stream interrupted") {
		t.Fatalf("expected copy error annotation, got: %q", got)
	}
}

func TestCollectEvents_WithSince_KeepsZeroTimestampEvents(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
	zeroTsEvent := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "unknown-ts",
			Namespace: "default",
		},
		Message: "timestamp unavailable",
	}
	oldEvent := &corev1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "old-ts",
			Namespace: "default",
		},
		LastTimestamp: metav1.NewTime(now.Add(-2 * time.Hour)),
		Message:       "too old",
	}

	kubeClient := kubefake.NewSimpleClientset(zeroTsEvent, oldEvent)
	runner := &Runner{
		kubeClient: kubeClient,
		nowFn:      func() time.Time { return now },
	}

	outputDir := t.TempDir()
	m := &manifest{}

	runner.collectEvents(context.Background(), outputDir, Options{
		Namespace: "default",
		Since:     "1h",
	}, m)

	eventsYAML, err := os.ReadFile(filepath.Join(outputDir, "events", "events.yaml"))
	if err != nil {
		t.Fatalf("read events yaml: %v", err)
	}

	got := string(eventsYAML)
	if !strings.Contains(got, "unknown-ts") {
		t.Fatalf("expected zero timestamp event to be kept, got: %s", got)
	}
	if strings.Contains(got, "old-ts") {
		t.Fatalf("expected old event to be filtered out, got: %s", got)
	}
}

func TestCollectPod_NoLogsPathIncludesNamespace(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add corev1 scheme: %v", err)
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shared-pod",
			Namespace: "ns-a",
		},
	}

	c := ctrlclientfake.NewClientBuilder().WithScheme(scheme).WithObjects(pod).Build()
	runner := &Runner{
		client:     c,
		kubeClient: kubefake.NewSimpleClientset(),
		nowFn:      time.Now,
	}

	outputDir := t.TempDir()
	m := &manifest{}
	runner.collectPod(context.Background(), outputDir, "ns-a", "shared-pod", Options{NoLogs: true}, m)

	var paths []string
	for _, a := range m.Artifacts {
		paths = append(paths, a.Path)
	}
	wantPaths := []string{
		"pods/ns-a/shared-pod/pod.yaml",
		"pods/ns-a/shared-pod/describe.txt",
		"pods/ns-a/shared-pod/logs.txt",
	}
	for _, want := range wantPaths {
		found := false
		for _, got := range paths {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected manifest artifact path %q, got paths: %v", want, paths)
		}
	}
}

func TestCollectPVC_PathIncludesNamespace(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add corev1 scheme: %v", err)
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "shared-pvc",
			Namespace: "ns-a",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			VolumeName: "pv-a",
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}

	c := ctrlclientfake.NewClientBuilder().WithScheme(scheme).WithObjects(pvc).Build()
	runner := &Runner{
		client: c,
		nowFn:  time.Now,
	}

	outputDir := t.TempDir()
	m := &manifest{}
	runner.collectPVC(context.Background(), outputDir, "ns-a", "shared-pvc", m)

	var paths []string
	for _, a := range m.Artifacts {
		paths = append(paths, a.Path)
	}
	wantPaths := []string{
		"storage/pvc-ns-a-shared-pvc.yaml",
		"storage/pvc-ns-a-shared-pvc.describe.txt",
	}
	for _, want := range wantPaths {
		found := false
		for _, got := range paths {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected manifest artifact path %q, got paths: %v", want, paths)
		}
	}
}
