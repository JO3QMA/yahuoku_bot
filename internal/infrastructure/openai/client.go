package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/spec"
)

const (
	defaultModel   = "gpt-4o-mini"
	defaultBaseURL = "https://api.openai.com/v1"
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// markdownCodeBlockRe は ```json ... ``` または ``` ... ``` のコードブロックにマッチする。
var markdownCodeBlockRe = regexp.MustCompile(`(?s)` + "```(?:json)?\\s*([\\s\\S]*?)```")

// Client は OpenAI Chat Completions API を用いて商品説明からスペックを抽出するクライアント。
type Client interface {
	ExtractSpec(ctx context.Context, title, description string) (*spec.Spec, error)
}

// specGenerator はプロンプトから JSON テキストを生成する（テストで差し替え可能）。
type specGenerator interface {
	generateSpecJSON(ctx context.Context, modelName, prompt string) (string, error)
}

type client struct {
	gen   specGenerator
	model string
}

type chatGenerator struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewClient は OpenAI 互換の Chat Completions エンドポイント向けクライアントを生成する。
// baseURL が空のときは https://api.openai.com/v1 を使う。
func NewClient(apiKey, model, baseURL string) (Client, error) {
	return NewClientWithHTTP(apiKey, model, baseURL, nil, nil)
}

// NewClientWithHTTP は HTTP クライアントと specGenerator を注入する（テスト用）。
// gen が nil のときは chatGenerator を使う。httpClient が nil のときは http.DefaultClient。
func NewClientWithHTTP(apiKey, model, baseURL string, httpClient *http.Client, gen specGenerator) (Client, error) {
	if model == "" {
		model = defaultModel
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if gen == nil {
		if httpClient == nil {
			httpClient = http.DefaultClient
		}
		gen = &chatGenerator{
			apiKey:     apiKey,
			baseURL:    baseURL,
			httpClient: httpClient,
		}
	}
	return &client{gen: gen, model: model}, nil
}

func (g *chatGenerator) generateSpecJSON(ctx context.Context, modelName, prompt string) (string, error) {
	u, err := url.Parse(g.baseURL + "/chat/completions")
	if err != nil {
		return "", fmt.Errorf("openai url: %w", err)
	}

	body := map[string]any{
		"model": modelName,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"response_format": map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
				"name":   "auction_spec",
				"strict": true,
				"schema": specJSONSchema(),
			},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("openai encode body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(raw))
	if err != nil {
		return "", fmt.Errorf("openai request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai http: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("openai read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("openai http %d: %s", resp.StatusCode, truncateForErr(string(respBody), 500))
	}

	var out chatCompletionsResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", fmt.Errorf("openai decode response: %w", err)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("no choices in openai response")
	}
	text := strings.TrimSpace(out.Choices[0].Message.Content)
	if text == "" {
		return "", fmt.Errorf("empty message content in openai response")
	}
	return text, nil
}

type chatCompletionsResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func truncateForErr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// ExtractSpec はタイトルと商品説明からPCスペック等を抽出する。
func (c *client) ExtractSpec(ctx context.Context, title, description string) (*spec.Spec, error) {
	plainDesc := htmlTagRe.ReplaceAllString(description, " ")
	if len(plainDesc) > 8000 {
		plainDesc = plainDesc[:8000] + "..."
	}

	prompt := fmt.Sprintf(`以下のヤフオク商品のタイトルと説明文から、PC・サーバー関連のスペック情報を抽出し、以下の7項目にそれぞれ独立して入れてください。

【抽出項目（各項目は独立）】
1. cpu_model_line: CPU型番 (x個数) (周波数)。例: "Xeon E-2224 (x1) (3.4GHz)"
2. core_thread_info: CPUコア数/スレッド数。例: "4コア/4スレッド"
3. socket_count: ソケット数。不明なら 0
4. memory_info: メモリー容量/枚数。例: "16GB" または "16GB x2"
5. storage_type: ストレージ種別。例: "SATA HDD", "NVMe SSD"
6. storage_capacity: ストレージ容量。例: "1TB x2"
7. other_notes: その他特記事項（OS、モデル名、状態の補足など）

【重要】
- タイトルにスペックが含まれることが多いため、タイトルを特に重視して抽出してください。
- 商品説明が空の場合は、タイトルのみから抽出してください。
- 各項目は独立させ、該当する情報だけをその項目に入れてください。不明な項目は空文字 "" または socket_count のみ 0 にしてください。
- condition は "新品" / "中古" / "不明"、shipping_free は送料無料なら true、落札者負担なら false、送料が不明なら null にしてください。

【タイトル】
%s

【商品説明】
%s`, title, plainDesc)

	prompt = sanitizeUTF8(prompt)

	text, err := c.gen.generateSpecJSON(ctx, c.model, prompt)
	if err != nil {
		return nil, err
	}

	jsonStr := extractJSONFromResponse(text)
	if strings.TrimSpace(jsonStr) == "" {
		return nil, fmt.Errorf("empty json in response")
	}

	var s spec.Spec
	if err := json.Unmarshal([]byte(jsonStr), &s); err != nil {
		return nil, fmt.Errorf("parse spec json: %w", err)
	}
	return &s, nil
}

func specJSONSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"cpu_model_line": map[string]any{
				"type":        "string",
				"description": "CPU型番 (x個数) (周波数)",
			},
			"core_thread_info": map[string]any{
				"type":        "string",
				"description": "CPUコア数/スレッド数",
			},
			"socket_count": map[string]any{
				"type":        "integer",
				"description": "ソケット数",
			},
			"memory_info": map[string]any{
				"type":        "string",
				"description": "メモリー容量/枚数",
			},
			"storage_type": map[string]any{
				"type":        "string",
				"description": "ストレージ種別",
			},
			"storage_capacity": map[string]any{
				"type":        "string",
				"description": "ストレージ容量",
			},
			"other_notes": map[string]any{
				"type":        "string",
				"description": "その他特記事項",
			},
			"condition": map[string]any{
				"type":        "string",
				"description": "商品の状態(新品/中古/不明)",
			},
			"shipping_free": map[string]any{
				"description": "送料無料かどうか。不明はnull",
				"anyOf": []any{
					map[string]any{"type": "boolean"},
					map[string]any{"type": "null"},
				},
			},
		},
		"required": []string{
			"cpu_model_line", "core_thread_info", "socket_count", "memory_info",
			"storage_type", "storage_capacity", "other_notes", "condition", "shipping_free",
		},
	}
}

// extractJSONFromResponse はレスポンステキストから JSON を抽出する。markdown のコードブロックを除去する。
func extractJSONFromResponse(text string) string {
	text = strings.TrimSpace(text)
	if m := markdownCodeBlockRe.FindStringSubmatch(text); len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return text
}

func sanitizeUTF8(s string) string {
	return strings.ToValidUTF8(s, "")
}
