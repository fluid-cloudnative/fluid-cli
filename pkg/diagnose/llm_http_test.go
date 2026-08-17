package diagnose

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPLLMClient_Diagnose(t *testing.T) {
	t.Parallel()

	var gotReq chatCompletionRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path: got %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Fatalf("authorization: got %q", auth)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		_ = json.NewEncoder(w).Encode(chatCompletionResponse{
			Choices: []struct {
				Message chatMessage `json:"message"`
			}{{Message: chatMessage{Role: "assistant", Content: "Likely mount failure on fuse pod."}}},
		})
	}))
	defer server.Close()

	client := &HTTPLLMClient{HTTPClient: server.Client()}
	analysis, err := client.Diagnose(context.Background(), LLMRequest{
		Endpoint: server.URL,
		APIKey:   "test-key",
		Model:    "test-model",
		Prompt:   "## Instructions\n- diagnose only\n\n## Dataset\n- Name: demo\n",
	})
	if err != nil {
		t.Fatalf("Diagnose: %v", err)
	}
	if analysis != "Likely mount failure on fuse pod." {
		t.Fatalf("analysis: got %q", analysis)
	}
	if gotReq.Model != "test-model" {
		t.Fatalf("model: got %q", gotReq.Model)
	}
	if len(gotReq.Messages) != 2 {
		t.Fatalf("messages: got %d", len(gotReq.Messages))
	}
	if gotReq.Messages[0].Role != "system" || !strings.Contains(gotReq.Messages[0].Content, "diagnosis only") {
		t.Fatalf("system message: %+v", gotReq.Messages[0])
	}
	if gotReq.Messages[1].Role != "user" || !strings.Contains(gotReq.Messages[1].Content, "## Dataset") {
		t.Fatalf("user message: %+v", gotReq.Messages[1])
	}
}

func TestChatCompletionsURL(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"https://api.openai.com":     "https://api.openai.com/v1/chat/completions",
		"https://api.openai.com/":    "https://api.openai.com/v1/chat/completions",
		"https://api.openai.com/v1":  "https://api.openai.com/v1/chat/completions",
		"https://api.openai.com/v1/": "https://api.openai.com/v1/chat/completions",
	}
	for in, want := range cases {
		if got := chatCompletionsURL(in); got != want {
			t.Fatalf("%q: got %q want %q", in, got, want)
		}
	}
}
