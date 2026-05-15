package openai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_extractJSONFromResponse(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "plain json",
			text: `{"cpu_model_line":"X","core_thread_info":"","socket_count":0}`,
			want: `{"cpu_model_line":"X","core_thread_info":"","socket_count":0}`,
		},
		{
			name: "json in markdown code block",
			text: "```json\n{\"cpu_model_line\":\"Y\"}\n```",
			want: `{"cpu_model_line":"Y"}`,
		},
		{
			name: "plain code block without json",
			text: "```\n{\"cpu_model_line\":\"Z\"}\n```",
			want: `{"cpu_model_line":"Z"}`,
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

const minSpecJSON = `{"cpu_model_line":"X","core_thread_info":"","socket_count":0,"memory_info":"","storage_type":"","storage_capacity":"","other_notes":"","condition":"","shipping_free":null}`

func TestClient_ExtractSpec_viaStub(t *testing.T) {
	c, err := NewClientWithHTTP("k", "m", "", nil, &stubGen{text: minSpecJSON})
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
	c, err := NewClientWithHTTP("k", "m", "", nil, &stubGen{text: `{"cpu_model_line":"","core_thread_info":"","socket_count":0,"memory_info":"","storage_type":"","storage_capacity":"","other_notes":"","condition":"","shipping_free":null}`})
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.ExtractSpec(context.Background(), "t", string(long))
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_ExtractSpec_errors(t *testing.T) {
	c, _ := NewClientWithHTTP("k", "m", "", nil, &stubGen{err: errors.New("gen")})
	_, err := c.ExtractSpec(context.Background(), "t", "d")
	if err == nil {
		t.Fatal("expected error")
	}
	c2, _ := NewClientWithHTTP("k", "m", "", nil, &stubGen{text: "   "})
	_, err = c2.ExtractSpec(context.Background(), "t", "d")
	if err == nil {
		t.Fatal("empty json")
	}
	c3, _ := NewClientWithHTTP("k", "m", "", nil, &stubGen{text: `{`})
	_, err = c3.ExtractSpec(context.Background(), "t", "d")
	if err == nil {
		t.Fatal("unmarshal")
	}
}

func TestNewClient_defaultModel(t *testing.T) {
	_, err := NewClientWithHTTP("", "", "", nil, &stubGen{text: minSpecJSON})
	if err != nil {
		t.Fatal(err)
	}
}

func TestClient_chatCompletions_httptest(t *testing.T) {
	payload := minSpecJSON
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			t.Errorf("method %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{
				"message": map[string]any{"content": payload},
			}},
		})
	}))
	t.Cleanup(srv.Close)

	hc := srv.Client()
	c, err := NewClientWithHTTP("k", "gpt-4o-mini", srv.URL, hc, nil)
	if err != nil {
		t.Fatal(err)
	}
	sp, err := c.ExtractSpec(context.Background(), "title", "desc")
	if err != nil {
		t.Fatal(err)
	}
	if sp.CPUModelLine != "X" {
		t.Fatalf("%+v", sp)
	}
}

func TestClient_chatCompletions_httpError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	t.Cleanup(srv.Close)

	c, err := NewClientWithHTTP("k", "m", srv.URL, srv.Client(), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.ExtractSpec(context.Background(), "t", "d")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Fatalf("%v", err)
	}
}

func TestClient_chatCompletions_noChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClientWithHTTP("k", "m", srv.URL, srv.Client(), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.ExtractSpec(context.Background(), "t", "d")
	if err == nil || !strings.Contains(err.Error(), "choices") {
		t.Fatalf("%v", err)
	}
}

func Test_specJSONSchema_smoke(t *testing.T) {
	s := specJSONSchema()
	if s == nil || s["type"] != "object" {
		t.Fatal(s)
	}
}
