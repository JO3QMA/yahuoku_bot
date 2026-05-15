package gemini

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
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
		{
			name: "plain code block without json",
			text: "```\n{\"cpu_model\":\"Xeon\"}\n```",
			want: `{"cpu_model":"Xeon"}`,
		},
		{
			name: "with leading trailing whitespace",
			text: "  \n  ```json\n  {\"memory_gb\":24}\n  ```  \n",
			want: `{"memory_gb":24}`,
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
				Content:      &genai.Content{Parts: []genai.Part{genai.Text("x")}},
			}},
		})
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("nil content", func(t *testing.T) {
		_, err := extractTextFromResponse(&genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{{
				FinishReason: genai.FinishReasonStop,
				Content:      nil,
			}},
		})
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("empty text", func(t *testing.T) {
		_, err := extractTextFromResponse(&genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{{
				FinishReason: genai.FinishReasonStop,
				Content:      &genai.Content{Parts: []genai.Part{genai.Text("  ")}},
			}},
		})
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("no text part", func(t *testing.T) {
		_, err := extractTextFromResponse(&genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{{
				FinishReason: genai.FinishReasonStop,
				Content:      &genai.Content{Parts: []genai.Part{}},
			}},
		})
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("success unspecified", func(t *testing.T) {
		s, err := extractTextFromResponse(&genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{{
				FinishReason: genai.FinishReasonUnspecified,
				Content:      &genai.Content{Parts: []genai.Part{genai.Text(`{"cpu_model_line":""}`)}},
			}},
		})
		if err != nil || s == "" {
			t.Fatal(err, s)
		}
	})
}

type stubGen struct {
	text string
	err  error
}

func (s *stubGen) generateSpecJSON(ctx context.Context, modelName, prompt string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.text, nil
}

func TestClient_ExtractSpec_viaStub(t *testing.T) {
	c, err := NewClientWithGenerator("k", "m", &stubGen{text: `{"cpu_model_line":"X","core_thread_info":"","socket_count":0,"memory_info":"","storage_type":"","storage_capacity":"","other_notes":""}`})
	if err != nil {
		t.Fatal(err)
	}
	sp, err := c.ExtractSpec(context.Background(), "t", "d")
	if err != nil {
		t.Fatal(err)
	}
	if sp.CPUModelLine != "X" {
		t.Fatal(sp)
	}
}

func TestClient_ExtractSpec_longDescription(t *testing.T) {
	long := make([]byte, 9000)
	for i := range long {
		long[i] = 'a'
	}
	c, _ := NewClientWithGenerator("k", "m", &stubGen{text: `{"cpu_model_line":"","core_thread_info":"","socket_count":0,"memory_info":"","storage_type":"","storage_capacity":"","other_notes":""}`})
	_, err := c.ExtractSpec(context.Background(), "t", string(long))
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_ExtractSpec_errors(t *testing.T) {
	c, _ := NewClientWithGenerator("k", "m", &stubGen{err: errors.New("gen")})
	_, err := c.ExtractSpec(context.Background(), "t", "d")
	if err == nil {
		t.Fatal("expected error")
	}
	c2, _ := NewClientWithGenerator("k", "m", &stubGen{text: "   "})
	_, err = c2.ExtractSpec(context.Background(), "t", "d")
	if err == nil {
		t.Fatal("empty json")
	}
	c3, _ := NewClientWithGenerator("k", "m", &stubGen{text: `{`})
	_, err = c3.ExtractSpec(context.Background(), "t", "d")
	if err == nil {
		t.Fatal("unmarshal")
	}
}

func TestNewClient_defaultModel(t *testing.T) {
	_, err := NewClientWithGenerator("", "", &stubGen{text: `{"cpu_model_line":"","core_thread_info":"","socket_count":0,"memory_info":"","storage_type":"","storage_capacity":"","other_notes":""}`})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSpecSchema_smoke(t *testing.T) {
	s := specSchema()
	if s == nil || s.Type != genai.TypeObject {
		t.Fatal(s)
	}
}

func Test_genaiGenerator_generateHook(t *testing.T) {
	gc, err := genai.NewClient(context.Background(), option.WithAPIKey("unit-test-dummy-key"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		genaiGenerateHook = nil
		_ = gc.Close()
	})
	g := &genaiGenerator{gc: gc}

	t.Run("hook error", func(t *testing.T) {
		genaiGenerateHook = func(context.Context, *genai.GenerativeModel, []genai.Part) (*genai.GenerateContentResponse, error) {
			return nil, errors.New("x")
		}
		t.Cleanup(func() { genaiGenerateHook = nil })
		_, err := g.generateSpecJSON(context.Background(), defaultModel, "p")
		if err == nil || !strings.Contains(err.Error(), "gemini") {
			t.Fatalf("%v", err)
		}
	})

	t.Run("hook success", func(t *testing.T) {
		genaiGenerateHook = func(context.Context, *genai.GenerativeModel, []genai.Part) (*genai.GenerateContentResponse, error) {
			return &genai.GenerateContentResponse{
				Candidates: []*genai.Candidate{{
					FinishReason: genai.FinishReasonStop,
					Content: &genai.Content{Parts: []genai.Part{genai.Text(`{"cpu_model_line":"Z","core_thread_info":"","socket_count":0,"memory_info":"","storage_type":"","storage_capacity":"","other_notes":""}`)}},
				}},
			}, nil
		}
		t.Cleanup(func() { genaiGenerateHook = nil })
		text, err := g.generateSpecJSON(context.Background(), defaultModel, "p")
		if err != nil || !strings.Contains(text, "Z") {
			t.Fatalf("%v %q", err, text)
		}
	})
}
