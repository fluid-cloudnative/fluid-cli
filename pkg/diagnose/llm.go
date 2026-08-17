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

// LLM integration uses an OpenAI-compatible chat completions API (net/http only).
//
//   POST {endpoint}/v1/chat/completions
//   Authorization: Bearer <api_key>
//   Body: { "model": "...", "messages": [ { "role": "system", ... }, { "role": "user", ... } ] }

package diagnose

import (
	"context"
	"strings"
)

const defaultLLMModel = "gpt-4o-mini"

// DefaultLLMModel returns the default model name when none is configured.
func DefaultLLMModel() string {
	return defaultLLMModel
}

type LLMRequest struct {
	Endpoint string
	APIKey   string
	Model    string
	Prompt   string
}

type LLMClient interface {
	Diagnose(ctx context.Context, req LLMRequest) (string, error)
}

// NoopLLMClient skips remote calls.
type NoopLLMClient struct{}

func (NoopLLMClient) Diagnose(_ context.Context, _ LLMRequest) (string, error) {
	return "", nil
}

// NewLLMClient returns an HTTP client when endpoint is set and skip is false.
func NewLLMClient(endpoint string, skip bool) LLMClient {
	if skip || strings.TrimSpace(endpoint) == "" {
		return NoopLLMClient{}
	}
	return &HTTPLLMClient{}
}
