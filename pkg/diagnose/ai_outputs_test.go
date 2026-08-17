package diagnose

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	fluidv1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWriteAIOutputs_LLMAnalysis(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(chatCompletionResponse{
			Choices: []struct {
				Message chatMessage `json:"message"`
			}{{Message: chatMessage{Content: "Analysis result"}}},
		})
	}))
	defer server.Close()

	dir := t.TempDir()
	paths, err := writeAIOutputs(context.Background(), dir, BuildContextInput{
		GeneratedAt: time.Now(),
		Dataset: &fluidv1alpha1.Dataset{
			ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
		},
	}, Options{
		LLMEndpoint: server.URL,
		LLMAPIKey:   "key",
		LLMModel:    "test",
		LLMSkip:     false,
	}, nil)
	if err != nil {
		t.Fatalf("writeAIOutputs: %v", err)
	}
	analysisPath := filepath.Join(dir, defaultAnalysisFile)
	if paths.AnalysisPath != analysisPath {
		t.Fatalf("AnalysisPath: got %q want %q", paths.AnalysisPath, analysisPath)
	}
	data, err := os.ReadFile(analysisPath)
	if err != nil {
		t.Fatalf("read analysis: %v", err)
	}
	if !strings.Contains(string(data), "Analysis result") {
		t.Fatalf("analysis content: %q", string(data))
	}
}
