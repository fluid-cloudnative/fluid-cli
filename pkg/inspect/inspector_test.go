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
	"testing"
	"time"

	fluidv1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
	"github.com/fluid-cloudnative/fluid/pkg/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var fixedTime = time.Date(2026, 3, 26, 2, 28, 9, 0, time.UTC)

func makeTestDataset() *fluidv1alpha1.Dataset {
	return &fluidv1alpha1.Dataset{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "data.fluid.io/v1alpha1",
			Kind:       "Dataset",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test",
			Namespace:         "default",
			UID:               types.UID("f7aa137a-b796-44b6-8de5-7a056fba2b1a"),
			ResourceVersion:   "4591",
			Generation:        1,
			CreationTimestamp: metav1.Time{Time: fixedTime},
			Labels: map[string]string{
				"fluid.io/dataset-id": "default-test",
			},
			Finalizers: []string{"fluid-dataset-controller-finalizer"},
		},
		Spec: fluidv1alpha1.DatasetSpec{
			Mounts: []fluidv1alpha1.Mount{
				{
					MountPoint: "https://downloads.apache.org/zookeeper/stable/",
					Name:       "zookeeper",
				},
			},
			NodeAffinity: &fluidv1alpha1.CacheableNodeAffinity{
				Required: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "nonroot",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"true"},
								},
							},
						},
					},
				},
			},
		},
		Status: fluidv1alpha1.DatasetStatus{
			Phase:    fluidv1alpha1.BoundDatasetPhase,
			UfsTotal: "19.18MiB",
			FileNum:  "6",
			HCFSStatus: &fluidv1alpha1.HCFSStatus{
				Endpoint:                    "alluxio://test-master-0.default:25960",
				UnderlayerFileSystemVersion: "3.3.1",
			},
			CacheStates: common.CacheStateList{
				common.CacheCapacity:         "2.00GiB",
				common.Cached:                "0.00B",
				common.CachedPercentage:      "0.0%",
				common.CacheHitRatio:         "0.0%",
				common.CacheThroughputRatio:  "0.0%",
				common.LocalHitRatio:         "0.0%",
				common.RemoteHitRatio:        "0.0%",
				common.LocalThroughputRatio:  "0.0%",
				common.RemoteThroughputRatio: "0.0%",
			},
			Runtimes: []fluidv1alpha1.Runtime{
				{
					Name:      "test",
					Namespace: "default",
					Category:  "Accelerate",
					Type:      "alluxio",
				},
			},
			Conditions: []fluidv1alpha1.DatasetCondition{
				{
					Type:               fluidv1alpha1.DatasetReady,
					Status:             corev1.ConditionTrue,
					Reason:             "DatasetReady",
					Message:            "The ddc runtime is ready.",
					LastUpdateTime:     metav1.Time{Time: time.Date(2026, 3, 26, 2, 30, 22, 0, time.UTC)},
					LastTransitionTime: metav1.Time{Time: time.Date(2026, 3, 26, 2, 30, 22, 0, time.UTC)},
				},
			},
		},
	}
}

func TestBuildIdentity(t *testing.T) {
	ds := makeTestDataset()
	id := buildIdentity(ds)

	if id.Name != "test" {
		t.Errorf("Name: got %q, want %q", id.Name, "test")
	}
	if id.Namespace != "default" {
		t.Errorf("Namespace: got %q, want %q", id.Namespace, "default")
	}
	if id.APIVersion != "data.fluid.io/v1alpha1" {
		t.Errorf("APIVersion: got %q, want %q", id.APIVersion, "data.fluid.io/v1alpha1")
	}
	if id.Kind != "Dataset" {
		t.Errorf("Kind: got %q, want %q", id.Kind, "Dataset")
	}
	if id.UID != "f7aa137a-b796-44b6-8de5-7a056fba2b1a" {
		t.Errorf("UID: got %q", id.UID)
	}
	if id.ResourceVersion != "4591" {
		t.Errorf("ResourceVersion: got %q", id.ResourceVersion)
	}
	if id.Generation != 1 {
		t.Errorf("Generation: got %d, want 1", id.Generation)
	}
	if len(id.Labels) != 1 || id.Labels["fluid.io/dataset-id"] != "default-test" {
		t.Errorf("Labels: unexpected %v", id.Labels)
	}
	if len(id.Finalizers) != 1 || id.Finalizers[0] != "fluid-dataset-controller-finalizer" {
		t.Errorf("Finalizers: unexpected %v", id.Finalizers)
	}
	if id.CreationTimestamp != fixedTime.Format(time.RFC3339) {
		t.Errorf("CreationTimestamp: got %q", id.CreationTimestamp)
	}
}

func TestBuildIdentity_DefaultsAPIVersionAndKind(t *testing.T) {
	ds := makeTestDataset()
	ds.APIVersion = ""
	ds.Kind = ""
	id := buildIdentity(ds)
	if id.APIVersion != "data.fluid.io/v1alpha1" {
		t.Errorf("expected default APIVersion, got %q", id.APIVersion)
	}
	if id.Kind != "Dataset" {
		t.Errorf("expected default Kind, got %q", id.Kind)
	}
}

func TestBuildSpecSummary(t *testing.T) {
	ds := makeTestDataset()
	spec := buildSpecSummary(ds)

	if len(spec.Mounts) != 1 {
		t.Fatalf("Mounts: got %d, want 1", len(spec.Mounts))
	}
	if spec.Mounts[0].Name != "zookeeper" {
		t.Errorf("Mount Name: got %q", spec.Mounts[0].Name)
	}
	if spec.Mounts[0].MountPoint != "https://downloads.apache.org/zookeeper/stable/" {
		t.Errorf("Mount MountPoint: got %q", spec.Mounts[0].MountPoint)
	}

	if len(spec.NodeAffinity.Terms) != 1 {
		t.Fatalf("NodeAffinity Terms: got %d, want 1", len(spec.NodeAffinity.Terms))
	}
	term := spec.NodeAffinity.Terms[0]
	if len(term) != 1 {
		t.Fatalf("NodeAffinity Term[0] exprs: got %d, want 1", len(term))
	}
	if term[0].Key != "nonroot" || term[0].Operator != "In" {
		t.Errorf("NodeAffinity expr: unexpected key=%q operator=%q", term[0].Key, term[0].Operator)
	}
	if len(term[0].Values) != 1 || term[0].Values[0] != "true" {
		t.Errorf("NodeAffinity expr values: unexpected %v", term[0].Values)
	}
}

func TestBuildSpecSummary_NoAffinity(t *testing.T) {
	ds := makeTestDataset()
	ds.Spec.NodeAffinity = nil
	spec := buildSpecSummary(ds)
	if len(spec.NodeAffinity.Terms) != 0 {
		t.Errorf("expected empty Terms when NodeAffinity is nil, got %v", spec.NodeAffinity.Terms)
	}
}

func TestBuildStatusSummary(t *testing.T) {
	ds := makeTestDataset()
	st := buildStatusSummary(ds)

	if st.Phase != "Bound" {
		t.Errorf("Phase: got %q, want %q", st.Phase, "Bound")
	}
	if st.UfsTotal != "19.18MiB" {
		t.Errorf("UfsTotal: got %q", st.UfsTotal)
	}
	if st.FileNum != "6" {
		t.Errorf("FileNum: got %q", st.FileNum)
	}
	if st.HCFS.Endpoint != "alluxio://test-master-0.default:25960" {
		t.Errorf("HCFS Endpoint: got %q", st.HCFS.Endpoint)
	}
	if st.HCFS.UnderlayerFileSystemVersion != "3.3.1" {
		t.Errorf("HCFS Version: got %q", st.HCFS.UnderlayerFileSystemVersion)
	}
	if st.CacheState.CacheCapacity != "2.00GiB" {
		t.Errorf("CacheCapacity: got %q", st.CacheState.CacheCapacity)
	}
	if st.CacheState.CacheHitRatio != "0.0%" {
		t.Errorf("CacheHitRatio: got %q", st.CacheState.CacheHitRatio)
	}
	if len(st.Runtimes) != 1 {
		t.Fatalf("Runtimes: got %d, want 1", len(st.Runtimes))
	}
	if st.Runtimes[0].Type != "alluxio" {
		t.Errorf("Runtime Type: got %q", st.Runtimes[0].Type)
	}
	if len(st.Conditions) != 1 {
		t.Fatalf("Conditions: got %d, want 1", len(st.Conditions))
	}
	if st.Conditions[0].Type != "Ready" || st.Conditions[0].Status != "True" {
		t.Errorf("Condition: got type=%q status=%q", st.Conditions[0].Type, st.Conditions[0].Status)
	}
}

func TestBuildStatusSummary_NoHCFS(t *testing.T) {
	ds := makeTestDataset()
	ds.Status.HCFSStatus = nil
	st := buildStatusSummary(ds)
	if st.HCFS.Endpoint != "<none>" {
		t.Errorf("HCFS Endpoint with nil HCFSStatus: got %q, want <none>", st.HCFS.Endpoint)
	}
}

func TestBuildStatusSummary_EmptyFields(t *testing.T) {
	ds := makeTestDataset()
	ds.Status.UfsTotal = ""
	ds.Status.FileNum = ""
	st := buildStatusSummary(ds)
	if st.UfsTotal != "<unknown>" {
		t.Errorf("UfsTotal empty: got %q, want <unknown>", st.UfsTotal)
	}
	if st.FileNum != "<unknown>" {
		t.Errorf("FileNum empty: got %q, want <unknown>", st.FileNum)
	}
}
