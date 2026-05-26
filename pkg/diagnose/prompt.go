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
)

func FormatPrompt(ctx *DiagnosticContext) string {
	if ctx == nil {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Version: %s\n", ctx.Version)
	fmt.Fprintf(&b, "GeneratedAt: %s\n\n", ctx.GeneratedAt)
	writeModelInstructions(&b)
	writeFAQSection(&b, ctx.MatchedFAQs)
	writeReferenceFAQSection(&b, ctx.ReferenceFAQs)
	writeDatasetSection(&b, ctx.Dataset)
	writeRuntimeSection(&b, ctx.Runtimes)
	writePodSection(&b, ctx.Pods)
	writeEventSection(&b, ctx.Events)
	writeSummarySection(&b, ctx.Summary)
	return strings.TrimSpace(b.String()) + "\n"
}

func ContextAsJSON(ctx *DiagnosticContext) ([]byte, error) {
	return json.MarshalIndent(ctx, "", "  ")
}

const modelInstructionsText = `- Focus on diagnosis only: unhealthy signals, evidence correlation, ranked hypotheses, and uncertainties.
- When Matched FAQs are present, compare cluster evidence to those known Fluid issue patterns and say which apply or conflict.
- When Reference FAQs are present, use them as background knowledge only; do not claim they matched unless evidence supports it.
- Do not provide remediation instructions, shell commands, or kubectl/helm operations.
- If data is insufficient, state what additional observations would increase confidence.`

func writeModelInstructions(b *strings.Builder) {
	b.WriteString("## Instructions\n")
	b.WriteString(modelInstructionsText)
	b.WriteString("\n\n")
}

func writeFAQSection(b *strings.Builder, faqs []FAQMatchContext) {
	b.WriteString("## Matched FAQs (known Fluid issue patterns)\n")
	if len(faqs) == 0 {
		b.WriteString("- None matched by rule engine for this snapshot.\n\n")
		return
	}
	for _, faq := range faqs {
		fmt.Fprintf(b, "- [%s] %s (%s, confidence=%s)\n", faq.ID, faq.Title, faq.Category, faq.Confidence)
		fmt.Fprintf(b, "  Symptom: %s\n", faq.Symptom)
		fmt.Fprintf(b, "  Likely cause: %s\n", faq.LikelyCause)
		fmt.Fprintf(b, "  Verify: %s\n", faq.WhatToVerify)
	}
	b.WriteString("\n")
}

func writeReferenceFAQSection(b *strings.Builder, faqs []FAQReferenceContext) {
	if len(faqs) == 0 {
		return
	}
	b.WriteString("## Reference FAQs (background knowledge)\n")
	for _, faq := range faqs {
		fmt.Fprintf(b, "- [%s] %s\n", faq.ID, faq.Title)
		if faq.Answer != "" {
			fmt.Fprintf(b, "  Answer: %s\n", strings.ReplaceAll(faq.Answer, "\n", "\n  "))
		}
	}
	b.WriteString("\n")
}

// SplitPromptForChat separates system instructions from user diagnostic content.
func SplitPromptForChat(prompt string) (system string, user string) {
	system = "You are a Kubernetes and Fluid storage expert assisting with dataset diagnosis.\n\n" + modelInstructionsText
	marker := "## Instructions"
	if idx := strings.Index(prompt, marker); idx >= 0 {
		rest := prompt[idx+len(marker):]
		if nl := strings.Index(rest, "\n"); nl >= 0 {
			rest = strings.TrimLeft(rest[nl:], "\n")
		}
		user = strings.TrimSpace(rest)
	} else {
		user = strings.TrimSpace(prompt)
	}
	if user == "" {
		user = prompt
	}
	return system, user
}

func writeDatasetSection(b *strings.Builder, ds DatasetContext) {
	b.WriteString("## Dataset\n")
	fmt.Fprintf(b, "- Name: %s\n", ds.Name)
	fmt.Fprintf(b, "- Namespace: %s\n", ds.Namespace)
	fmt.Fprintf(b, "- Phase: %s\n", ds.Phase)

	if len(ds.Mounts) == 0 {
		b.WriteString("- Mounts: <none>\n")
	} else {
		b.WriteString("- Mounts:\n")
		for _, m := range ds.Mounts {
			fmt.Fprintf(b, "  - %s\n", m)
		}
	}

	if len(ds.Conditions) == 0 {
		b.WriteString("- Conditions: <none>\n")
	} else {
		b.WriteString("- Conditions:\n")
		for _, c := range ds.Conditions {
			fmt.Fprintf(b, "  - %s=%s reason=%s message=%s\n", c.Type, c.Status, c.Reason, c.Message)
		}
	}

	if len(ds.Labels) == 0 {
		b.WriteString("- Labels: <none>\n\n")
		return
	}
	keys := make([]string, 0, len(ds.Labels))
	for k := range ds.Labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	b.WriteString("- Labels:\n")
	for _, k := range keys {
		fmt.Fprintf(b, "  - %s=%s\n", k, ds.Labels[k])
	}
	b.WriteString("\n")
}

func writeRuntimeSection(b *strings.Builder, runtimes []RuntimeContext) {
	b.WriteString("## Runtimes\n")
	if len(runtimes) == 0 {
		b.WriteString("- <none>\n\n")
		return
	}
	for _, rt := range runtimes {
		fmt.Fprintf(b, "- %s/%s type=%s phase=%s\n", rt.Namespace, rt.Name, rt.Type, rt.Phase)
		if len(rt.Conditions) == 0 {
			continue
		}
		for _, c := range rt.Conditions {
			fmt.Fprintf(b, "  - condition %s=%s reason=%s message=%s\n", c.Type, c.Status, c.Reason, c.Message)
		}
	}
	b.WriteString("\n")
}

func writePodSection(b *strings.Builder, pods []PodContext) {
	b.WriteString("## Pods\n")
	if len(pods) == 0 {
		b.WriteString("- <none>\n\n")
		return
	}
	for _, pod := range pods {
		fmt.Fprintf(b, "- %s/%s phase=%s ready=%t restarts=%d node=%s\n",
			pod.Namespace, pod.Name, pod.Phase, pod.Ready, pod.RestartCount, pod.NodeName)
		for _, c := range pod.ContainerInfo {
			fmt.Fprintf(b, "  - container=%s state=%s ready=%t restarts=%d reason=%s message=%s\n",
				c.Name, c.State, c.Ready, c.Restarts, c.Reason, c.Message)
		}
	}
	b.WriteString("\n")
}

func writeEventSection(b *strings.Builder, events []EventContext) {
	b.WriteString("## Events\n")
	if len(events) == 0 {
		b.WriteString("- <none>\n\n")
		return
	}
	for _, ev := range events {
		fmt.Fprintf(b, "- [%s] %s %s %s: %s\n", ev.Timestamp, ev.Type, ev.Reason, ev.Object, ev.Message)
	}
	b.WriteString("\n")
}

func writeSummarySection(b *strings.Builder, s SummaryContext) {
	b.WriteString("## Summary\n")
	fmt.Fprintf(b, "- Runtime count: %d\n", s.RuntimeCount)
	fmt.Fprintf(b, "- Pod count: %d\n", s.PodCount)
	fmt.Fprintf(b, "- Warning events: %d\n", s.WarningCount)
	fmt.Fprintf(b, "- Logs included: %t\n", s.LogsIncluded)
	if s.Since != "" {
		fmt.Fprintf(b, "- Since filter: %s\n", s.Since)
	}
	if len(s.PodPhases) == 0 {
		b.WriteString("- Pod phases: <none>\n")
	} else {
		keys := make([]string, 0, len(s.PodPhases))
		for k := range s.PodPhases {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b.WriteString("- Pod phases:\n")
		for _, k := range keys {
			fmt.Fprintf(b, "  - %s=%d\n", k, s.PodPhases[k])
		}
	}
	if len(s.Notes) == 0 {
		b.WriteString("- Notes: <none>\n")
		return
	}
	b.WriteString("- Notes:\n")
	for _, n := range s.Notes {
		fmt.Fprintf(b, "  - %s\n", n)
	}
}
