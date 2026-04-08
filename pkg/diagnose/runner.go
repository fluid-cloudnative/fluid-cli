// Copyright 2026 The Fluid Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package diagnose

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fluid-cloudnative/fluid-cli/pkg/inspect"
	fluidv1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
	"github.com/fluid-cloudnative/fluid/pkg/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type Options struct {
	DatasetName           string
	Namespace             string
	Output                string
	Archive               bool
	OutputDir             string
	NoLogs                bool
	IncludeControllerLogs bool
	Since                 string
}

type Result struct {
	OutputPath          string
	ArchivePath         string
	PartialFailureCount int
	Stdout              string
}

type Runner struct {
	client     client.Client
	kubeClient kubernetes.Interface
	nowFn      func() time.Time
}

type manifest struct {
	Dataset    string          `json:"dataset"`
	Namespace  string          `json:"namespace"`
	Generated  string          `json:"generated"`
	Artifacts  []manifestEntry `json:"artifacts"`
	Failures   int             `json:"failures"`
	ArchivedTo string          `json:"archivedTo,omitempty"`
}

type manifestEntry struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func NewRunner(c client.Client, kubeClient kubernetes.Interface) *Runner {
	return &Runner{
		client:     c,
		kubeClient: kubeClient,
		nowFn:      time.Now,
	}
}

func (r *Runner) Run(ctx context.Context, opts Options) (*Result, error) {
	if opts.DatasetName == "" {
		return nil, fmt.Errorf("dataset name is required")
	}
	if opts.Namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if opts.Output == "" {
		opts.Output = "dir"
	}
	if opts.Output != "dir" && opts.Output != "stdout" {
		return nil, fmt.Errorf("invalid output mode %q, expected dir|stdout", opts.Output)
	}
	if opts.Output == "stdout" && opts.Archive {
		return nil, fmt.Errorf("--archive cannot be used with --output=stdout")
	}

	if opts.Output == "stdout" {
		return r.runStdout(ctx, opts)
	}

	baseDir := opts.OutputDir
	if baseDir == "" {
		baseDir = fmt.Sprintf("fluid-diagnose-%s-%s", opts.DatasetName, r.nowFn().Format("20060102-150405"))
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}

	m := &manifest{
		Dataset:   opts.DatasetName,
		Namespace: opts.Namespace,
		Generated: r.nowFn().UTC().Format(time.RFC3339),
	}

	dataset := &fluidv1alpha1.Dataset{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: opts.DatasetName, Namespace: opts.Namespace}, dataset); err != nil {
		return nil, fmt.Errorf("getting dataset %q in namespace %q: %w", opts.DatasetName, opts.Namespace, err)
	}
	r.writeYAML(baseDir, "dataset.yaml", dataset, m)
	r.writeText(baseDir, "dataset.describe.txt", describeDataset(dataset), m)

	for _, rt := range dataset.Status.Runtimes {
		runtimeNS := rt.Namespace
		if runtimeNS == "" {
			runtimeNS = opts.Namespace
		}
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "data.fluid.io",
			Version: "v1alpha1",
			Kind:    rt.Type,
		})
		rtPathPrefix := fmt.Sprintf("runtimes/runtime-%s-%s", strings.ToLower(strings.TrimSuffix(rt.Type, "Runtime")), rt.Name)
		if err := r.client.Get(ctx, types.NamespacedName{Name: rt.Name, Namespace: runtimeNS}, u); err != nil {
			m.Artifacts = append(m.Artifacts, manifestEntry{
				Path:   rtPathPrefix + ".yaml",
				Status: "failed",
				Reason: fmt.Sprintf("runtime get failed: %v", err),
			})
			continue
		}
		r.writeYAML(baseDir, rtPathPrefix+".yaml", u.Object, m)
		r.writeText(baseDir, rtPathPrefix+".describe.txt", describeUnstructuredRuntime(u), m)
	}

	inspector := inspect.New(r.client)
	report, err := inspector.Run(ctx, opts.DatasetName, opts.Namespace)
	if err != nil {
		return nil, fmt.Errorf("discovering associated resources: %w", err)
	}

	seenPods := map[string]struct{}{}
	seenPVCs := map[string]struct{}{}
	seenPVs := map[string]struct{}{}

	for _, rr := range report.Runtimes {
		for _, row := range rr.Resources {
			switch row.Kind {
			case "Pod":
				key := row.Namespace + "/" + row.Name
				if _, ok := seenPods[key]; ok {
					continue
				}
				seenPods[key] = struct{}{}
				r.collectPod(ctx, baseDir, row.Namespace, row.Name, opts, m)
			case "PVC":
				key := row.Namespace + "/" + row.Name
				if _, ok := seenPVCs[key]; ok {
					continue
				}
				seenPVCs[key] = struct{}{}
				r.collectPVC(ctx, baseDir, row.Namespace, row.Name, m)
			case "PV":
				if _, ok := seenPVs[row.Name]; ok {
					continue
				}
				seenPVs[row.Name] = struct{}{}
				r.collectPV(ctx, baseDir, row.Name, m)
			}
		}
	}

	// Ensure canonical PV/PVC candidates are attempted even when not discovered.
	r.collectPVC(ctx, baseDir, opts.Namespace, opts.DatasetName, m)
	r.collectPV(ctx, baseDir, fmt.Sprintf("%s-%s", opts.Namespace, opts.DatasetName), m)

	r.collectEvents(ctx, baseDir, opts, m)
	if opts.IncludeControllerLogs && !opts.NoLogs {
		r.collectControllerLogs(ctx, baseDir, opts, m)
	}

	summary := buildSummary(dataset, report, m)
	r.writeText(baseDir, "summary.txt", summary, m)

	failures := 0
	for _, a := range m.Artifacts {
		if a.Status == "failed" {
			failures++
		}
	}
	m.Failures = failures

	manifestBytes, _ := json.MarshalIndent(m, "", "  ")
	_ = os.WriteFile(filepath.Join(baseDir, "manifest.json"), manifestBytes, 0o644)

	result := &Result{
		OutputPath:          baseDir,
		PartialFailureCount: failures,
	}
	if opts.Archive {
		archivePath := baseDir + ".tar.gz"
		if err := tarGzDir(baseDir, archivePath); err != nil {
			return nil, fmt.Errorf("creating archive: %w", err)
		}
		result.ArchivePath = archivePath
	}
	return result, nil
}

func (r *Runner) runStdout(ctx context.Context, opts Options) (*Result, error) {
	dataset := &fluidv1alpha1.Dataset{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: opts.DatasetName, Namespace: opts.Namespace}, dataset); err != nil {
		return nil, fmt.Errorf("getting dataset %q in namespace %q: %w", opts.DatasetName, opts.Namespace, err)
	}

	inspector := inspect.New(r.client)
	report, err := inspector.Run(ctx, opts.DatasetName, opts.Namespace)
	if err != nil {
		return nil, fmt.Errorf("discovering associated resources: %w", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Diagnose Report\n")
	fmt.Fprintf(&b, "Dataset: %s/%s\n", opts.Namespace, opts.DatasetName)
	fmt.Fprintf(&b, "Phase: %s\n", dataset.Status.Phase)
	fmt.Fprintf(&b, "Runtimes: %d\n", len(dataset.Status.Runtimes))
	for _, rt := range dataset.Status.Runtimes {
		fmt.Fprintf(&b, "  - %s (%s) ns=%s\n", rt.Name, rt.Type, defaultNS(rt.Namespace, opts.Namespace))
	}
	if len(dataset.Status.Conditions) > 0 {
		fmt.Fprintf(&b, "\nConditions:\n")
		for _, c := range dataset.Status.Conditions {
			fmt.Fprintf(&b, "  - %s=%s reason=%s message=%s\n", c.Type, c.Status, c.Reason, c.Message)
		}
	}

	podPhaseCounts := map[string]int{}
	totalPods := 0
	totalPVC := 0
	totalPV := 0
	for _, rr := range report.Runtimes {
		for _, row := range rr.Resources {
			switch row.Kind {
			case "Pod":
				podPhaseCounts[row.Status]++
				totalPods++
			case "PVC":
				totalPVC++
			case "PV":
				totalPV++
			}
		}
	}
	fmt.Fprintf(&b, "\nAssociated resources:\n")
	fmt.Fprintf(&b, "  Pods: %d\n", totalPods)
	if len(podPhaseCounts) > 0 {
		keys := make([]string, 0, len(podPhaseCounts))
		for k := range podPhaseCounts {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "    - %s: %d\n", k, podPhaseCounts[k])
		}
	}
	fmt.Fprintf(&b, "  PVCs: %d\n", totalPVC)
	fmt.Fprintf(&b, "  PVs: %d\n", totalPV)

	evList, err := r.kubeClient.CoreV1().Events(opts.Namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		sinceCutoff := time.Time{}
		if opts.Since != "" {
			if d, parseErr := time.ParseDuration(opts.Since); parseErr == nil {
				sinceCutoff = r.nowFn().Add(-d)
			}
		}
		var warnings []corev1.Event
		for _, ev := range evList.Items {
			if ev.Type != corev1.EventTypeWarning {
				continue
			}
			t := ev.LastTimestamp.Time
			if t.IsZero() {
				t = ev.EventTime.Time
			}
			if !sinceCutoff.IsZero() && !t.IsZero() && t.Before(sinceCutoff) {
				continue
			}
			warnings = append(warnings, ev)
		}
		sort.Slice(warnings, func(i, j int) bool {
			return warnings[i].LastTimestamp.Time.After(warnings[j].LastTimestamp.Time)
		})
		fmt.Fprintf(&b, "\nRecent warning events (up to 10):\n")
		if len(warnings) == 0 {
			fmt.Fprintf(&b, "  <none>\n")
		} else {
			limit := 10
			if len(warnings) < limit {
				limit = len(warnings)
			}
			for i := 0; i < limit; i++ {
				w := warnings[i]
				obj := ""
				if w.InvolvedObject.Kind != "" {
					obj = fmt.Sprintf(" [%s/%s]", w.InvolvedObject.Kind, w.InvolvedObject.Name)
				}
				fmt.Fprintf(&b, "  - %s: %s%s\n", w.Reason, w.Message, obj)
			}
		}
	}

	return &Result{
		Stdout: b.String(),
	}, nil
}

func (r *Runner) collectPod(ctx context.Context, baseDir, namespace, name string, opts Options, m *manifest) {
	pod := &corev1.Pod{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, pod); err != nil {
		m.Artifacts = append(m.Artifacts, manifestEntry{Path: "pods/" + name + "/pod.yaml", Status: "failed", Reason: err.Error()})
		return
	}
	r.writeYAML(baseDir, fmt.Sprintf("pods/%s/pod.yaml", name), pod, m)
	r.writeText(baseDir, fmt.Sprintf("pods/%s/describe.txt", name), describePod(pod), m)

	if opts.NoLogs {
		m.Artifacts = append(m.Artifacts, manifestEntry{Path: fmt.Sprintf("pods/%s/logs.txt", name), Status: "skipped", Reason: "--no-logs enabled"})
		return
	}

	var sincePtr *metav1.Time
	if opts.Since != "" {
		d, err := time.ParseDuration(opts.Since)
		if err == nil {
			t := metav1.NewTime(r.nowFn().Add(-d))
			sincePtr = &t
		}
	}

	var b strings.Builder
	for _, c := range pod.Spec.Containers {
		b.WriteString("=== container: " + c.Name + " ===\n")
		req := r.kubeClient.CoreV1().Pods(namespace).GetLogs(name, &corev1.PodLogOptions{
			Container: c.Name,
			SinceTime: sincePtr,
		})
		stream, err := req.Stream(ctx)
		if err != nil {
			b.WriteString("log stream error: " + err.Error() + "\n\n")
			continue
		}
		_, _ = io.Copy(&b, stream)
		_ = stream.Close()
		b.WriteString("\n")
	}
	r.writeText(baseDir, fmt.Sprintf("pods/%s/logs.txt", name), b.String(), m)
}

func (r *Runner) collectPVC(ctx context.Context, baseDir, namespace, name string, m *manifest) {
	pvc := &corev1.PersistentVolumeClaim{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, pvc); err != nil {
		m.Artifacts = append(m.Artifacts, manifestEntry{Path: fmt.Sprintf("storage/pvc-%s.yaml", name), Status: "failed", Reason: err.Error()})
		return
	}
	r.writeYAML(baseDir, fmt.Sprintf("storage/pvc-%s.yaml", name), pvc, m)
	r.writeText(baseDir, fmt.Sprintf("storage/pvc-%s.describe.txt", name), describePVC(pvc), m)
}

func (r *Runner) collectPV(ctx context.Context, baseDir, name string, m *manifest) {
	pv := &corev1.PersistentVolume{}
	if err := r.client.Get(ctx, types.NamespacedName{Name: name}, pv); err != nil {
		m.Artifacts = append(m.Artifacts, manifestEntry{Path: fmt.Sprintf("storage/pv-%s.yaml", name), Status: "failed", Reason: err.Error()})
		return
	}
	r.writeYAML(baseDir, fmt.Sprintf("storage/pv-%s.yaml", name), pv, m)
	r.writeText(baseDir, fmt.Sprintf("storage/pv-%s.describe.txt", name), describePV(pv), m)
}

func (r *Runner) collectEvents(ctx context.Context, baseDir string, opts Options, m *manifest) {
	evList, err := r.kubeClient.CoreV1().Events(opts.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		m.Artifacts = append(m.Artifacts, manifestEntry{Path: "events/events.yaml", Status: "failed", Reason: err.Error()})
		return
	}
	items := make([]corev1.Event, 0, len(evList.Items))
	sinceCutoff := time.Time{}
	if opts.Since != "" {
		if d, err := time.ParseDuration(opts.Since); err == nil {
			sinceCutoff = r.nowFn().Add(-d)
		}
	}
	for _, ev := range evList.Items {
		if !sinceCutoff.IsZero() && ev.LastTimestamp.Time.Before(sinceCutoff) && ev.EventTime.Time.Before(sinceCutoff) {
			continue
		}
		items = append(items, ev)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].LastTimestamp.Time.Before(items[j].LastTimestamp.Time)
	})
	r.writeYAML(baseDir, "events/events.yaml", map[string]any{"items": items}, m)
}

func (r *Runner) collectControllerLogs(ctx context.Context, baseDir string, opts Options, m *manifest) {
	pods, err := r.kubeClient.CoreV1().Pods(common.NamespaceFluidSystem).List(ctx, metav1.ListOptions{
		LabelSelector: "control-plane in (dataset-controller,runtime-controller)",
	})
	if err != nil {
		m.Artifacts = append(m.Artifacts, manifestEntry{Path: "controllers", Status: "failed", Reason: err.Error()})
		return
	}
	for _, pod := range pods.Items {
		req := r.kubeClient.CoreV1().Pods(common.NamespaceFluidSystem).GetLogs(pod.Name, &corev1.PodLogOptions{})
		stream, err := req.Stream(ctx)
		if err != nil {
			m.Artifacts = append(m.Artifacts, manifestEntry{Path: fmt.Sprintf("controllers/%s.log", pod.Name), Status: "failed", Reason: err.Error()})
			continue
		}
		defer stream.Close()
		b, err := io.ReadAll(stream)
		if err != nil {
			m.Artifacts = append(m.Artifacts, manifestEntry{Path: fmt.Sprintf("controllers/%s.log", pod.Name), Status: "failed", Reason: err.Error()})
			continue
		}
		r.writeText(baseDir, fmt.Sprintf("controllers/%s.log", pod.Name), string(b), m)
	}
}

func (r *Runner) writeYAML(baseDir, rel string, obj any, m *manifest) {
	b, err := yaml.Marshal(obj)
	if err != nil {
		m.Artifacts = append(m.Artifacts, manifestEntry{Path: rel, Status: "failed", Reason: fmt.Sprintf("yaml marshal error: %v", err)})
		return
	}
	r.writeFile(baseDir, rel, b, m)
}

func (r *Runner) writeText(baseDir, rel, content string, m *manifest) {
	r.writeFile(baseDir, rel, []byte(content), m)
}

func (r *Runner) writeFile(baseDir, rel string, content []byte, m *manifest) {
	path := filepath.Join(baseDir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		m.Artifacts = append(m.Artifacts, manifestEntry{Path: rel, Status: "failed", Reason: err.Error()})
		return
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		m.Artifacts = append(m.Artifacts, manifestEntry{Path: rel, Status: "failed", Reason: err.Error()})
		return
	}
	m.Artifacts = append(m.Artifacts, manifestEntry{Path: rel, Status: "collected"})
}

func describeDataset(ds *fluidv1alpha1.Dataset) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Name: %s\nNamespace: %s\nPhase: %s\n", ds.Name, ds.Namespace, ds.Status.Phase)
	for _, c := range ds.Status.Conditions {
		fmt.Fprintf(&b, "Condition %s=%s reason=%s message=%s\n", c.Type, c.Status, c.Reason, c.Message)
	}
	return b.String()
}

func describeUnstructuredRuntime(u *unstructured.Unstructured) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Name: %s\nNamespace: %s\nKind: %s\n", u.GetName(), u.GetNamespace(), u.GetKind())
	if phase, ok, _ := unstructured.NestedString(u.Object, "status", "phase"); ok {
		fmt.Fprintf(&b, "Phase: %s\n", phase)
	}
	return b.String()
}

func describePod(pod *corev1.Pod) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Name: %s\nNamespace: %s\nNode: %s\nPhase: %s\n", pod.Name, pod.Namespace, pod.Spec.NodeName, pod.Status.Phase)
	for _, cs := range pod.Status.ContainerStatuses {
		fmt.Fprintf(&b, "Container %s ready=%v restarts=%d\n", cs.Name, cs.Ready, cs.RestartCount)
	}
	return b.String()
}

func describePVC(pvc *corev1.PersistentVolumeClaim) string {
	return fmt.Sprintf("Name: %s\nNamespace: %s\nPhase: %s\nVolume: %s\n", pvc.Name, pvc.Namespace, pvc.Status.Phase, pvc.Spec.VolumeName)
}

func describePV(pv *corev1.PersistentVolume) string {
	claim := ""
	if pv.Spec.ClaimRef != nil {
		claim = pv.Spec.ClaimRef.Namespace + "/" + pv.Spec.ClaimRef.Name
	}
	return fmt.Sprintf("Name: %s\nPhase: %s\nClaim: %s\nStorageClass: %s\n", pv.Name, pv.Status.Phase, claim, pv.Spec.StorageClassName)
}

func buildSummary(ds *fluidv1alpha1.Dataset, report *inspect.DatasetReport, m *manifest) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Dataset: %s/%s\n", ds.Namespace, ds.Name)
	fmt.Fprintf(&b, "Dataset phase: %s\n", ds.Status.Phase)
	fmt.Fprintf(&b, "Runtimes: %d\n", len(report.Runtimes))

	podPhaseCounts := map[string]int{}
	for _, rt := range report.Runtimes {
		for _, row := range rt.Resources {
			if row.Kind == "Pod" {
				podPhaseCounts[row.Status]++
			}
		}
	}
	if len(podPhaseCounts) > 0 {
		fmt.Fprintf(&b, "Pod status counts:\n")
		keys := make([]string, 0, len(podPhaseCounts))
		for k := range podPhaseCounts {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "  - %s: %d\n", k, podPhaseCounts[k])
		}
	}

	failures := 0
	for _, a := range m.Artifacts {
		if a.Status == "failed" {
			failures++
		}
	}
	fmt.Fprintf(&b, "Collection failures: %d\n", failures)
	return b.String()
}

func defaultNS(ns, fallback string) string {
	if ns != "" {
		return ns
	}
	return fallback
}

func tarGzDir(srcDir, outFile string) error {
	out, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer out.Close()

	gzw := gzip.NewWriter(out)
	defer gzw.Close()
	tw := tar.NewWriter(gzw)
	defer tw.Close()

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(filepath.Dir(srcDir), path)
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = rel
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
}
