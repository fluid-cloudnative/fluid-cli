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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultLLMHTTPTimeout = 120 * time.Second

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// HTTPLLMClient calls an OpenAI-compatible chat completions API.
type HTTPLLMClient struct {
	HTTPClient *http.Client
}

func (c *HTTPLLMClient) Diagnose(ctx context.Context, req LLMRequest) (string, error) {
	if strings.TrimSpace(req.Endpoint) == "" {
		return "", fmt.Errorf("llm endpoint is required")
	}
	if strings.TrimSpace(req.APIKey) == "" {
		return "", fmt.Errorf("llm API key is required (set %s or diagnose.llm.api_key in ~/.fluid/config)", envLLMAPIKey)
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = defaultLLMModel
	}

	system, user := SplitPromptForChat(req.Prompt)
	body, err := json.Marshal(chatCompletionRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	})
	if err != nil {
		return "", fmt.Errorf("marshaling chat request: %w", err)
	}

	url := chatCompletionsURL(req.Endpoint)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+req.APIKey)

	client := c.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultLLMHTTPTimeout}
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("calling LLM endpoint %q: %w", url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return "", fmt.Errorf("reading LLM response: %w", err)
	}

	var parsed chatCompletionResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("decoding LLM response (HTTP %d): %w", resp.StatusCode, err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("LLM API error: %s", parsed.Error.Message)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if len(msg) > 500 {
			msg = msg[:500] + "..."
		}
		return "", fmt.Errorf("LLM API returned HTTP %d: %s", resp.StatusCode, msg)
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return "", fmt.Errorf("LLM API returned no choices")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

func chatCompletionsURL(base string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if strings.HasSuffix(base, "/v1") {
		return base + "/chat/completions"
	}
	return base + "/v1/chat/completions"
}
