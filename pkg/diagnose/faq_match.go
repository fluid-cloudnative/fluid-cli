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
	"strings"
)

// FAQWhen describes declarative match criteria (all set fields must match).
type FAQWhen struct {
	DatasetPhaseIn              []string `json:"datasetPhaseIn,omitempty" yaml:"datasetPhaseIn,omitempty"`
	DatasetPhaseEquals          string   `json:"datasetPhaseEquals,omitempty" yaml:"datasetPhaseEquals,omitempty"`
	DatasetMountsEmpty          *bool    `json:"datasetMountsEmpty,omitempty" yaml:"datasetMountsEmpty,omitempty"`
	RuntimeCountZero            *bool    `json:"runtimeCountZero,omitempty" yaml:"runtimeCountZero,omitempty"`
	RuntimeConditionStatusFalse *bool    `json:"runtimeConditionStatusFalse,omitempty" yaml:"runtimeConditionStatusFalse,omitempty"`
	RuntimePhaseNotIn           []string `json:"runtimePhaseNotIn,omitempty" yaml:"runtimePhaseNotIn,omitempty"`
	PodUnhealthy                *bool    `json:"podUnhealthy,omitempty" yaml:"podUnhealthy,omitempty"`
	EventReasonEquals           string   `json:"eventReasonEquals,omitempty" yaml:"eventReasonEquals,omitempty"`
	EventMessageContains        string   `json:"eventMessageContains,omitempty" yaml:"eventMessageContains,omitempty"`
}

func (w FAQWhen) isEmpty() bool {
	return len(w.DatasetPhaseIn) == 0 &&
		w.DatasetPhaseEquals == "" &&
		w.DatasetMountsEmpty == nil &&
		w.RuntimeCountZero == nil &&
		w.RuntimeConditionStatusFalse == nil &&
		len(w.RuntimePhaseNotIn) == 0 &&
		w.PodUnhealthy == nil &&
		w.EventReasonEquals == "" &&
		w.EventMessageContains == ""
}

func evaluateWhen(ctx *DiagnosticContext, w FAQWhen) bool {
	if w.isEmpty() {
		return false
	}
	if len(w.DatasetPhaseIn) > 0 {
		phase := strings.TrimSpace(ctx.Dataset.Phase)
		found := false
		for _, want := range w.DatasetPhaseIn {
			if strings.EqualFold(phase, strings.TrimSpace(want)) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if want := strings.TrimSpace(w.DatasetPhaseEquals); want != "" {
		if !strings.EqualFold(strings.TrimSpace(ctx.Dataset.Phase), want) {
			return false
		}
	}
	if w.DatasetMountsEmpty != nil && *w.DatasetMountsEmpty != (len(ctx.Dataset.Mounts) == 0) {
		return false
	}
	if w.RuntimeCountZero != nil && *w.RuntimeCountZero != (ctx.Summary.RuntimeCount == 0) {
		return false
	}
	if w.RuntimeConditionStatusFalse != nil && *w.RuntimeConditionStatusFalse {
		if !runtimeHasConditionFalse(ctx) {
			return false
		}
	}
	if len(w.RuntimePhaseNotIn) > 0 {
		if !runtimeHasPhaseNotIn(ctx, w.RuntimePhaseNotIn) {
			return false
		}
	}
	if w.PodUnhealthy != nil && *w.PodUnhealthy != podsUnhealthy(ctx) {
		return false
	}
	if reason := strings.TrimSpace(w.EventReasonEquals); reason != "" {
		if !hasEventReason(ctx, reason) {
			return false
		}
	}
	if substr := strings.TrimSpace(w.EventMessageContains); substr != "" {
		if !eventMessageContains(ctx, substr) {
			return false
		}
	}
	return true
}

func runtimeHasConditionFalse(ctx *DiagnosticContext) bool {
	for _, rt := range ctx.Runtimes {
		for _, c := range rt.Conditions {
			if strings.EqualFold(c.Status, "False") {
				return true
			}
		}
	}
	return false
}

func runtimeHasPhaseNotIn(ctx *DiagnosticContext, allowed []string) bool {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, p := range allowed {
		allowedSet[strings.ToLower(strings.TrimSpace(p))] = struct{}{}
	}
	for _, rt := range ctx.Runtimes {
		p := strings.TrimSpace(rt.Phase)
		if p == "" {
			continue
		}
		if _, ok := allowedSet[strings.ToLower(p)]; !ok {
			return true
		}
	}
	return false
}

func podsUnhealthy(ctx *DiagnosticContext) bool {
	for _, pod := range ctx.Pods {
		if pod.Phase != "Running" || !pod.Ready || pod.RestartCount > 0 {
			return true
		}
		for _, c := range pod.ContainerInfo {
			if c.State == "Waiting" || c.State == "Terminated" {
				return true
			}
		}
	}
	return false
}

func eventMessageContains(ctx *DiagnosticContext, substr string) bool {
	substr = strings.ToLower(strings.TrimSpace(substr))
	for _, ev := range ctx.Events {
		if strings.Contains(strings.ToLower(ev.Message), substr) {
			return true
		}
	}
	return false
}

func ruleFromWhen(entry FAQMatchContext, when FAQWhen) faqRule {
	return faqRule{
		entry: entry,
		match: func(ctx *DiagnosticContext) bool {
			return evaluateWhen(ctx, when)
		},
	}
}
