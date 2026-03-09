package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/spec"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const (
	defaultModel = "gemini-2.5-flash-lite"
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// markdownCodeBlockRe は ```json ... ``` または ``` ... ``` のコードブロックにマッチする。
var markdownCodeBlockRe = regexp.MustCompile(`(?s)` + "```(?:json)?\\s*([\\s\\S]*?)```")

// Client はGemini APIを用いて商品説明からスペックを抽出するクライアント。
type Client interface {
	ExtractSpec(ctx context.Context, title, description string) (*spec.Spec, error)
}

// client はClientの実装。
type client struct {
	genaiClient *genai.Client
	model       string
}

// NewClient はGemini APIクライアントを生成する。
func NewClient(apiKey string, model string) (Client, error) {
	if model == "" {
		model = defaultModel
	}
	c, err := genai.NewClient(context.Background(), option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("genai client: %w", err)
	}
	return &client{genaiClient: c, model: model}, nil
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
- condition は "新品" / "中古" / "不明"、shipping_free は送料無料なら true、落札者負担なら false にしてください。

【タイトル】
%s

【商品説明】
%s`, title, plainDesc)

	model := c.genaiClient.GenerativeModel(c.model)
	model.ResponseMIMEType = "application/json"
	model.ResponseSchema = specSchema()

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("gemini generate: %w", err)
	}

	text, err := extractTextFromResponse(resp)
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

func specSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"cpu_model_line":   {Type: genai.TypeString, Description: "CPU型番 (x個数) (周波数)"},
			"core_thread_info": {Type: genai.TypeString, Description: "CPUコア数/スレッド数"},
			"socket_count":     {Type: genai.TypeInteger, Description: "ソケット数"},
			"memory_info":      {Type: genai.TypeString, Description: "メモリー容量/枚数"},
			"storage_type":    {Type: genai.TypeString, Description: "ストレージ種別"},
			"storage_capacity": {Type: genai.TypeString, Description: "ストレージ容量"},
			"other_notes":      {Type: genai.TypeString, Description: "その他特記事項"},
			"condition":        {Type: genai.TypeString, Description: "商品の状態(新品/中古/不明)"},
			"shipping_free":    {Type: genai.TypeBoolean, Description: "送料無料かどうか"},
		},
		Required: []string{"cpu_model_line", "core_thread_info", "socket_count", "memory_info", "storage_type", "storage_capacity", "other_notes"},
	}
}

// extractTextFromResponse はレスポンスからテキストを取得する。FinishReason と空テキストをチェックする。
func extractTextFromResponse(resp *genai.GenerateContentResponse) (string, error) {
	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}
	cand := resp.Candidates[0]

	if cand.FinishReason != genai.FinishReasonStop && cand.FinishReason != genai.FinishReasonUnspecified {
		return "", fmt.Errorf("finish_reason was %s (expected Stop)", cand.FinishReason.String())
	}

	if cand.Content == nil {
		return "", fmt.Errorf("no content in response (finish_reason: %s)", cand.FinishReason.String())
	}

	for _, p := range cand.Content.Parts {
		if t, ok := p.(genai.Text); ok {
			text := string(t)
			if strings.TrimSpace(text) == "" {
				return "", fmt.Errorf("empty text in response (finish_reason: %s)", cand.FinishReason.String())
			}
			return text, nil
		}
	}
	return "", fmt.Errorf("no text part in response")
}

// extractJSONFromResponse はレスポンステキストから JSON を抽出する。markdown のコードブロックを除去する。
func extractJSONFromResponse(text string) string {
	text = strings.TrimSpace(text)
	if m := markdownCodeBlockRe.FindStringSubmatch(text); len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return text
}
