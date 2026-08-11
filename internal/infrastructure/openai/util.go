package openai

import (
	"fmt"
	"regexp"
	"strings"
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// markdownCodeBlockRe は ```json ... ``` または ``` ... ``` のコードブロックにマッチする。
var markdownCodeBlockRe = regexp.MustCompile("(?s)" + "```(?:json)?\\s*([\\s\\S]*?)```")

func plainDescription(description string) string {
	plain := htmlTagRe.ReplaceAllString(description, " ")
	if len(plain) > 8000 {
		plain = plain[:8000] + "..."
	}
	return sanitizeUTF8(plain)
}

func sanitizeUTF8(s string) string {
	return strings.ToValidUTF8(s, "")
}

// extractJSONFromResponse はレスポンステキストから JSON を抽出する。
func extractJSONFromResponse(text string) string {
	text = strings.TrimSpace(text)
	if m := markdownCodeBlockRe.FindStringSubmatch(text); len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return text
}

func extractTextFromResponse(resp *chatResponse) (string, error) {
	if resp == nil || len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	ch := resp.Choices[0]
	switch ch.FinishReason {
	case "stop", "":
	default:
		return "", fmt.Errorf("finish_reason was %q (expected stop)", ch.FinishReason)
	}
	text := strings.TrimSpace(textContent(ch.Message))
	if text == "" {
		return "", fmt.Errorf("empty text in response (finish_reason: %q)", ch.FinishReason)
	}
	return text, nil
}

// textContent は chatMessage のコンテンツからテキスト部分を取り出す。
// リクエスト側は []contentPart、レスポンス側は []any（JSON デコード後）を取り得る。
func textContent(m chatMessage) string {
	switch c := m.Content.(type) {
	case string:
		return c
	case []contentPart:
		var b strings.Builder
		for _, p := range c {
			if p.Type == "text" {
				b.WriteString(p.Text)
			}
		}
		return b.String()
	case []any:
		var b strings.Builder
		for _, item := range c {
			part, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if t, _ := part["type"].(string); t != "text" {
				continue
			}
			if s, ok := part["text"].(string); ok {
				b.WriteString(s)
			}
		}
		return b.String()
	default:
		return ""
	}
}
