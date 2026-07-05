package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	appmarket "jo3qma.com/yahoo_auctions_bot/internal/application/market"
	domainmarket "jo3qma.com/yahoo_auctions_bot/internal/domain/market"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
	"google.golang.org/genai"
)

const marketEstimateTimeoutSec = 20

// MarketEstimator は Gemini + Web 検索で MarketEstimate を返す。
type MarketEstimator struct {
	api   *genAIAPI
	model string
}

// NewMarketEstimator は MarketEstimator を生成する。
func NewMarketEstimator(apiKey, model string) (*MarketEstimator, error) {
	api, err := newGenAIAPI(apiKey)
	if err != nil {
		return nil, err
	}
	if model == "" {
		model = defaultAgentModel
	}
	return &MarketEstimator{api: api, model: model}, nil
}

// NewMarketEstimatorWithAPI はテスト用に API を注入する。
func NewMarketEstimatorWithAPI(api *genAIAPI, model string) *MarketEstimator {
	if model == "" {
		model = defaultAgentModel
	}
	return &MarketEstimator{api: api, model: model}
}

// Estimate は Web 検索ベースで MarketEstimate を返す。
func (e *MarketEstimator) Estimate(ctx context.Context, title, description string, p *product.Product, identityMissing bool) (*domainmarket.MarketEstimate, error) {
	if e == nil || e.api == nil || p == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(ctx, marketEstimateTimeoutSec*time.Second)
	defer cancel()

	query := buildMarketSearchQuery(title, description, p)
	summary, _, err := e.api.groundedSearch(ctx, e.model, query)
	if err != nil {
		return nil, fmt.Errorf("market grounded search: %w", err)
	}
	if summary == "" {
		log.Printf("[market_estimate] groundedSearch returned empty summary for query: %s", query)
	}

	text, err := e.api.generateJSON(ctx, e.model, buildMarketEstimatePrompt(title, description, p, identityMissing, summary), marketEstimateSchema())
	if err != nil {
		return nil, fmt.Errorf("market estimate json: %w", err)
	}

	var parsed marketEstimateJSON
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil, fmt.Errorf("parse market estimate: %w", err)
	}
	if parsed.LowPrice <= 0 || parsed.HighPrice <= 0 {
		log.Printf("[market_estimate] invalid price range: low=%d, high=%d", parsed.LowPrice, parsed.HighPrice)
		return nil, nil
	}
	low, high := parsed.LowPrice, parsed.HighPrice
	if low > high {
		low, high = high, low
	}
	note := strings.TrimSpace(parsed.Note)
	if note == "" {
		if identityMissing {
			note = "Web検索・型番未特定"
		} else {
			note = "Web検索・型番一致"
		}
	}
	return &domainmarket.MarketEstimate{LowPrice: low, HighPrice: high, Note: note}, nil
}

var _ appmarket.WebMarketEstimator = (*MarketEstimator)(nil)

type marketEstimateJSON struct {
	LowPrice  int64  `json:"low_price"`
	HighPrice int64  `json:"high_price"`
	Note      string `json:"note"`
}

func marketEstimateSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"low_price":  {Type: genai.TypeInteger, Description: "想定価格帯の下限（円、送料別）"},
			"high_price": {Type: genai.TypeInteger, Description: "想定価格帯の上限（円、送料別）"},
			"note": {
				Type:        genai.TypeString,
				Description: "根拠の一言（例: Web検索・型番未特定）",
			},
		},
		Required: []string{"low_price", "high_price", "note"},
	}
}

func buildMarketSearchQuery(title, description string, p *product.Product) string {
	parts := []string{strings.TrimSpace(truncateString(title, 500))}
	if _, v, ok := domainmarket.IdentityValue(p); ok {
		parts = append(parts, v)
	}
	if plain := strings.TrimSpace(plainDescription(description)); plain != "" {
		parts = append(parts, plain)
	}
	parts = append(parts, "ヤフオク 落札 相場 中古")
	return strings.Join(parts, " ")
}

func buildMarketEstimatePrompt(title, description string, p *product.Product, identityMissing bool, searchSummary string) string {
	var b strings.Builder
	b.WriteString("あなたは日本のオークション相場に詳しいアシスタントです。\n")
	b.WriteString("ヤフオクの類似商品の落札価格（送料別）の相場帯を推定してください。\n")
	b.WriteString("落札見込みではなく、市場相場の価格帯（下限・上限）を返してください。\n\n")
	b.WriteString("## 出品情報\n")
	b.WriteString("タイトル: " + truncateString(sanitizeUTF8(title), 500) + "\n")
	if d := plainDescription(description); d != "" {
		b.WriteString("説明: " + d + "\n")
	}
	if p != nil {
		b.WriteString("Category: " + string(p.Category) + "\n")
		if p.Condition != "" {
			b.WriteString("状態: " + p.Condition + "\n")
		}
		for _, f := range p.Fields {
			if f.Value != "" && f.Value != "不明" {
				b.WriteString(f.Key + ": " + f.Value + "\n")
			}
		}
	}
	if identityMissing {
		b.WriteString("\n型番（IdentityField）は未特定です。タイトル・説明・検索結果から推定してください。\n")
	} else {
		b.WriteString("\n型番（IdentityField）は特定済みです。同一型番の相場を優先してください。\n")
	}
	if searchSummary != "" {
		b.WriteString("\n## Web検索サマリ\n")
		b.WriteString(searchSummary)
		b.WriteString("\n")
	}
	b.WriteString("\nnote には必ず「Web検索」を含め、型番未特定なら「型番未特定」、特定済みなら「型番一致」を含めてください。\n")
	return b.String()
}
