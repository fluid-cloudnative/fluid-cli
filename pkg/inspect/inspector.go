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

// Package inspect implements the core resource-discovery logic for fluid inspect.
package inspect

import (
	"context"
	"fmt"
	"strings"
	"time"

	fluidv1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
	"github.com/fluid-cloudnative/fluid/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// runtimeTypeToAppLabel maps the Runtime.Type string from Dataset.Status.Runtimes
// to the "app" label value used by Fluid on pods/statefulsets/daemonsets.
var runtimeTypeToAppLabel = map[string]string{
	"AlluxioRuntime":  "alluxio",
	"JuiceFSRuntime":  "juicefs",
	"JindoRuntime":    "jindo",
	"ThinRuntime":     "thin",
	"VineyardRuntime": "vineyard",
	"EFCRuntime":      "efc",
	"GooseFSRuntime":  "goosefs",
}

// Inspector discovers all resources associated with a Fluid Dataset.
type Inspector struct {
	client client.Client
}

// New creates a new Inspector using the provided controller-runtime client.
func New(c client.Client) *Inspector {
	return &Inspector{client: c}
}

// Run fetches the Dataset and all its associated resources, returning a DatasetReport.
func (i *Inspector) Run(ctx context.Context, name, namespace string) (*DatasetReport, error) {
	dataset := &fluidv1alpha1.Dataset{}
	if err := i.client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, dataset); err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("dataset %q not found in namespace %q", name, namespace)
		}
		return nil, fmt.Errorf("getting dataset: %w", err)
	}

	report := &DatasetReport{
		Identity: buildIdentity(dataset),
		Spec:     buildSpecSummary(dataset),
		Status:   buildStatusSummary(dataset),
	}

	for _, rt := range dataset.Status.Runtimes {
		rtReport := i.inspectRuntime(ctx, rt, dataset)
		report.Runtimes = append(report.Runtimes, rtReport)
	}

	report.DataOps = i.listDataOps(ctx, name, namespace)

	return report, nil
}

// buildIdentity extracts Dataset identity and metadata fields.
func buildIdentity(dataset *fluidv1alpha1.Dataset) DatasetIdentity {
	finalizers := make([]string, len(dataset.Finalizers))
	copy(finalizers, dataset.Finalizers)

	labels := make(map[string]string)
	for k, v := range dataset.Labels {
		labels[k] = v
	}

	apiVersion := dataset.APIVersion
	if apiVersion == "" {
		apiVersion = "data.fluid.io/v1alpha1"
	}
	kind := dataset.Kind
	if kind == "" {
		kind = "Dataset"
	}

	return DatasetIdentity{
		Name:              dataset.Name,
		Namespace:         dataset.Namespace,
		Labels:            labels,
		APIVersion:        apiVersion,
		Kind:              kind,
		CreationTimestamp: dataset.CreationTimestamp.UTC().Format(time.RFC3339),
		Generation:        dataset.Generation,
		ResourceVersion:   dataset.ResourceVersion,
		UID:               string(dataset.UID),
		Finalizers:        finalizers,
	}
}

// buildSpecSummary extracts the key spec fields.
func buildSpecSummary(dataset *fluidv1alpha1.Dataset) DatasetSpecSummary {
	spec := DatasetSpecSummary{}

	for _, m := range dataset.Spec.Mounts {
		spec.Mounts = append(spec.Mounts, MountSummary{
			Name:       m.Name,
			MountPoint: m.MountPoint,
		})
	}

	if dataset.Spec.NodeAffinity != nil && dataset.Spec.NodeAffinity.Required != nil {
		for _, term := range dataset.Spec.NodeAffinity.Required.NodeSelectorTerms {
			var reqs []NodeSelectorRequirement
			for _, expr := range term.MatchExpressions {
				reqs = append(reqs, NodeSelectorRequirement{
					Key:      expr.Key,
					Operator: string(expr.Operator),
					Values:   expr.Values,
				})
			}
			spec.NodeAffinity.Terms = append(spec.NodeAffinity.Terms, reqs)
		}
	}

	return spec
}

// buildStatusSummary extracts key Dataset status fields.
func buildStatusSummary(dataset *fluidv1alpha1.Dataset) DatasetStatusSummary {
	s := DatasetStatusSummary{
		Phase:    string(dataset.Status.Phase),
		UfsTotal: orUnknown(dataset.Status.UfsTotal),
		FileNum:  orUnknown(dataset.Status.FileNum),
	}

	if dataset.Status.HCFSStatus != nil {
		s.HCFS = HCFSSummary{
			Endpoint:                    orUnknown(dataset.Status.HCFSStatus.Endpoint),
			UnderlayerFileSystemVersion: orUnknown(dataset.Status.HCFSStatus.UnderlayerFileSystemVersion),
		}
	} else {
		s.HCFS = HCFSSummary{
			Endpoint:                    "<none>",
			UnderlayerFileSystemVersion: "<none>",
		}
	}

	cs := dataset.Status.CacheStates
	s.CacheState = CacheStateSummary{
		CacheCapacity:         orUnknown(cs[common.CacheCapacity]),
		Cached:                orUnknown(cs[common.Cached]),
		CachedPercentage:      orUnknown(cs[common.CachedPercentage]),
		CacheHitRatio:         orUnknown(cs[common.CacheHitRatio]),
		CacheThroughputRatio:  orUnknown(cs[common.CacheThroughputRatio]),
		LocalHitRatio:         orUnknown(cs[common.LocalHitRatio]),
		RemoteHitRatio:        orUnknown(cs[common.RemoteHitRatio]),
		LocalThroughputRatio:  orUnknown(cs[common.LocalThroughputRatio]),
		RemoteThroughputRatio: orUnknown(cs[common.RemoteThroughputRatio]),
	}

	for _, rt := range dataset.Status.Runtimes {
		s.Runtimes = append(s.Runtimes, RuntimeSummary{
			Name:      rt.Name,
			Namespace: rt.Namespace,
			Category:  string(rt.Category),
			Type:      rt.Type,
		})
	}

	for _, c := range dataset.Status.Conditions {
		s.Conditions = append(s.Conditions, ConditionSummary{
			Type:               string(c.Type),
			Status:             string(c.Status),
			Reason:             c.Reason,
			Message:            c.Message,
			LastUpdateTime:     c.LastUpdateTime.UTC().Format(time.RFC3339),
			LastTransitionTime: c.LastTransitionTime.UTC().Format(time.RFC3339),
		})
	}

	return s
}

func orUnknown(s string) string {
	if s == "" {
		return "<unknown>"
	}
	return s
}

func (i *Inspector) inspectRuntime(ctx context.Context, rt fluidv1alpha1.Runtime, dataset *fluidv1alpha1.Dataset) RuntimeReport {
	rr := RuntimeReport{
		RuntimeType: rt.Type,
		RuntimeName: rt.Name,
	}

	ns := rt.Namespace
	if ns == "" {
		ns = dataset.Namespace
	}

	appLabel := runtimeTypeToAppLabel[rt.Type]

	baseSelector := labels.Set{"release": rt.Name}
	if appLabel != "" {
		baseSelector["app"] = appLabel
	}

	// Pods
	podList := &corev1.PodList{}
	if err := i.client.List(ctx, podList,
		client.InNamespace(ns),
		client.MatchingLabels(baseSelector),
	); err == nil {
		for idx := range podList.Items {
			pod := &podList.Items[idx]
			rr.Resources = append(rr.Resources, podRow(pod))
		}
	}

	// StatefulSets
	stsList := &appsv1.StatefulSetList{}
	if err := i.client.List(ctx, stsList,
		client.InNamespace(ns),
		client.MatchingLabels(baseSelector),
	); err == nil {
		for idx := range stsList.Items {
			sts := &stsList.Items[idx]
			rr.Resources = append(rr.Resources, stsRow(sts))
		}
	}

	// DaemonSets (fuse pods are typically managed via DaemonSet)
	dsList := &appsv1.DaemonSetList{}
	if err := i.client.List(ctx, dsList,
		client.InNamespace(ns),
		client.MatchingLabels(baseSelector),
	); err == nil {
		for idx := range dsList.Items {
			ds := &dsList.Items[idx]
			rr.Resources = append(rr.Resources, dsRow(ds))
		}
	}

	// Deployments (some runtimes use Deployments for master)
	deployList := &appsv1.DeploymentList{}
	if err := i.client.List(ctx, deployList,
		client.InNamespace(ns),
		client.MatchingLabels(baseSelector),
	); err == nil {
		for idx := range deployList.Items {
			d := &deployList.Items[idx]
			rr.Resources = append(rr.Resources, deployRow(d))
		}
	}

	// Services
	svcList := &corev1.ServiceList{}
	if err := i.client.List(ctx, svcList,
		client.InNamespace(ns),
		client.MatchingLabels(labels.Set{"release": rt.Name}),
	); err == nil {
		for idx := range svcList.Items {
			svc := &svcList.Items[idx]
			rr.Resources = append(rr.Resources, svcRow(svc))
		}
	}

	// PVCs — Fluid PVC name matches the Dataset name, namespace matches Dataset namespace
	pvc := &corev1.PersistentVolumeClaim{}
	if err := i.client.Get(ctx, types.NamespacedName{
		Name:      dataset.Name,
		Namespace: dataset.Namespace,
	}, pvc); err == nil {
		rr.Resources = append(rr.Resources, pvcRow(pvc))
	}

	// Also check for additional PVCs with managed-by label
	pvcList := &corev1.PersistentVolumeClaimList{}
	if err := i.client.List(ctx, pvcList,
		client.InNamespace(ns),
		client.MatchingLabels(labels.Set{"fluid.io/managed-by": "fluid"}),
	); err == nil {
		for idx := range pvcList.Items {
			p := &pvcList.Items[idx]
			// avoid duplicating the primary PVC already found above
			if p.Name == dataset.Name && p.Namespace == dataset.Namespace {
				continue
			}
			if strings.Contains(p.Name, rt.Name) {
				rr.Resources = append(rr.Resources, pvcRow(p))
			}
		}
	}

	// PV — convention: {namespace}-{datasetName}
	pvName := fmt.Sprintf("%s-%s", dataset.Namespace, dataset.Name)
	pv := &corev1.PersistentVolume{}
	if err := i.client.Get(ctx, types.NamespacedName{Name: pvName}, pv); err == nil {
		rr.Resources = append(rr.Resources, pvRow(pv))
	}

	return rr
}

// listDataOps finds DataLoads, DataMigrates, and DataBackups referencing the dataset.
func (i *Inspector) listDataOps(ctx context.Context, datasetName, namespace string) []ResourceRow {
	var rows []ResourceRow

	// DataLoads
	dlList := &fluidv1alpha1.DataLoadList{}
	if err := i.client.List(ctx, dlList, client.InNamespace(namespace)); err == nil {
		for idx := range dlList.Items {
			dl := &dlList.Items[idx]
			if dl.Spec.Dataset.Name == datasetName {
				rows = append(rows, ResourceRow{
					Kind:      "DataLoad",
					Namespace: dl.Namespace,
					Name:      dl.Name,
					Status:    string(dl.Status.Phase),
					Age:       humanAge(dl.CreationTimestamp.Time),
				})
			}
		}
	}

	// DataMigrates
	dmList := &fluidv1alpha1.DataMigrateList{}
	if err := i.client.List(ctx, dmList, client.InNamespace(namespace)); err == nil {
		for idx := range dmList.Items {
			dm := &dmList.Items[idx]
			refName := ""
			if dm.Spec.From.DataSet != nil {
				refName = dm.Spec.From.DataSet.Name
			}
			if refName == datasetName {
				rows = append(rows, ResourceRow{
					Kind:      "DataMigrate",
					Namespace: dm.Namespace,
					Name:      dm.Name,
					Status:    string(dm.Status.Phase),
					Age:       humanAge(dm.CreationTimestamp.Time),
				})
			}
		}
	}

	// DataBackups
	dbList := &fluidv1alpha1.DataBackupList{}
	if err := i.client.List(ctx, dbList, client.InNamespace(namespace)); err == nil {
		for idx := range dbList.Items {
			db := &dbList.Items[idx]
			if db.Spec.Dataset == datasetName {
				rows = append(rows, ResourceRow{
					Kind:      "DataBackup",
					Namespace: db.Namespace,
					Name:      db.Name,
					Status:    string(db.Status.Phase),
					Age:       humanAge(db.CreationTimestamp.Time),
				})
			}
		}
	}

	return rows
}

// ---- row builders ----

func podRow(pod *corev1.Pod) ResourceRow {
	status := string(pod.Status.Phase)
	if pod.DeletionTimestamp != nil {
		status = "Terminating"
	}
	restarts := int32(0)
	for _, cs := range pod.Status.ContainerStatuses {
		restarts += cs.RestartCount
	}
	return ResourceRow{
		Kind:      "Pod",
		Namespace: pod.Namespace,
		Name:      pod.Name,
		Status:    status,
		Node:      pod.Spec.NodeName,
		Restarts:  fmt.Sprintf("%d", restarts),
		Age:       humanAge(pod.CreationTimestamp.Time),
	}
}

func stsRow(sts *appsv1.StatefulSet) ResourceRow {
	status := fmt.Sprintf("%d/%d", sts.Status.ReadyReplicas, sts.Status.Replicas)
	return ResourceRow{
		Kind:      "StatefulSet",
		Namespace: sts.Namespace,
		Name:      sts.Name,
		Status:    status,
		Age:       humanAge(sts.CreationTimestamp.Time),
	}
}

func dsRow(ds *appsv1.DaemonSet) ResourceRow {
	status := fmt.Sprintf("%d/%d", ds.Status.NumberReady, ds.Status.DesiredNumberScheduled)
	return ResourceRow{
		Kind:      "DaemonSet",
		Namespace: ds.Namespace,
		Name:      ds.Name,
		Status:    status,
		Age:       humanAge(ds.CreationTimestamp.Time),
	}
}

func deployRow(d *appsv1.Deployment) ResourceRow {
	status := fmt.Sprintf("%d/%d", d.Status.ReadyReplicas, d.Status.Replicas)
	return ResourceRow{
		Kind:      "Deployment",
		Namespace: d.Namespace,
		Name:      d.Name,
		Status:    status,
		Age:       humanAge(d.CreationTimestamp.Time),
	}
}

func svcRow(svc *corev1.Service) ResourceRow {
	clusterIP := svc.Spec.ClusterIP
	return ResourceRow{
		Kind:      "Service",
		Namespace: svc.Namespace,
		Name:      svc.Name,
		Status:    clusterIP,
		Age:       humanAge(svc.CreationTimestamp.Time),
	}
}

func pvcRow(pvc *corev1.PersistentVolumeClaim) ResourceRow {
	return ResourceRow{
		Kind:      "PVC",
		Namespace: pvc.Namespace,
		Name:      pvc.Name,
		Status:    string(pvc.Status.Phase),
		Age:       humanAge(pvc.CreationTimestamp.Time),
	}
}

func pvRow(pv *corev1.PersistentVolume) ResourceRow {
	return ResourceRow{
		Kind:      "PV",
		Namespace: "",
		Name:      pv.Name,
		Status:    string(pv.Status.Phase),
		Age:       humanAge(pv.CreationTimestamp.Time),
	}
}

func humanAge(t time.Time) string {
	if t.IsZero() {
		return "<unknown>"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
