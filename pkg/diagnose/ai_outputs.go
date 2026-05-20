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
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultContextFile  = "context.json"
	defaultPromptFile   = "prompt.txt"
	defaultAnalysisFile = "llm-analysis.txt"
)

type aiOutputPaths struct {
	ContextPath  string
	PromptPath   string
	AnalysisPath string
}

// writeAIOutputs builds DiagnosticContext and writes context.json, prompt, and optional LLM analysis.
func writeAIOutputs(ctx context.Context, baseDir string, in BuildContextInput, opts Options, stderr io.Writer) (aiOutputPaths, error) {
	diagCtx := BuildContext(in)
	contextBytes, err := ContextAsJSON(diagCtx)
	if err != nil {
		return aiOutputPaths{}, fmt.Errorf("marshaling diagnostic context: %w", err)
	}
	prompt := FormatPrompt(diagCtx)

	paths := aiOutputPaths{}
	if baseDir != "" {
		contextPath := filepath.Join(baseDir, defaultContextFile)
		if err := os.WriteFile(contextPath, contextBytes, 0o644); err != nil {
			return aiOutputPaths{}, fmt.Errorf("writing %s: %w", defaultContextFile, err)
		}
		paths.ContextPath = contextPath

		promptPath := filepath.Join(baseDir, defaultPromptFile)
		if err := os.WriteFile(promptPath, []byte(prompt), 0o644); err != nil {
			return aiOutputPaths{}, fmt.Errorf("writing %s: %w", defaultPromptFile, err)
		}
		paths.PromptPath = promptPath
	}

	extraPrompt := strings.TrimSpace(opts.PromptFile)
	if extraPrompt != "" {
		if err := os.WriteFile(extraPrompt, []byte(prompt), 0o644); err != nil {
			return aiOutputPaths{}, fmt.Errorf("writing prompt file %q: %w", extraPrompt, err)
		}
		if paths.PromptPath == "" {
			paths.PromptPath = extraPrompt
		}
	}

	if opts.LLMSkip || strings.TrimSpace(opts.LLMEndpoint) == "" {
		return paths, nil
	}

	client := NewLLMClient(opts.LLMEndpoint, opts.LLMSkip)
	analysis, err := client.Diagnose(ctx, LLMRequest{
		Endpoint: opts.LLMEndpoint,
		APIKey:   opts.LLMAPIKey,
		Model:    opts.LLMModel,
		Prompt:   prompt,
	})
	if err != nil {
		return paths, fmt.Errorf("LLM diagnosis: %w", err)
	}

	analysisPath := extraPrompt
	if baseDir != "" {
		analysisPath = filepath.Join(baseDir, defaultAnalysisFile)
	}
	if analysisPath == "" {
		return paths, fmt.Errorf("LLM analysis produced output but no path to write it (use --output=dir or --prompt-file)")
	}
	if err := os.WriteFile(analysisPath, []byte(analysis+"\n"), 0o644); err != nil {
		return paths, fmt.Errorf("writing LLM analysis: %w", err)
	}
	paths.AnalysisPath = analysisPath

	if stderr != nil {
		fmt.Fprintf(stderr, "diagnose: LLM analysis written to %s\n", analysisPath)
	}

	return paths, nil
}
