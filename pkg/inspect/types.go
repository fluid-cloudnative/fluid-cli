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

// ResourceRow is a single row in the inspect output table.
type ResourceRow struct {
	Kind      string
	Namespace string
	Name      string
	Status    string
	// Wide-only fields
	Node     string
	Restarts string
	Age      string
}

// MountSummary captures a single mount point from the Dataset spec.
type MountSummary struct {
	Name       string
	MountPoint string
}

// NodeSelectorRequirement represents a single label-selector expression.
type NodeSelectorRequirement struct {
	Key      string
	Operator string
	Values   []string
}

// NodeAffinitySummary holds the required node-selector terms (label expressions).
type NodeAffinitySummary struct {
	// Each inner slice represents one NodeSelectorTerm's MatchExpressions.
	Terms [][]NodeSelectorRequirement
}

// CacheStateSummary contains the Dataset's cache-state metrics.
type CacheStateSummary struct {
	CacheCapacity         string
	Cached                string
	CachedPercentage      string
	CacheHitRatio         string
	CacheThroughputRatio  string
	LocalHitRatio         string
	RemoteHitRatio        string
	LocalThroughputRatio  string
	RemoteThroughputRatio string
}

// HCFSSummary holds HCFS endpoint info.
type HCFSSummary struct {
	Endpoint                    string
	UnderlayerFileSystemVersion string
}

// ConditionSummary holds a single Dataset condition.
type ConditionSummary struct {
	Type               string
	Status             string
	Reason             string
	Message            string
	LastUpdateTime     string
	LastTransitionTime string
}

// RuntimeSummary holds a single Runtime entry from Dataset status.
type RuntimeSummary struct {
	Name      string
	Namespace string
	Category  string
	Type      string
}

// DatasetStatusSummary holds key Dataset status fields.
type DatasetStatusSummary struct {
	Phase      string
	UfsTotal   string
	FileNum    string
	HCFS       HCFSSummary
	Runtimes   []RuntimeSummary
	CacheState CacheStateSummary
	Conditions []ConditionSummary
}

// DatasetSpecSummary holds key Dataset spec fields.
type DatasetSpecSummary struct {
	Mounts       []MountSummary
	NodeAffinity NodeAffinitySummary
}

// DatasetIdentity holds Dataset identity/metadata fields.
type DatasetIdentity struct {
	Name              string
	Namespace         string
	Labels            map[string]string
	APIVersion        string
	Kind              string
	CreationTimestamp string
	Generation        int64
	ResourceVersion   string
	UID               string
	Finalizers        []string
}

// DatasetReport collects all discovered resources for a Dataset.
type DatasetReport struct {
	Identity DatasetIdentity
	Spec     DatasetSpecSummary
	Status   DatasetStatusSummary
	Runtimes []RuntimeReport
	DataOps  []ResourceRow
}

// RuntimeReport collects resources discovered for one Runtime.
type RuntimeReport struct {
	RuntimeType string
	RuntimeName string
	Resources   []ResourceRow
}
