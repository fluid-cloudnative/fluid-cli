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
)

type LLMRequest struct {
	Endpoint string
	Prompt   string
}

type LLMClient interface {
	Diagnose(ctx context.Context, req LLMRequest) (string, error)
}

// NoopLLMClient is a Phase 3 placeholder. HTTP integration is intentionally deferred.
type NoopLLMClient struct{}

func (NoopLLMClient) Diagnose(_ context.Context, req LLMRequest) (string, error) {
	return "", fmt.Errorf("llm diagnose is not implemented (endpoint=%q)", req.Endpoint)
}
