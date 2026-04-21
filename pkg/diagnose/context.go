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
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/fluid-cloudnative/fluid-cli/pkg/inspect"
	fluidv1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	defaultPromptVersion = "fluid-diagnose-prompt/v1"
	maxPromptEvents      = 50
	maxMessageLength     = 500
)

// DiagnosticContext is a stable, JSON-serializable data shape for AI-assisted diagnosis.
// It intentionally keeps only key fields to avoid sending excessive cluster payloads.
type DiagnosticContext struct {
	Version     string           `json:"version"`
	GeneratedAt string           `json:"generatedAt"`
	Dataset     DatasetContext   `json:"dataset"`
	Runtimes    []RuntimeContext `json:"runtimes"`
	Pods        []PodContext     `json:"pods"`
	Events      []EventContext   `json:"events"`
	Summary     SummaryContext   `json:"summary"`
}

type DatasetContext struct {
	Name       string             `json:"name"`
	Namespace  string             `json:"namespace"`
	Phase      string             `json:"phase"`
	Conditions []ConditionContext `json:"conditions"`
	Mounts     []string           `json:"mounts"`
	Labels     map[string]string  `json:"labels,omitempty"`
}

type RuntimeContext struct {
	Name       string             `json:"name"`
	Namespace  string             `json:"namespace"`
	Type       string             `json:"type"`
	Phase      string             `json:"phase"`
	Conditions []ConditionContext `json:"conditions"`
}

type PodContext struct {
	Name          string                 `json:"name"`
	Namespace     string                 `json:"namespace"`
	Phase         string                 `json:"phase"`
	NodeName      string                 `json:"nodeName,omitempty"`
	RestartCount  int32                  `json:"restartCount"`
	Ready         bool                   `json:"ready"`
	ContainerInfo []ContainerStateDigest `json:"containerInfo,omitempty"`
}

type ContainerStateDigest struct {
	Name     string `json:"name"`
	Ready    bool   `json:"ready"`
	Restarts int32  `json:"restarts"`
	State    string `json:"state"`
	Reason   string `json:"reason,omitempty"`
	Message  string `json:"message,omitempty"`
}

type EventContext struct {
	Type      string `json:"type"`
	Reason    string `json:"reason"`
	Object    string `json:"object,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Message   string `json:"message"`
}

type ConditionContext struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

type SummaryContext struct {
	RuntimeCount int            `json:"runtimeCount"`
	PodCount     int            `json:"podCount"`
	PodPhases    map[string]int `json:"podPhases"`
	WarningCount int            `json:"warningCount"`
	LogsIncluded bool           `json:"logsIncluded"`
	Since        string         `json:"since,omitempty"`
	Notes        []string       `json:"notes,omitempty"`
}

type BuildContextInput struct {
	GeneratedAt time.Time
	Dataset     *fluidv1alpha1.Dataset
	Report      *inspect.DatasetReport
	RuntimeObjs []*unstructured.Unstructured
	Pods        []corev1.Pod
	Events      []corev1.Event
	NoLogs      bool
	Since       string
}

func BuildContext(in BuildContextInput) *DiagnosticContext {
	ctx := &DiagnosticContext{
		Version:     defaultPromptVersion,
		GeneratedAt: in.GeneratedAt.UTC().Format(time.RFC3339),
		Dataset:     buildDatasetContext(in.Dataset),
		Runtimes:    buildRuntimeContexts(in.Dataset, in.RuntimeObjs),
		Pods:        buildPodContexts(in.Pods),
		Events:      buildEventContexts(in.Events),
		Summary: SummaryContext{
			PodPhases:    map[string]int{},
			LogsIncluded: !in.NoLogs,
			Since:        strings.TrimSpace(in.Since),
		},
	}

	for _, pod := range ctx.Pods {
		ctx.Summary.PodPhases[pod.Phase]++
	}
	ctx.Summary.RuntimeCount = len(ctx.Runtimes)
	ctx.Summary.PodCount = len(ctx.Pods)
	ctx.Summary.WarningCount = len(ctx.Events)

	if in.NoLogs {
		ctx.Summary.Notes = append(ctx.Summary.Notes, "Pod logs are omitted because --no-logs is enabled.")
	}
	if in.Report != nil && len(in.Report.Runtimes) == 0 {
		ctx.Summary.Notes = append(ctx.Summary.Notes, "No associated runtime resources were discovered.")
	}

	return ctx
}

func (c *DiagnosticContext) MarshalJSON() ([]byte, error) {
	type alias DiagnosticContext
	return json.Marshal((*alias)(c))
}

func buildDatasetContext(ds *fluidv1alpha1.Dataset) DatasetContext {
	if ds == nil {
		return DatasetContext{}
	}

	out := DatasetContext{
		Name:      ds.Name,
		Namespace: ds.Namespace,
		Phase:     string(ds.Status.Phase),
		Labels:    redactMap(ds.Labels),
	}

	for _, m := range ds.Spec.Mounts {
		if m.MountPoint != "" {
			out.Mounts = append(out.Mounts, m.MountPoint)
			continue
		}
		if m.Name != "" {
			out.Mounts = append(out.Mounts, m.Name)
		}
	}
	sort.Strings(out.Mounts)

	for _, cond := range ds.Status.Conditions {
		out.Conditions = append(out.Conditions, ConditionContext{
			Type:    string(cond.Type),
			Status:  string(cond.Status),
			Reason:  cond.Reason,
			Message: trimText(cond.Message, maxMessageLength),
		})
	}
	sort.Slice(out.Conditions, func(i, j int) bool {
		return out.Conditions[i].Type < out.Conditions[j].Type
	})
	return out
}

func buildRuntimeContexts(ds *fluidv1alpha1.Dataset, runtimes []*unstructured.Unstructured) []RuntimeContext {
	phaseByRuntime := map[string]string{}
	conditionsByRuntime := map[string][]ConditionContext{}
	for _, rt := range runtimes {
		if rt == nil {
			continue
		}
		key := fmt.Sprintf("%s/%s", rt.GetNamespace(), rt.GetName())
		if phase, ok, _ := unstructured.NestedString(rt.Object, "status", "phase"); ok {
			phaseByRuntime[key] = phase
		}

		rawConds, ok, _ := unstructured.NestedSlice(rt.Object, "status", "conditions")
		if !ok {
			continue
		}

		conds := make([]ConditionContext, 0, len(rawConds))
		for _, raw := range rawConds {
			m, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			conds = append(conds, ConditionContext{
				Type:    nestedString(m, "type"),
				Status:  nestedString(m, "status"),
				Reason:  nestedString(m, "reason"),
				Message: trimText(nestedString(m, "message"), maxMessageLength),
			})
		}
		sort.Slice(conds, func(i, j int) bool {
			return conds[i].Type < conds[j].Type
		})
		conditionsByRuntime[key] = conds
	}

	out := make([]RuntimeContext, 0, len(ds.Status.Runtimes))
	for _, rt := range ds.Status.Runtimes {
		ns := rt.Namespace
		if ns == "" {
			ns = ds.Namespace
		}
		key := fmt.Sprintf("%s/%s", ns, rt.Name)
		out = append(out, RuntimeContext{
			Name:       rt.Name,
			Namespace:  ns,
			Type:       string(rt.Type),
			Phase:      phaseByRuntime[key],
			Conditions: conditionsByRuntime[key],
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Namespace == out[j].Namespace {
			return out[i].Name < out[j].Name
		}
		return out[i].Namespace < out[j].Namespace
	})
	return out
}

func buildPodContexts(pods []corev1.Pod) []PodContext {
	out := make([]PodContext, 0, len(pods))
	for _, pod := range pods {
		p := PodContext{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Phase:     string(pod.Status.Phase),
			NodeName:  pod.Spec.NodeName,
		}

		var totalRestarts int32
		allReady := true
		for _, cs := range pod.Status.ContainerStatuses {
			totalRestarts += cs.RestartCount
			allReady = allReady && cs.Ready
			p.ContainerInfo = append(p.ContainerInfo, ContainerStateDigest{
				Name:     cs.Name,
				Ready:    cs.Ready,
				Restarts: cs.RestartCount,
				State:    containerStateName(cs.State),
				Reason:   trimText(containerStateReason(cs.State), maxMessageLength),
				Message:  trimText(containerStateMessage(cs.State), maxMessageLength),
			})
		}
		p.RestartCount = totalRestarts
		p.Ready = allReady

		sort.Slice(p.ContainerInfo, func(i, j int) bool {
			return p.ContainerInfo[i].Name < p.ContainerInfo[j].Name
		})
		out = append(out, p)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Namespace == out[j].Namespace {
			return out[i].Name < out[j].Name
		}
		return out[i].Namespace < out[j].Namespace
	})
	return out
}

func buildEventContexts(events []corev1.Event) []EventContext {
	filtered := make([]corev1.Event, 0, len(events))
	for _, ev := range events {
		if ev.Type == corev1.EventTypeWarning {
			filtered = append(filtered, ev)
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return eventTime(filtered[i]).After(eventTime(filtered[j]))
	})
	if len(filtered) > maxPromptEvents {
		filtered = filtered[:maxPromptEvents]
	}

	out := make([]EventContext, 0, len(filtered))
	for _, ev := range filtered {
		ts := eventTime(ev).UTC()
		object := ""
		if ev.InvolvedObject.Kind != "" || ev.InvolvedObject.Name != "" {
			object = strings.Trim(strings.Join([]string{ev.InvolvedObject.Kind, ev.InvolvedObject.Name}, "/"), "/")
		}
		out = append(out, EventContext{
			Type:      ev.Type,
			Reason:    ev.Reason,
			Object:    object,
			Timestamp: ts.Format(time.RFC3339),
			Message:   trimText(ev.Message, maxMessageLength),
		})
	}
	return out
}

func redactMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make(map[string]string, len(in))
	for _, k := range keys {
		v := in[k]
		if isSensitiveKey(k) {
			out[k] = "<redacted>"
			continue
		}
		out[k] = trimText(v, maxMessageLength)
	}
	return out
}

func isSensitiveKey(k string) bool {
	key := strings.ToLower(k)
	sensitive := []string{"password", "secret", "token", "apikey", "api-key", "accesskey", "credential"}
	for _, needle := range sensitive {
		if strings.Contains(key, needle) {
			return true
		}
	}
	return false
}

func trimText(s string, limit int) string {
	if limit <= 0 {
		return ""
	}
	s = strings.TrimSpace(s)
	if len(s) <= limit {
		return s
	}
	return strings.TrimSpace(s[:limit]) + "...(truncated)"
}

func eventTime(ev corev1.Event) time.Time {
	if !ev.EventTime.Time.IsZero() {
		return ev.EventTime.Time
	}
	if !ev.LastTimestamp.Time.IsZero() {
		return ev.LastTimestamp.Time
	}
	return ev.CreationTimestamp.Time
}

func nestedString(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func containerStateName(state corev1.ContainerState) string {
	switch {
	case state.Waiting != nil:
		return "Waiting"
	case state.Running != nil:
		return "Running"
	case state.Terminated != nil:
		return "Terminated"
	default:
		return "Unknown"
	}
}

func containerStateReason(state corev1.ContainerState) string {
	switch {
	case state.Waiting != nil:
		return state.Waiting.Reason
	case state.Terminated != nil:
		return state.Terminated.Reason
	default:
		return ""
	}
}

func containerStateMessage(state corev1.ContainerState) string {
	switch {
	case state.Waiting != nil:
		return state.Waiting.Message
	case state.Terminated != nil:
		return state.Terminated.Message
	default:
		return ""
	}
}
