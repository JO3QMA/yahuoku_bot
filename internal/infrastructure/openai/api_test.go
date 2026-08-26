package openai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestIsRetryableOpenAIError(t *testing.T) {
	if !isRetryableOpenAIError(errors.New("openai API status 429: rate limit")) {
		t.Fatal("429")
	}
	if !isRetryableOpenAIError(errors.New("openai API status 503: unavailable")) {
		t.Fatal("503")
	}
	if !isRetryableOpenAIError(errors.New("openai request: connection refused")) {
		t.Fatal("network error")
	}
	if isRetryableOpenAIError(errors.New("openai API status 401: invalid key")) {
		t.Fatal("401 not retryable")
	}
	if isRetryableOpenAIError(fmt.Errorf("openai request: %w", context.Canceled)) {
		t.Fatal("context canceled not retryable")
	}
	if isRetryableOpenAIError(fmt.Errorf("openai request: %w", context.DeadlineExceeded)) {
		t.Fatal("deadline exceeded not retryable")
	}
}

func TestChat_retries429ThenSuccess(t *testing.T) {
	api := &apiClient{httpClient: &http.Client{}}
	calls := 0
	api.stubChat = func(context.Context, string, []chatMessage, *chatConfig) (*chatResponse, error) {
		calls++
		if calls < 3 {
			return nil, fmt.Errorf("openai API status 429: rate limit")
		}
		return jsonResponse(`{"category":"other"}`), nil
	}

	_, err := api.chat(context.Background(), "m", nil, nil)
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if calls != 3 {
		t.Fatalf("calls=%d want 3", calls)
	}
}

func TestChat_nonRetryableFailsFast(t *testing.T) {
	api := &apiClient{httpClient: &http.Client{}}
	calls := 0
	api.stubChat = func(context.Context, string, []chatMessage, *chatConfig) (*chatResponse, error) {
		calls++
		return nil, fmt.Errorf("openai API status 401: invalid key")
	}

	_, err := api.chat(context.Background(), "m", nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("calls=%d want 1", calls)
	}
}

func TestChat_doesNotRetryContextCanceled(t *testing.T) {
	api := &apiClient{httpClient: &http.Client{}}
	calls := 0
	api.stubChat = func(context.Context, string, []chatMessage, *chatConfig) (*chatResponse, error) {
		calls++
		return nil, fmt.Errorf("openai request: %w", context.Canceled)
	}

	_, err := api.chat(context.Background(), "m", nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v want canceled", err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d want 1", calls)
	}
}

func TestChat_ctxCancelDuringBackoff(t *testing.T) {
	api := &apiClient{httpClient: &http.Client{}}
	api.stubChat = func(context.Context, string, []chatMessage, *chatConfig) (*chatResponse, error) {
		return nil, fmt.Errorf("openai API status 503")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := api.chat(ctx, "m", nil, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v want deadline exceeded", err)
	}
}
