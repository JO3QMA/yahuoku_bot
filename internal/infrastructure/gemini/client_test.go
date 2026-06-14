package gemini

import (
	"context"
	"errors"
	"strings"
	"testing"

	appauction "jo3qma.com/yahoo_auctions_bot/internal/application/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
	"google.golang.org/genai"
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
	t.Run("no candidates", func(t *testing.T) {
		_, err := extractTextFromResponse(&genai.GenerateContentResponse{})
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("bad finish", func(t *testing.T) {
		_, err := extractTextFromResponse(&genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{{
				FinishReason: genai.FinishReasonMaxTokens,
				Content:      &genai.Content{Parts: []*genai.Part{{Text: "x"}}},
			}},
		})
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("success", func(t *testing.T) {
		s, err := extractTextFromResponse(&genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{{
				FinishReason: genai.FinishReasonStop,
				Content:      &genai.Content{Parts: []*genai.Part{{Text: `{"category":"other"}`}}},
			}},
		})
		if err != nil || s == "" {
			t.Fatal(err, s)
		}
	})
}

type stubRunner struct {
	result *product.Product
	err    error
}

func (s *stubRunner) run(context.Context, appauction.ExtractInput) (*product.Product, error) {
	return s.result, s.err
}

func TestClient_Extract_viaStub(t *testing.T) {
	c := NewClientWithRunner(&stubRunner{result: &product.Product{
		Category: product.CategoryServer,
		Fields:   []product.Field{{Key: "cpu_model_line", Value: "X"}},
	}})
	pd, err := c.Extract(context.Background(), appauction.ExtractInput{Title: "t", Description: "d"})
	if err != nil {
		t.Fatal(err)
	}
	if pd.Category != "server" || len(pd.Fields) != 1 || pd.Fields[0].Value != "X" {
		t.Fatal(pd)
	}
}

func TestClient_Extract_errors(t *testing.T) {
	c := NewClientWithRunner(&stubRunner{err: errors.New("gen")})
	_, err := c.Extract(context.Background(), appauction.ExtractInput{Title: "t"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestProductSchema_smoke(t *testing.T) {
	s := productSchema()
	if s == nil || s.Type != genai.TypeObject {
		t.Fatal(s)
	}
}

func Test_genAIAPI_generateHook(t *testing.T) {
	api, err := newGenAIAPI("unit-test-dummy-key")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { generateHook = nil })

	t.Run("hook error", func(t *testing.T) {
		generateHook = func(context.Context, *genai.Client, string, []*genai.Content, *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
			return nil, errors.New("x")
		}
		t.Cleanup(func() { generateHook = nil })
		_, err := api.generateJSON(context.Background(), defaultFastModel, "p", productSchema())
		if err == nil || !strings.Contains(err.Error(), "x") {
			t.Fatalf("%v", err)
		}
	})

	t.Run("hook success", func(t *testing.T) {
		generateHook = func(context.Context, *genai.Client, string, []*genai.Content, *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					FinishReason: genai.FinishReasonStop,
					Content:      &genai.Content{Parts: []*genai.Part{{Text: `{"category":"server","condition":"","fields":[{"key":"cpu_model_line","value":"Z"}]}`}}},
				}},
			}, nil
		}
		t.Cleanup(func() { generateHook = nil })
		text, err := api.generateJSON(context.Background(), defaultFastModel, "p", productSchema())
		if err != nil || !strings.Contains(text, "Z") {
			t.Fatalf("%v %q", err, text)
		}
	})
}

func Test_pipeline_stage1_only(t *testing.T) {
	stage1 := `{"category":"other","condition":"","shipping_free":null,"fields":[],"missing_keys":[],"candidate_queries":[]}`
	stage4 := `{"category":"other","condition":"","fields":[]}`
	call := 0
	generateHook = func(_ context.Context, _ *genai.Client, _ string, _ []*genai.Content, cfg *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
		call++
		text := stage4
		if cfg != nil && cfg.ResponseSchema != nil {
			if _, ok := cfg.ResponseSchema.Properties["missing_keys"]; ok {
				text = stage1
			}
		}
		return jsonResponse(text), nil
	}
	t.Cleanup(func() { generateHook = nil })

	api, err := newGenAIAPI("k")
	if err != nil {
		t.Fatal(err)
	}
	c := NewClientWithRunner(newPipeline(api, Options{FastModel: "m"}))
	pd, err := c.Extract(context.Background(), appauction.ExtractInput{Title: "t", Description: "d"})
	if err != nil {
		t.Fatal(err)
	}
	if pd == nil {
		t.Fatal("nil product")
	}
	if call < 2 {
		t.Fatalf("expected at least 2 api calls, got %d", call)
	}
}

func jsonResponse(text string) *genai.GenerateContentResponse {
	return &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{{
			FinishReason: genai.FinishReasonStop,
			Content:      &genai.Content{Parts: []*genai.Part{{Text: text}}},
		}},
	}
}
