package gemini

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"google.golang.org/genai"
)

func TestIsRetryableGeminiError(t *testing.T) {
	if !isRetryableGeminiError(errors.New("gemini generate: Error 503, UNAVAILABLE")) {
		t.Fatal("503")
	}
	if !isRetryableGeminiError(errors.New("gemini generate: Error 429, rate limit")) {
		t.Fatal("429")
	}
	if isRetryableGeminiError(errors.New("gemini generate: Error 401")) {
		t.Fatal("401 not retryable")
	}
}

func TestGenerate_retries503ThenSuccess(t *testing.T) {
	api, err := newGenAIAPI("unit-test-dummy-key")
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	api.stubGenerate = func(context.Context, string, []*genai.Content, *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
		calls++
		if calls < 3 {
			return nil, fmt.Errorf("gemini generate: Error 503, Message: high demand")
		}
		return jsonResponse(`{"category":"other"}`), nil
	}

	_, err = api.generate(context.Background(), "m", nil, nil)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if calls != 3 {
		t.Fatalf("calls=%d want 3", calls)
	}
}

func TestGenerate_nonRetryableFailsFast(t *testing.T) {
	api, err := newGenAIAPI("unit-test-dummy-key")
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	api.stubGenerate = func(context.Context, string, []*genai.Content, *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
		calls++
		return nil, fmt.Errorf("gemini generate: Error 401, invalid key")
	}

	_, err = api.generate(context.Background(), "m", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("calls=%d want 1", calls)
	}
}

func TestGenerate_ctxCancelDuringBackoff(t *testing.T) {
	api, err := newGenAIAPI("unit-test-dummy-key")
	if err != nil {
		t.Fatal(err)
	}
	api.stubGenerate = func(context.Context, string, []*genai.Content, *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
		return nil, fmt.Errorf("gemini generate: Error 503")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err = api.generate(ctx, "m", nil, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v want deadline exceeded", err)
	}
}
