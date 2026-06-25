package gemini

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/genai"
)

func Test_runLookupSpec_respects_limit(t *testing.T) {
	count := 1
	notes := []string{}
	fr := runLookupSpec(context.Background(), &genAIAPI{}, Options{MaxSearchCalls: 1}.Normalize(), &genai.FunctionCall{
		Name: "lookup_spec",
		Args: map[string]any{"query": "test", "field_key": "model"},
	}, &count, &notes)
	if fr["error"] != "search limit reached" {
		t.Fatalf("%v", fr)
	}
}

func Test_runLookupSpec_does_not_increment_on_failure(t *testing.T) {
	api := &genAIAPI{}
	api.stubGroundedSearch = func(context.Context, string, string) (string, []string, error) {
		return "", nil, errors.New("search failed")
	}

	count := 0
	notes := []string{}
	fr := runLookupSpec(context.Background(), api, Options{MaxSearchCalls: 3}.Normalize(), &genai.FunctionCall{
		Name: "lookup_spec",
		Args: map[string]any{"query": "test", "field_key": "model"},
	}, &count, &notes)
	if fr["error"] == nil {
		t.Fatalf("expected error response, got %v", fr)
	}
	if count != 0 {
		t.Fatalf("expected count 0 on failure, got %d", count)
	}
}
