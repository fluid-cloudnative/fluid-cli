// Copyright 2025 The Fluid Authors.
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

package inspect

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

// buildTestReport returns a fully-populated DatasetReport for rendering tests.
func buildTestReport() *DatasetReport {
	return &DatasetReport{
		Identity: DatasetIdentity{
			Name:              "test",
			Namespace:         "default",
			Labels:            map[string]string{"fluid.io/dataset-id": "default-test"},
			APIVersion:        "data.fluid.io/v1alpha1",
			Kind:              "Dataset",
			CreationTimestamp: "2026-03-26T02:28:09Z",
			Generation:        1,
			ResourceVersion:   "4591",
			UID:               "f7aa137a-b796-44b6-8de5-7a056fba2b1a",
			Finalizers:        []string{"fluid-dataset-controller-finalizer"},
		},
		Spec: DatasetSpecSummary{
			Mounts: []MountSummary{
				{Name: "zookeeper", MountPoint: "https://downloads.apache.org/zookeeper/stable/"},
			},
			NodeAffinity: NodeAffinitySummary{
				Terms: [][]NodeSelectorRequirement{
					{
						{Key: "nonroot", Operator: "In", Values: []string{"true"}},
					},
				},
			},
		},
		Status: DatasetStatusSummary{
			Phase:    "Bound",
			UfsTotal: "19.18MiB",
			FileNum:  "6",
			HCFS: HCFSSummary{
				Endpoint:                    "alluxio://test-master-0.default:25960",
				UnderlayerFileSystemVersion: "3.3.1",
			},
			CacheState: CacheStateSummary{
				CacheCapacity:         "2.00GiB",
				Cached:                "0.00B",
				CachedPercentage:      "0.0%",
				CacheHitRatio:         "0.0%",
				CacheThroughputRatio:  "0.0%",
				LocalHitRatio:         "0.0%",
				RemoteHitRatio:        "0.0%",
				LocalThroughputRatio:  "0.0%",
				RemoteThroughputRatio: "0.0%",
			},
			Runtimes: []RuntimeSummary{
				{Name: "test", Namespace: "default", Category: "Accelerate", Type: "alluxio"},
			},
			Conditions: []ConditionSummary{
				{
					Type:               "Ready",
					Status:             "True",
					Reason:             "DatasetReady",
					Message:            "The ddc runtime is ready.",
					LastUpdateTime:     "2026-03-26T02:30:22Z",
					LastTransitionTime: "2026-03-26T02:30:22Z",
				},
			},
		},
		Runtimes: []RuntimeReport{
			{
				RuntimeType: "alluxio",
				RuntimeName: "test",
				Resources: []ResourceRow{
					{Kind: "StatefulSet", Namespace: "default", Name: "test-master", Status: "1/1", Age: "11d"},
					{Kind: "DaemonSet", Namespace: "default", Name: "test-fuse", Status: "1/1", Age: "11d"},
					{Kind: "Pod", Namespace: "default", Name: "test-master-0", Status: "Running", Node: "node1", Restarts: "0", Age: "11d"},
					{Kind: "Service", Namespace: "default", Name: "test-master-0", Status: "10.96.0.1", Age: "11d"},
					{Kind: "PVC", Namespace: "default", Name: "test", Status: "Bound", Age: "11d"},
					{Kind: "PV", Namespace: "", Name: "default-test", Status: "Bound", Age: "11d"},
				},
			},
		},
		DataOps: []ResourceRow{
			{Kind: "DataLoad", Namespace: "default", Name: "test-dataload", Status: "Complete", Age: "2d"},
		},
	}
}

func TestPrintTable_ContainsIdentityFields(t *testing.T) {
	report := buildTestReport()
	var buf bytes.Buffer
	if err := Print(&buf, report, "table", false); err != nil {
		t.Fatalf("Print error: %v", err)
	}
	out := buf.String()

	checks := []string{
		"Name:         test",
		"Namespace:    default",
		"fluid.io/dataset-id=default-test",
		"API Version:  data.fluid.io/v1alpha1",
		"Kind:         Dataset",
		"Creation Timestamp:  2026-03-26T02:28:09Z",
		"Generation:          1",
		"Resource Version:    4591",
		"UID:                 f7aa137a-b796-44b6-8de5-7a056fba2b1a",
		"fluid-dataset-controller-finalizer",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\nFull output:\n%s", want, out)
		}
	}
}

func TestPrintTable_ContainsSpecFields(t *testing.T) {
	report := buildTestReport()
	var buf bytes.Buffer
	if err := Print(&buf, report, "table", false); err != nil {
		t.Fatalf("Print error: %v", err)
	}
	out := buf.String()

	checks := []string{
		"Spec:",
		"Mount Point:  https://downloads.apache.org/zookeeper/stable/",
		"Name:         zookeeper",
		"Node Affinity:",
		"nonroot",
		"In",
		"true",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestPrintTable_ContainsStatusFields(t *testing.T) {
	report := buildTestReport()
	var buf bytes.Buffer
	if err := Print(&buf, report, "table", false); err != nil {
		t.Fatalf("Print error: %v", err)
	}
	out := buf.String()

	checks := []string{
		"Status:",
		"Cache Capacity:           2.00GiB",
		"Cache Hit Ratio:          0.0%",
		"Cached:                   0.00B",
		"File Num:                6",
		"Endpoint:                        alluxio://test-master-0.default:25960",
		"Underlayer File System Version:  3.3.1",
		"Phase:          Bound",
		"Ufs Total:    19.18MiB",
		"DatasetReady",
		"The ddc runtime is ready.",
		"Category:   Accelerate",
		"Type:       alluxio",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestPrintTable_ContainsResourceTable(t *testing.T) {
	report := buildTestReport()
	var buf bytes.Buffer
	if err := Print(&buf, report, "table", false); err != nil {
		t.Fatalf("Print error: %v", err)
	}
	out := buf.String()

	checks := []string{
		"Associated Resources:",
		"Runtime: test (alluxio)",
		"KIND",
		"StatefulSet",
		"DaemonSet",
		"test-master",
		"test-fuse",
		"Data Operations:",
		"DataLoad",
		"test-dataload",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestPrintTable_WideIncludesNodeAndRestarts(t *testing.T) {
	report := buildTestReport()
	var buf bytes.Buffer
	if err := Print(&buf, report, "table", true); err != nil {
		t.Fatalf("Print error: %v", err)
	}
	out := buf.String()

	if !strings.Contains(out, "NODE") {
		t.Error("wide output should contain NODE column header")
	}
	if !strings.Contains(out, "RESTARTS") {
		t.Error("wide output should contain RESTARTS column header")
	}
	if !strings.Contains(out, "node1") {
		t.Error("wide output should contain node value 'node1'")
	}
}

func TestPrintTable_EmptyRuntimes(t *testing.T) {
	report := buildTestReport()
	report.Runtimes = nil
	var buf bytes.Buffer
	if err := Print(&buf, report, "table", false); err != nil {
		t.Fatalf("Print error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Associated Resources: none") {
		t.Errorf("expected 'Associated Resources: none', got:\n%s", out)
	}
}

func TestPrintTable_EmptyLabels(t *testing.T) {
	report := buildTestReport()
	report.Identity.Labels = nil
	var buf bytes.Buffer
	if err := Print(&buf, report, "table", false); err != nil {
		t.Fatalf("Print error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Labels:       <none>") {
		t.Errorf("expected Labels: <none>, got:\n%s", out)
	}
}

func TestPrintTable_EmptyFinalizers(t *testing.T) {
	report := buildTestReport()
	report.Identity.Finalizers = nil
	var buf bytes.Buffer
	if err := Print(&buf, report, "table", false); err != nil {
		t.Fatalf("Print error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Finalizers:          <none>") {
		t.Errorf("expected Finalizers: <none>, got:\n%s", out)
	}
}

func TestPrintJSON_IsValidJSON(t *testing.T) {
	report := buildTestReport()
	var buf bytes.Buffer
	if err := Print(&buf, report, "json", false); err != nil {
		t.Fatalf("Print JSON error: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("JSON output is invalid: %v\n%s", err, buf.String())
	}
	// Verify top-level keys present
	for _, key := range []string{"Identity", "Spec", "Status", "Runtimes"} {
		if _, ok := parsed[key]; !ok {
			t.Errorf("JSON missing key %q", key)
		}
	}
}

func TestPrintYAML_IsValidYAML(t *testing.T) {
	report := buildTestReport()
	var buf bytes.Buffer
	if err := Print(&buf, report, "yaml", false); err != nil {
		t.Fatalf("Print YAML error: %v", err)
	}
	var parsed map[string]interface{}
	if err := yaml.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("YAML output is invalid: %v\n%s", err, buf.String())
	}
	for _, key := range []string{"Identity", "Spec", "Status", "Runtimes"} {
		if _, ok := parsed[key]; !ok {
			t.Errorf("YAML missing key %q", key)
		}
	}
}

func TestPrintTable_SectionOrder(t *testing.T) {
	report := buildTestReport()
	var buf bytes.Buffer
	if err := Print(&buf, report, "table", false); err != nil {
		t.Fatalf("Print error: %v", err)
	}
	out := buf.String()

	namePos := strings.Index(out, "Name:         test")
	specPos := strings.Index(out, "Spec:")
	statusPos := strings.Index(out, "Status:")
	resourcesPos := strings.Index(out, "Associated Resources:")

	if namePos < 0 || specPos < 0 || statusPos < 0 || resourcesPos < 0 {
		t.Fatalf("one or more sections missing from output")
	}
	if !(namePos < specPos && specPos < statusPos && statusPos < resourcesPos) {
		t.Errorf("sections out of order: Name=%d Spec=%d Status=%d Resources=%d",
			namePos, specPos, statusPos, resourcesPos)
	}
}
