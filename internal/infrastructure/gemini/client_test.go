package gemini

import (
	"context"
	"errors"
	"strings"
	"testing"

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

func Test_truncateString_maxZero(t *testing.T) {
	if got := truncateString("hello", 0); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := truncateString("", 0); got != "" {
		t.Fatalf("got %q", got)
	}
}

func Test_escapeForQuotedPrompt(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{`GPU "Special"`, `GPU \"Special\"`},
		{"line1\nline2", `line1\nline2`},
		{`path\to\file`, `path\\to\\file`},
	}
	for _, tt := range tests {
		if got := escapeForQuotedPrompt(tt.in); got != tt.want {
			t.Fatalf("escapeForQuotedPrompt(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func Test_truncateString_multibyte(t *testing.T) {
	s := strings.Repeat("あ", 501)
	got := truncateString(s, 500)
	runes := []rune(got)
	if len(runes) != 503 { // 500 + "..."
		t.Fatalf("len=%d got %q", len(runes), got)
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("suffix missing: %q", got)
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

type stubClient struct {
	result *product.Product
	err    error
}

func (s stubClient) Extract(context.Context, product.ExtractInput) (*product.Product, error) {
	return s.result, s.err
}

func TestClient_Extract_viaStub(t *testing.T) {
	c := stubClient{result: &product.Product{
		Category: product.CategoryServer,
		Fields:   []product.Field{{Key: "cpu_model_line", Value: "X"}},
	}}
	pd, err := c.Extract(context.Background(), product.ExtractInput{Title: "t", Description: "d"})
	if err != nil {
		t.Fatal(err)
	}
	if pd.Category != "server" || len(pd.Fields) != 1 || pd.Fields[0].Value != "X" {
		t.Fatal(pd)
	}
}

func TestClient_Extract_errors(t *testing.T) {
	c := stubClient{err: errors.New("gen")}
	_, err := c.Extract(context.Background(), product.ExtractInput{Title: "t"})
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

func Test_genAIAPI_stubGenerate(t *testing.T) {
	api, err := newGenAIAPI("unit-test-dummy-key")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("stub error", func(t *testing.T) {
		api.stubGenerate = func(context.Context, string, []*genai.Content, *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
			return nil, errors.New("x")
		}
		_, err := api.generateJSON(context.Background(), defaultFastModel, "p", productSchema())
		if err == nil || !strings.Contains(err.Error(), "x") {
			t.Fatalf("%v", err)
		}
	})

	t.Run("stub success", func(t *testing.T) {
		api.stubGenerate = func(context.Context, string, []*genai.Content, *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					FinishReason: genai.FinishReasonStop,
					Content:      &genai.Content{Parts: []*genai.Part{{Text: `{"category":"server","condition":"","fields":[{"key":"cpu_model_line","value":"Z"}]}`}}},
				}},
			}, nil
		}
		text, err := api.generateJSON(context.Background(), defaultFastModel, "p", productSchema())
		if err != nil || !strings.Contains(text, "Z") {
			t.Fatalf("%v %q", err, text)
		}
	})
}

func jsonResponse(text string) *genai.GenerateContentResponse {
	return &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{{
			FinishReason: genai.FinishReasonStop,
			Content:      &genai.Content{Parts: []*genai.Part{{Text: text}}},
		}},
	}
}
