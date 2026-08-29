package openai

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func jsonResponse(text string) *chatResponse {
	return &chatResponse{
		Choices: []chatChoice{{
			FinishReason: "stop",
			Message:      chatMessage{Role: "assistant", Content: text},
		}},
	}
}

func toolCallResponse(calls ...toolCall) *chatResponse {
	return &chatResponse{
		Choices: []chatChoice{{
			FinishReason: "tool_calls",
			Message:      chatMessage{Role: "assistant", ToolCalls: calls},
		}},
	}
}

// lastUserText は直近の user メッセージのテキストを返す。
func lastUserText(msgs []chatMessage) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			return textContent(msgs[i])
		}
	}
	return ""
}

func stubStageExtractor(t *testing.T, stage1 string) (*apiClient, *int) {
	t.Helper()
	api := &apiClient{httpClient: &http.Client{}}
	var toolsCalled int
	api.stubChat = func(_ context.Context, _ string, _ []chatMessage, cfg *chatConfig) (*chatResponse, error) {
		if cfg != nil && len(cfg.Tools) > 0 {
			toolsCalled++
		}
		return jsonResponse(stage1), nil
	}
	return api, &toolsCalled
}

func Test_extract_skips_search_when_resolved(t *testing.T) {
	stage1 := `{"category":"gpu","condition":"中古","shipping_free":false,"fields":[{"key":"model","value":"RTX 3080"}],"missing_keys":[]}`
	api, toolsCalled := stubStageExtractor(t, stage1)

	pd, err := NewTestClient(api, Options{}).Extract(context.Background(), "GPU", "NVIDIA GeForce RTX 3080 10GB", nil)
	if err != nil {
		t.Fatal(err)
	}
	if pd.Category != "gpu" {
		t.Fatalf("%v", pd)
	}
	if *toolsCalled > 0 {
		t.Fatal("search supplement should not run when unresolved is empty")
	}
}

func Test_extract_skips_search_when_only_installed_configuration_missing(t *testing.T) {
	stage1 := `{"category":"server","condition":"中古","shipping_free":false,"fields":[{"key":"server_model","value":"Dell R740"}],"missing_keys":["cpu_model_line","memory_info"]}`
	api, toolsCalled := stubStageExtractor(t, stage1)

	pd, err := NewTestClient(api, Options{}).Extract(context.Background(), "Dell R740", "中古サーバー", nil)
	if err != nil {
		t.Fatal(err)
	}
	if pd.Category != "server" {
		t.Fatalf("%v", pd)
	}
	if *toolsCalled > 0 {
		t.Fatal("search supplement should not run when only InstalledConfiguration keys are missing")
	}
}

func Test_extract_accepts_object_shaped_fields(t *testing.T) {
	stage1 := `{"category":"gpu","condition":"中古","shipping_free":false,"fields":{"model":"RTX 3080"},"missing_keys":[]}`
	api, toolsCalled := stubStageExtractor(t, stage1)

	pd, err := NewTestClient(api, Options{}).Extract(context.Background(), "GPU", "NVIDIA GeForce RTX 3080 10GB", nil)
	if err != nil {
		t.Fatal(err)
	}
	if pd.Category != "gpu" {
		t.Fatalf("%v", pd)
	}
	if len(pd.Fields) != 1 || pd.Fields[0].Key != "model" || pd.Fields[0].Value != "RTX 3080" {
		t.Fatalf("fields=%+v", pd.Fields)
	}
	if *toolsCalled > 0 {
		t.Fatal("search supplement should not run when unresolved is empty")
	}
}

func Test_extract_text_only(t *testing.T) {
	stage1 := `{"category":"other","condition":"","shipping_free":null,"fields":[],"missing_keys":[]}`
	api, _ := stubStageExtractor(t, stage1)

	pd, err := NewTestClient(api, Options{FastModel: "m"}).Extract(context.Background(), "t", "d", nil)
	if err != nil {
		t.Fatal(err)
	}
	if pd == nil {
		t.Fatal("nil product")
	}
}

func Test_extract_search_supplement_with_lookup(t *testing.T) {
	stage1 := `{"category":"server","condition":"","shipping_free":null,"fields":[],"missing_keys":["server_model"]}`
	agentDone := `{"fields":[{"key":"server_model","value":"PowerEdge R740"}],"done":true}`

	api := &apiClient{httpClient: &http.Client{}}
	api.stubLookup = func(_ context.Context, q string) (string, error) {
		return "Dell PowerEdge R740 の固定仕様。", nil
	}
	var calls int
	api.stubChat = func(_ context.Context, _ string, msgs []chatMessage, cfg *chatConfig) (*chatResponse, error) {
		text := lastUserText(msgs)
		if strings.Contains(text, "missing_keys") {
			return jsonResponse(stage1), nil
		}
		if cfg != nil && len(cfg.Tools) > 0 {
			calls++
			if calls == 1 {
				return toolCallResponse(toolCall{
					ID:   "call_1",
					Type: "function",
					Function: toolCallFunction{
						Name:      "lookup_spec",
						Arguments: `{"query":"Dell R740","field_key":"server_model"}`,
					},
				}), nil
			}
			return jsonResponse(agentDone), nil
		}
		t.Fatal("unexpected chat call after search supplement")
		return nil, nil
	}

	pd, err := NewTestClient(api, Options{}).Extract(context.Background(), "Dell R740", "中古サーバー", nil)
	if err != nil {
		t.Fatal(err)
	}
	if pd.Category != "server" {
		t.Fatalf("%v", pd)
	}
	found := false
	for _, f := range pd.Fields {
		if f.Key == "server_model" && f.Value == "PowerEdge R740" {
			found = true
		}
	}
	if !found {
		t.Fatalf("server_model not supplemented: %+v", pd.Fields)
	}
}
