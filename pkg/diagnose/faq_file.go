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
	"os"
	"path/filepath"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"
)

const faqFileVersion = "fluid-diagnose-faq/v1"
const maxFAQReferenceAnswerLength = 1200

// FAQOptions controls known-issue matching during diagnose.
type FAQOptions struct {
	Skip bool
	File string
}

// FAQFileEntry is one FAQ rule in a YAML catalog (e.g. from the Fluid main repo).
type FAQFileEntry struct {
	ID           string      `json:"id" yaml:"id"`
	Title        string      `json:"title" yaml:"title"`
	Category     string      `json:"category" yaml:"category"`
	Confidence   string      `json:"confidence,omitempty" yaml:"confidence,omitempty"`
	Symptom      string      `json:"symptom" yaml:"symptom"`
	LikelyCause  string      `json:"likelyCause" yaml:"likelyCause"`
	WhatToVerify string      `json:"whatToVerify" yaml:"whatToVerify"`
	When         FAQWhen     `json:"when" yaml:"when"`
}

type faqFile struct {
	Version string         `json:"version" yaml:"version"`
	FAQs    []FAQFileEntry `json:"faqs" yaml:"faqs"`
}

// LoadFAQCatalog returns the FAQ rule catalog (built-in plus optional file overrides).
func LoadFAQCatalog(filePath string) ([]faqRule, error) {
	rules, _, err := LoadFAQData(filePath)
	return rules, err
}

// LoadFAQData returns rule-based FAQs and reference-only FAQs from an optional file.
func LoadFAQData(filePath string) ([]faqRule, []FAQReferenceContext, error) {
	rules := builtinFAQs()
	if strings.TrimSpace(filePath) == "" {
		return rules, nil, nil
	}
	fileRules, references, err := loadFAQFile(filePath)
	if err != nil {
		return nil, nil, err
	}
	return mergeFAQRules(rules, fileRules), references, nil
}

func loadFAQFile(path string) ([]faqRule, []FAQReferenceContext, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("reading FAQ file %q: %w", path, err)
	}
	if isMarkdownFAQFile(path, data) {
		refs := parseMarkdownFAQReferences(path, string(data))
		if len(refs) == 0 {
			return nil, nil, fmt.Errorf("FAQ file %q: no markdown FAQ sections found", path)
		}
		return nil, refs, nil
	}
	rules, err := loadFAQRulesFromYAML(path, data)
	if err != nil {
		return nil, nil, err
	}
	return rules, nil, nil
}

func loadFAQRulesFromFile(path string) ([]faqRule, error) {
	rules, _, err := loadFAQFile(path)
	return rules, err
}

func loadFAQRulesFromYAML(path string, data []byte) ([]faqRule, error) {
	var doc faqFile
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing FAQ file %q: %w", path, err)
	}
	if doc.Version != "" && doc.Version != faqFileVersion {
		return nil, fmt.Errorf("FAQ file %q: unsupported version %q (expected %q)", path, doc.Version, faqFileVersion)
	}
	if len(doc.FAQs) == 0 {
		return nil, fmt.Errorf("FAQ file %q: no faqs defined", path)
	}

	rules := make([]faqRule, 0, len(doc.FAQs))
	seen := make(map[string]struct{}, len(doc.FAQs))
	for i, raw := range doc.FAQs {
		entry, when, err := validateFAQFileEntry(raw, i)
		if err != nil {
			return nil, fmt.Errorf("FAQ file %q: %w", path, err)
		}
		if _, ok := seen[entry.ID]; ok {
			return nil, fmt.Errorf("FAQ file %q: duplicate id %q", path, entry.ID)
		}
		seen[entry.ID] = struct{}{}
		rules = append(rules, ruleFromWhen(entry, when))
	}
	return rules, nil
}

func validateFAQFileEntry(raw FAQFileEntry, index int) (FAQMatchContext, FAQWhen, error) {
	prefix := fmt.Sprintf("faqs[%d]", index)
	if strings.TrimSpace(raw.ID) == "" {
		return FAQMatchContext{}, FAQWhen{}, fmt.Errorf("%s: id is required", prefix)
	}
	if strings.TrimSpace(raw.Title) == "" {
		return FAQMatchContext{}, FAQWhen{}, fmt.Errorf("%s: title is required", prefix)
	}
	cat := FAQCategory(strings.TrimSpace(raw.Category))
	if cat != FAQCategoryMisconfiguration && cat != FAQCategoryOperational {
		return FAQMatchContext{}, FAQWhen{}, fmt.Errorf("%s: category must be misconfiguration or operational", prefix)
	}
	if raw.When.isEmpty() {
		return FAQMatchContext{}, FAQWhen{}, fmt.Errorf("%s: when must include at least one matcher", prefix)
	}
	confidence := strings.TrimSpace(raw.Confidence)
	if confidence == "" {
		confidence = "medium"
	}
	return FAQMatchContext{
		ID:           strings.TrimSpace(raw.ID),
		Title:        strings.TrimSpace(raw.Title),
		Category:     cat,
		Confidence:   confidence,
		Symptom:      strings.TrimSpace(raw.Symptom),
		LikelyCause:  strings.TrimSpace(raw.LikelyCause),
		WhatToVerify: strings.TrimSpace(raw.WhatToVerify),
	}, raw.When, nil
}

func mergeFAQRules(base, overrides []faqRule) []faqRule {
	byID := make(map[string]faqRule, len(base)+len(overrides))
	for _, r := range base {
		byID[r.entry.ID] = r
	}
	for _, r := range overrides {
		byID[r.entry.ID] = r
	}
	out := make([]faqRule, 0, len(byID))
	for _, r := range byID {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].entry.ID < out[j].entry.ID
	})
	return out
}

func isMarkdownFAQFile(path string, data []byte) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".md" || ext == ".markdown" {
		return true
	}
	return strings.HasPrefix(strings.TrimSpace(string(data)), "## ")
}

func parseMarkdownFAQReferences(path, content string) []FAQReferenceContext {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var refs []FAQReferenceContext
	var title string
	var body []string
	position := 0

	flush := func() {
		cleanTitle := cleanMarkdownFAQTitle(title)
		answer := trimFAQAnswer(strings.Join(body, "\n"))
		if cleanTitle == "" || answer == "" {
			return
		}
		position++
		refs = append(refs, FAQReferenceContext{
			ID:       slugFAQTitle(cleanTitle),
			Title:    cleanTitle,
			Answer:   answer,
			Source:   path,
			Position: position,
		})
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			flush()
			title = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			body = body[:0]
			continue
		}
		if title != "" {
			body = append(body, line)
		}
	}
	flush()

	return refs
}

func cleanMarkdownFAQTitle(title string) string {
	title = strings.TrimSpace(title)
	dot := strings.Index(title, ".")
	if dot >= 0 && allDigits(strings.TrimSpace(title[:dot])) {
		title = strings.TrimSpace(title[dot+1:])
	}
	return strings.TrimSpace(title)
}

func trimFAQAnswer(answer string) string {
	answer = strings.TrimSpace(answer)
	answer = strings.ReplaceAll(answer, "**Answer**:", "Answer:")
	answer = strings.ReplaceAll(answer, "**Answer**", "Answer")
	answer = strings.TrimSpace(answer)
	if len(answer) <= maxFAQReferenceAnswerLength {
		return answer
	}
	return strings.TrimSpace(answer[:maxFAQReferenceAnswerLength]) + "...(truncated)"
}

func slugFAQTitle(title string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
