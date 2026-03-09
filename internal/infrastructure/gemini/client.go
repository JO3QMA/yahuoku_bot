package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/spec"

	htmlmd "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const (
	defaultModel = "gemini-2.5-flash-lite"
)

// whitespaceRe は連続する空白文字（スペース・タブ・改行など）を1つのスペースに正規化するための正規表現。
// LLM に渡す前にテキストを圧縮してトークン消費を抑える目的で利用する。
var whitespaceRe = regexp.MustCompile(`\s+`)

// markdownCodeBlockRe は ```json ... ``` または ``` ... ``` のコードブロックにマッチする。
var markdownCodeBlockRe = regexp.MustCompile(`(?s)` + "```(?:json)?\\s*([\\s\\S]*?)```")

// CleanHTMLToText は商品説明などのHTML文字列から、テキスト解析に不要な要素を取り除きつつ、
// プレーンテキストのみを抽出して返す。
//
// 主な処理内容:
//   - <script>, <style>, <iframe>, <noscript> などを削除
//   - 残りのノードからテキストを抽出
//   - 連続する空白・改行を1つのスペースに圧縮
//   - 前後の空白を削除
func CleanHTMLToText(htmlContent string) (string, error) {
	// HTMLをDOMツリーとしてパースする。
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("parse html for text: %w", err)
	}

	// テキスト解析に不要なタグを削除する。
	doc.Find("script, style, iframe, noscript").Each(func(_ int, s *goquery.Selection) {
		s.Remove()
	})

	// 残ったノードからテキストのみを抽出する。
	text := doc.Text()

	// 連続する空白・改行・タブなどを1つのスペースに圧縮する。
	text = whitespaceRe.ReplaceAllString(text, " ")

	// 前後の不要な空白を削除する。
	text = strings.TrimSpace(text)

	return text, nil
}

// HTMLToMarkdown は商品説明などのHTML文字列を、構造（見出し・箇条書きなど）を維持しつつ
// クリーンなMarkdownへ変換して返す。
//
// オークションのテキスト解析では画像は不要なため、<img> タグはMarkdown出力から除外する。
func HTMLToMarkdown(htmlContent string) (string, error) {
	// ベースURLは特に使わないため空文字とし、相対パスはそのままにする。
	converter := htmlmd.NewConverter("", true, nil)

	// 画像タグはテキスト解析ではノイズとなるため、出力から完全に除外するルールを追加する。
	converter.AddRules(htmlmd.Rule{
		Filter: []string{"img"},
		Replacement: func(_ string, _ *goquery.Selection, _ *htmlmd.Options) *string {
			empty := ""
			return &empty
		},
	})

	md, err := converter.ConvertString(htmlContent)
	if err != nil {
		return "", fmt.Errorf("convert html to markdown: %w", err)
	}

	// 出力Markdownの前後の余計な空白・改行を削除して正規化する。
	md = strings.TrimSpace(md)

	return md, nil
}

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
	// 商品説明はHTMLで渡されることを想定し、まずはHTMLをクリーンなプレーンテキストに変換する。
	plainDesc, err := CleanHTMLToText(description)
	if err != nil {
		// HTMLパースに失敗した場合でもスペック抽出自体は継続したいため、
		// 元の説明文を簡易に正規化したテキストとしてフォールバック利用する。
		plainDesc = whitespaceRe.ReplaceAllString(description, " ")
		plainDesc = strings.TrimSpace(plainDesc)
	}

	// LLMに渡すテキスト量を抑えるため、説明文が長すぎる場合は先頭8000文字で打ち切る。
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

	// 最終的にGeminiへ渡す前にプロンプト全体をUTF-8として正規化する。
	prompt = sanitizeUTF8(prompt)

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

func sanitizeUTF8(s string) string {
	return strings.ToValidUTF8(s, "")
}

