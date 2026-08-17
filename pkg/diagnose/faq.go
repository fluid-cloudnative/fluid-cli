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
	"fmt"
	"sort"
	"strings"

	fluidv1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
)

// FAQCategory groups known issues for reporting and prompts.
type FAQCategory string

const (
	FAQCategoryMisconfiguration FAQCategory = "misconfiguration"
	FAQCategoryOperational      FAQCategory = "operational"
)

// FAQMatchContext is a known-issue hint attached to DiagnosticContext after rule matching.
type FAQMatchContext struct {
	ID           string      `json:"id"`
	Title        string      `json:"title"`
	Category     FAQCategory `json:"category"`
	Confidence   string      `json:"confidence"`
	Symptom      string      `json:"symptom"`
	LikelyCause  string      `json:"likelyCause"`
	WhatToVerify string      `json:"whatToVerify"`
}

// FAQReferenceContext is parsed FAQ prose supplied as LLM reference material.
// Unlike FAQMatchContext, these entries are not deterministic diagnosis matches.
type FAQReferenceContext struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Answer   string `json:"answer"`
	Source   string `json:"source,omitempty"`
	Position int    `json:"position,omitempty"`
}

type faqRule struct {
	entry FAQMatchContext
	match func(*DiagnosticContext) bool
}

func boolPtr(v bool) *bool { return &v }

// builtinFAQs is the embedded starter catalog (overridable via --faq-file).
func builtinFAQs() []faqRule {
	return []faqRule{
		ruleFromWhen(FAQMatchContext{
			ID: "faq-dataset-not-bound", Title: "Dataset is not bound to a runtime",
			Category: FAQCategoryMisconfiguration, Confidence: "high",
			Symptom:      "Dataset phase is NotBound or Pending for an extended period",
			LikelyCause:  "Missing/incorrect Runtime reference, runtime not installed, or runtime failed to become ready",
			WhatToVerify: "Dataset status.runtimes, Runtime CR existence, and runtime controller health",
		}, FAQWhen{DatasetPhaseIn: []string{
			string(fluidv1alpha1.NotBoundDatasetPhase),
			string(fluidv1alpha1.PendingDatasetPhase),
		}}),
		ruleFromWhen(FAQMatchContext{
			ID: "faq-dataset-failed", Title: "Dataset entered Failed phase",
			Category: FAQCategoryOperational, Confidence: "high",
			Symptom:      "Dataset phase is Failed",
			LikelyCause:  "Runtime initialization, mount, or data loading failure",
			WhatToVerify: "Dataset conditions, runtime conditions, and warning events",
		}, FAQWhen{DatasetPhaseEquals: string(fluidv1alpha1.FailedDatasetPhase)}),
		ruleFromWhen(FAQMatchContext{
			ID: "faq-dataset-no-mounts", Title: "Dataset spec has no mounts configured",
			Category: FAQCategoryMisconfiguration, Confidence: "high",
			Symptom:      "Dataset spec.mounts is empty",
			LikelyCause:  "Incomplete Dataset specification or wrong Dataset template",
			WhatToVerify: "Dataset YAML spec.mounts and intended data source configuration",
		}, FAQWhen{DatasetMountsEmpty: boolPtr(true)}),
		ruleFromWhen(FAQMatchContext{
			ID: "faq-no-runtime-reported", Title: "No runtime associated with the Dataset",
			Category: FAQCategoryMisconfiguration, Confidence: "medium",
			Symptom:      "Dataset status lists zero runtimes",
			LikelyCause:  "Runtime not created, label/selector mismatch, or discovery gap",
			WhatToVerify: "Expected Runtime CR for this Dataset and dataset-controller logs",
		}, FAQWhen{RuntimeCountZero: boolPtr(true)}),
		ruleFromWhen(FAQMatchContext{
			ID: "faq-runtime-condition-false", Title: "Runtime reports not-ready conditions",
			Category: FAQCategoryOperational, Confidence: "medium",
			Symptom:      "One or more Runtime conditions are False",
			LikelyCause:  "Underlying engine (e.g. Alluxio/Jindo) not healthy or misconfigured runtime spec",
			WhatToVerify: "Runtime status.conditions and engine-specific operator logs",
		}, FAQWhen{RuntimeConditionStatusFalse: boolPtr(true)}),
		ruleFromWhen(FAQMatchContext{
			ID: "faq-runtime-phase-not-ready", Title: "Runtime phase is not Ready or Bound",
			Category: FAQCategoryOperational, Confidence: "medium",
			Symptom:      "Runtime status.phase is set and is not Ready/Bound",
			LikelyCause:  "Runtime still initializing or stuck in a transitional/error phase",
			WhatToVerify: "Runtime status.phase and controller logs",
		}, FAQWhen{RuntimePhaseNotIn: []string{"Ready", "Bound"}}),
		ruleFromWhen(FAQMatchContext{
			ID: "faq-pod-unhealthy", Title: "Fluid-related pods are unhealthy",
			Category: FAQCategoryOperational, Confidence: "medium",
			Symptom:      "Pods not Running/Ready or containers restarting",
			LikelyCause:  "Resource limits, image pull, mount failure, or node scheduling issues",
			WhatToVerify: "Pod container states, node capacity, and pod logs in the bundle",
		}, FAQWhen{PodUnhealthy: boolPtr(true)}),
		ruleFromWhen(FAQMatchContext{
			ID: "faq-event-failed-mount", Title: "Volume mount failures detected",
			Category: FAQCategoryMisconfiguration, Confidence: "high",
			Symptom:      "Warning events mention FailedMount",
			LikelyCause:  "PVC/PV mismatch, wrong mount path, or CSI/node mount issues",
			WhatToVerify: "PVC/PV artifacts, pod volume mounts, and storage class compatibility",
		}, FAQWhen{EventReasonEquals: "FailedMount"}),
		ruleFromWhen(FAQMatchContext{
			ID: "faq-event-failed-scheduling", Title: "Pod scheduling failures detected",
			Category: FAQCategoryOperational, Confidence: "high",
			Symptom:      "Warning events mention FailedScheduling",
			LikelyCause:  "Insufficient resources, node selectors/taints, or topology constraints",
			WhatToVerify: "Pod spec resources/affinity and node availability",
		}, FAQWhen{EventReasonEquals: "FailedScheduling"}),
	}
}

func hasEventReason(ctx *DiagnosticContext, reason string) bool {
	for _, ev := range ctx.Events {
		if strings.EqualFold(ev.Reason, reason) {
			return true
		}
		if strings.Contains(strings.ToLower(ev.Message), strings.ToLower(reason)) {
			return true
		}
	}
	return false
}

func matchFAQs(ctx *DiagnosticContext, catalog []faqRule) []FAQMatchContext {
	if ctx == nil {
		return nil
	}
	matches := make([]FAQMatchContext, 0)
	for _, rule := range catalog {
		if rule.match(ctx) {
			matches = append(matches, rule.entry)
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].ID < matches[j].ID
	})
	return matches
}

func attachFAQMatches(ctx *DiagnosticContext, opts FAQOptions) error {
	if ctx == nil || opts.Skip {
		return nil
	}
	catalog, references, err := LoadFAQData(opts.File)
	if err != nil {
		return err
	}
	ctx.MatchedFAQs = matchFAQs(ctx, catalog)
	ctx.ReferenceFAQs = references
	if len(ctx.MatchedFAQs) > 0 {
		note := fmt.Sprintf("Matched %d known FAQ issue pattern(s); see matchedFAQs in context.json.", len(ctx.MatchedFAQs))
		if strings.TrimSpace(opts.File) != "" {
			note += fmt.Sprintf(" (catalog: built-in + %s)", opts.File)
		}
		ctx.Summary.Notes = append(ctx.Summary.Notes, note)
	}
	if len(ctx.ReferenceFAQs) > 0 {
		ctx.Summary.Notes = append(ctx.Summary.Notes,
			fmt.Sprintf("Loaded %d reference FAQ(s) from %s.", len(ctx.ReferenceFAQs), opts.File))
	}
	return nil
}
