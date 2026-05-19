package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const (
	defaultModel = "gemini-2.5-flash-lite"
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// markdownCodeBlockRe は ```json ... ``` または ``` ... ``` のコードブロックにマッチする。
var markdownCodeBlockRe = regexp.MustCompile(`(?s)` + "```(?:json)?\\s*([\\s\\S]*?)```")

// Client はGemini APIを用いて商品説明から商品情報を抽出するクライアント。
type Client interface {
	ExtractProduct(ctx context.Context, title, description string) (*product.ProductDetail, error)
}

// specGenerator はプロンプトから JSON テキストを生成する（テストで差し替え可能）。
type specGenerator interface {
	generateSpecJSON(ctx context.Context, modelName, prompt string) (string, error)
}

// client はClientの実装。
type client struct {
	gen   specGenerator
	model string
}

type genaiGenerator struct {
	gc *genai.Client
}

// genaiGenerateHook は同一パッケージのテストが GenerateContent を差し替えるためのフック（本番では常に nil）。
var genaiGenerateHook func(ctx context.Context, model *genai.GenerativeModel, parts []genai.Part) (*genai.GenerateContentResponse, error)

func (g *genaiGenerator) generateSpecJSON(ctx context.Context, modelName, prompt string) (string, error) {
	model := g.gc.GenerativeModel(modelName)
	model.ResponseMIMEType = "application/json"
	model.ResponseSchema = productSchema()

	var resp *genai.GenerateContentResponse
	var err error
	if genaiGenerateHook != nil {
		resp, err = genaiGenerateHook(ctx, model, []genai.Part{genai.Text(prompt)})
	} else {
		resp, err = model.GenerateContent(ctx, genai.Text(prompt))
	}
	if err != nil {
		return "", fmt.Errorf("gemini generate: %w", err)
	}
	return extractTextFromResponse(resp)
}

// NewClient はGemini APIクライアントを生成する。
func NewClient(apiKey string, model string) (Client, error) {
	return NewClientWithGenerator(apiKey, model, nil)
}

// NewClientWithGenerator は specGenerator を注入する（テスト用）。gen が nil のときは本番の genai クライアントを使う。
func NewClientWithGenerator(apiKey string, model string, gen specGenerator) (Client, error) {
	if model == "" {
		model = defaultModel
	}
	if gen == nil {
		gc, err := genai.NewClient(context.Background(), option.WithAPIKey(apiKey))
		if err != nil {
			return nil, fmt.Errorf("genai client: %w", err)
		}
		gen = &genaiGenerator{gc: gc}
	}
	return &client{gen: gen, model: model}, nil
}

type extractResponse struct {
	Category     string          `json:"category"`
	Condition    string          `json:"condition"`
	ShippingFree *bool           `json:"shipping_free"`
	Fields       []product.Field `json:"fields"`
}

// ExtractProduct はタイトルと商品説明からジャンル判別とテンプレート項目を抽出する。
func (c *client) ExtractProduct(ctx context.Context, title, description string) (*product.ProductDetail, error) {
	plainDesc := htmlTagRe.ReplaceAllString(description, " ")
	if len(plainDesc) > 8000 {
		plainDesc = plainDesc[:8000] + "..."
	}

	prompt := sanitizeUTF8(buildExtractPrompt(title, plainDesc))

	text, err := c.gen.generateSpecJSON(ctx, c.model, prompt)
	if err != nil {
		return nil, err
	}

	jsonStr := extractJSONFromResponse(text)
	if strings.TrimSpace(jsonStr) == "" {
		return nil, fmt.Errorf("empty json in response")
	}

	var raw extractResponse
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil, fmt.Errorf("parse product json: %w", err)
	}

	cat := product.ParseCategory(raw.Category)
	detail := &product.ProductDetail{
		Category:     cat,
		Condition:    raw.Condition,
		ShippingFree: raw.ShippingFree,
		Fields:       product.ValidateFields(cat, raw.Fields),
	}
	return detail, nil
}

func productSchema() *genai.Schema {
	categoryEnums := make([]string, len(product.AllCategories))
	for i, c := range product.AllCategories {
		categoryEnums[i] = string(c)
	}

	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"category": {
				Type: genai.TypeString,
				Enum: categoryEnums,
			},
			"condition":     {Type: genai.TypeString, Description: "商品の状態(新品/中古/不明)"},
			"shipping_free": {Type: genai.TypeBoolean, Description: "送料無料かどうか"},
			"fields": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"key":   {Type: genai.TypeString},
						"value": {Type: genai.TypeString},
					},
					Required: []string{"key", "value"},
				},
			},
		},
		Required: []string{"category", "condition", "fields"},
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

func sanitizeUTF8(s string) string {
	return strings.ToValidUTF8(s, "")
}
