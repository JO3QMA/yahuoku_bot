package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const maxResponseBytes = 16 << 20 // 16MB

// chatConfig は Chat Completions リクエストの追加設定。
type chatConfig struct {
	ResponseFormat *responseFormat
	Tools          []tool
	ToolChoice     any
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
	Tools          []tool          `json:"tools,omitempty"`
	ToolChoice     any             `json:"tool_choice,omitempty"`
}

type chatMessage struct {
	Role       string     `json:"role"`
	Content    any        `json:"content"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type contentPart struct {
	Type     string        `json:"type"`
	Text     string        `json:"text,omitempty"`
	ImageURL *imageURLPart `json:"image_url,omitempty"`
}

type imageURLPart struct {
	URL string `json:"url"`
}

type toolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function toolCallFunction `json:"function"`
}

type toolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type tool struct {
	Type     string       `json:"type"`
	Function functionDecl `json:"function"`
}

type functionDecl struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
}

type chatChoice struct {
	Index        int         `json:"index"`
	Message      chatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// apiClient は OpenAI 互換 Chat Completions API の HTTP クライアント。
type apiClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	// stubChat / stubLookup はテスト用。非 nil のとき実 API を呼ばない。
	stubChat   func(context.Context, string, []chatMessage, *chatConfig) (*chatResponse, error)
	stubLookup func(context.Context, string) (string, error)
}

func newAPIClient(apiKey, baseURL string) (*apiClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("openai api key is required")
	}
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("openai base url is required")
	}
	return &apiClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 300 * time.Second},
	}, nil
}

func (a *apiClient) chat(ctx context.Context, model string, messages []chatMessage, cfg *chatConfig) (*chatResponse, error) {
	const maxRetries = 5
	backoff := []time.Duration{0, 500 * time.Millisecond, time.Second, 2 * time.Second, 4 * time.Second}
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff[attempt]):
			}
		}
		resp, err := a.chatOnce(ctx, model, messages, cfg)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !isRetryableOpenAIError(err) {
			return nil, err
		}
		if attempt < maxRetries-1 {
			log.Printf("[openai] retry %d/%d: %v", attempt+1, maxRetries, err)
		}
	}
	return nil, lastErr
}

func (a *apiClient) chatOnce(ctx context.Context, model string, messages []chatMessage, cfg *chatConfig) (*chatResponse, error) {
	if a.stubChat != nil {
		return a.stubChat(ctx, model, messages, cfg)
	}
	req := chatRequest{Model: model, Messages: messages}
	if cfg != nil {
		req.ResponseFormat = cfg.ResponseFormat
		req.Tools = cfg.Tools
		req.ToolChoice = cfg.ToolChoice
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encode chat request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.apiKey)

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai API status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out chatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode chat response: %w", err)
	}
	return &out, nil
}

func isRetryableOpenAIError(err error) bool {
	s := err.Error()
	if strings.Contains(s, "429") || strings.Contains(s, "503") {
		return true
	}
	// ネットワークエラー（接続失敗・タイムアウト）は再試行する。
	return strings.Contains(s, "openai request:")
}

func (a *apiClient) generateJSON(ctx context.Context, model, prompt string) (string, error) {
	messages := []chatMessage{{Role: "user", Content: prompt}}
	cfg := &chatConfig{ResponseFormat: &responseFormat{Type: "json_object"}}
	resp, err := a.chat(ctx, model, messages, cfg)
	if err != nil {
		return "", err
	}
	return extractTextFromResponse(resp)
}

func (a *apiClient) generateJSONWithImages(ctx context.Context, model, prompt string, images []fetchedImage) (string, error) {
	parts := []contentPart{{Type: "text", Text: prompt}}
	for _, img := range images {
		parts = append(parts, contentPart{
			Type: "image_url",
			ImageURL: &imageURLPart{
				URL: "data:" + img.MIMEType + ";base64," + base64.StdEncoding.EncodeToString(img.Data),
			},
		})
	}
	messages := []chatMessage{{Role: "user", Content: parts}}
	cfg := &chatConfig{ResponseFormat: &responseFormat{Type: "json_object"}}
	resp, err := a.chat(ctx, model, messages, cfg)
	if err != nil {
		return "", err
	}
	return extractTextFromResponse(resp)
}

func (a *apiClient) generateWithTools(ctx context.Context, model string, messages []chatMessage, tools []tool) (*chatResponse, error) {
	cfg := &chatConfig{Tools: tools, ToolChoice: "auto"}
	return a.chat(ctx, model, messages, cfg)
}

// lookupSpec は型番・固定仕様の補完情報をモデル自身の知識から取得する。
// 旧 Gemini 実装の Google Search グラウンディングに相当する機能を、
// 外部検索サービスに依存せず OpenAI 互換 API だけで実現する。
func (a *apiClient) lookupSpec(ctx context.Context, model, query string) (string, error) {
	if a.stubLookup != nil {
		return a.stubLookup(ctx, query)
	}
	prompt := "あなたは商品スペックの調査アシスタントです。次のクエリについて、商品スペックの補完に使える事実だけを簡潔に日本語でまとめてください。型番同定と固定的仕様（例: ベイサイズ、対応CPU世代）のみを扱い、推測やBTO構成の推定はしないでください。\n\nクエリ: " + query
	messages := []chatMessage{{Role: "user", Content: prompt}}
	resp, err := a.chat(ctx, model, messages, nil)
	if err != nil {
		return "", err
	}
	return extractTextFromResponse(resp)
}
