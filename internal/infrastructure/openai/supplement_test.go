package openai

import (
	"context"
	"errors"
	"strings"
	"testing"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

func Test_runLookupSpec_respects_limit(t *testing.T) {
	count := 1
	notes := []string{}
	fr := runLookupSpec(context.Background(), &apiClient{}, Options{MaxSearchCalls: 1}.Normalize(), toolCall{
		Function: toolCallFunction{Name: "lookup_spec", Arguments: `{"query":"test","field_key":"model"}`},
	}, &count, &notes)
	if fr["error"] != "search limit reached" {
		t.Fatalf("%v", fr)
	}
}

func Test_runLookupSpec_does_not_increment_on_failure(t *testing.T) {
	api := &apiClient{}
	api.stubLookup = func(context.Context, string) (string, error) {
		return "", errors.New("search failed")
	}

	count := 0
	notes := []string{}
	fr := runLookupSpec(context.Background(), api, Options{MaxSearchCalls: 3}.Normalize(), toolCall{
		Function: toolCallFunction{Name: "lookup_spec", Arguments: `{"query":"test","field_key":"model"}`},
	}, &count, &notes)
	if fr["error"] == nil {
		t.Fatalf("expected error response, got %v", fr)
	}
	if count != 0 {
		t.Fatalf("expected count 0 on failure, got %d", count)
	}
}

func Test_runLookupSpec_success(t *testing.T) {
	api := &apiClient{}
	api.stubLookup = func(_ context.Context, q string) (string, error) {
		return "summary for " + q, nil
	}

	count := 0
	notes := []string{}
	fr := runLookupSpec(context.Background(), api, Options{MaxSearchCalls: 3}.Normalize(), toolCall{
		Function: toolCallFunction{Name: "lookup_spec", Arguments: `{"query":"Dell R740","field_key":"server_model"}`},
	}, &count, &notes)
	if fr["error"] != nil {
		t.Fatalf("%v", fr)
	}
	if count != 1 {
		t.Fatalf("count=%d", count)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "[server_model]") {
		t.Fatalf("notes=%v", notes)
	}
	if fr["field_key"] != "server_model" {
		t.Fatalf("%v", fr)
	}
}

func Test_lookupSpecTool(t *testing.T) {
	tl := lookupSpecTool()
	if tl.Type != "function" || tl.Function.Name != "lookup_spec" {
		t.Fatalf("%+v", tl)
	}
	params := tl.Function.Parameters
	if params["type"] != "object" {
		t.Fatalf("params: %v", params)
	}
	if props, ok := params["properties"].(map[string]any); !ok || props["query"] == nil || props["field_key"] == nil {
		t.Fatalf("properties: %v", params)
	}
	required, ok := params["required"].([]string)
	if !ok || len(required) != 2 {
		t.Fatalf("required: %v", params)
	}
}

func Test_parseAgentFieldsJSON(t *testing.T) {
	got, err := parseAgentFieldsJSON(`{"fields":[{"key":"server_model","value":"R740"}],"done":true}`)
	if err != nil {
		t.Fatal(err)
	}
	if !bool(got.Done) || len(got.Fields) != 1 || got.Fields[0].Value != "R740" {
		t.Fatalf("%+v", got)
	}
}

func Test_toProduct(t *testing.T) {
	pd := toProduct(&extractResponse{
		Category:  "server",
		Condition: "中古",
		Fields:    fieldList{{Key: "server_model", Value: "R740"}},
	})
	if pd.Category != product.CategoryServer {
		t.Fatalf("%v", pd.Category)
	}
}
