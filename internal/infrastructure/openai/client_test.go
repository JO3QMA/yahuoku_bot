package openai

import (
	"context"
	"errors"
	"strings"
	"testing"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

func Test_extractJSONFromResponse(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "plain json",
			text: `{"cpu_model":"Xeon E3-1230 v6","core_count":4}`,
			want: `{"cpu_model":"Xeon E3-1230 v6","core_count":4}`,
		},
		{
			name: "json in markdown code block",
			text: "```json\n{\"cpu_model\":\"Xeon E3-1230 v6\",\"core_count\":4}\n```",
			want: `{"cpu_model":"Xeon E3-1230 v6","core_count":4}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSONFromResponse(tt.text)
			if got != tt.want {
				t.Errorf("extractJSONFromResponse() = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_sanitizeUTF8(t *testing.T) {
	s := string([]byte{0xff, 0xfe}) + "ok"
	got := sanitizeUTF8(s)
	if got != "ok" {
		t.Fatalf("%q", got)
	}
}

func Test_extractTextFromResponse(t *testing.T) {
	t.Run("no choices", func(t *testing.T) {
		_, err := extractTextFromResponse(&chatResponse{})
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("bad finish", func(t *testing.T) {
		_, err := extractTextFromResponse(&chatResponse{
			Choices: []chatChoice{{
				FinishReason: "length",
				Message:      chatMessage{Content: "x"},
			}},
		})
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("success", func(t *testing.T) {
		s, err := extractTextFromResponse(&chatResponse{
			Choices: []chatChoice{{
				FinishReason: "stop",
				Message:      chatMessage{Content: `{"category":"other"}`},
			}},
		})
		if err != nil || s == "" {
			t.Fatal(err, s)
		}
	})
}

type stubClient struct {
	result *product.Product
	err    error
}

func (s stubClient) Extract(context.Context, string, string, []string) (*product.Product, error) {
	return s.result, s.err
}

func TestClient_Extract_viaStub(t *testing.T) {
	c := stubClient{result: &product.Product{
		Category: product.CategoryServer,
		Fields:   []product.Field{{Key: "cpu_model_line", Value: "X"}},
	}}
	pd, err := c.Extract(context.Background(), "t", "d", nil)
	if err != nil {
		t.Fatal(err)
	}
	if pd.Category != "server" || len(pd.Fields) != 1 || pd.Fields[0].Value != "X" {
		t.Fatal(pd)
	}
}

func TestClient_Extract_errors(t *testing.T) {
	c := stubClient{err: errors.New("gen")}
	_, err := c.Extract(context.Background(), "t", "", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func Test_apiClient_generateJSON_viaStub(t *testing.T) {
	api := &apiClient{httpClient: nil}

	t.Run("stub error", func(t *testing.T) {
		api.stubChat = func(context.Context, string, []chatMessage, *chatConfig) (*chatResponse, error) {
			return nil, errors.New("x")
		}
		_, err := api.generateJSON(context.Background(), defaultFastModel, "p")
		if err == nil || !strings.Contains(err.Error(), "x") {
			t.Fatalf("%v", err)
		}
	})

	t.Run("stub success", func(t *testing.T) {
		api.stubChat = func(_ context.Context, _ string, msgs []chatMessage, cfg *chatConfig) (*chatResponse, error) {
			if cfg == nil || cfg.ResponseFormat == nil || cfg.ResponseFormat.Type != "json_object" {
				t.Fatalf("expected json_object response format, got %+v", cfg)
			}
			if len(msgs) != 1 || msgs[0].Role != "user" {
				t.Fatalf("messages: %+v", msgs)
			}
			return jsonResponse(`{"category":"server","condition":"","fields":[{"key":"cpu_model_line","value":"Z"}]}`), nil
		}
		text, err := api.generateJSON(context.Background(), defaultFastModel, "p")
		if err != nil || !strings.Contains(text, "Z") {
			t.Fatalf("%v %q", err, text)
		}
	})
}
