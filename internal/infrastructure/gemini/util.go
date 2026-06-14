package gemini

import (
	"fmt"
	"regexp"
	"strings"

	"google.golang.org/genai"
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// markdownCodeBlockRe は ```json ... ``` または ``` ... ``` のコードブロックにマッチする。
var markdownCodeBlockRe = regexp.MustCompile(`(?s)` + "```(?:json)?\\s*([\\s\\S]*?)```")

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

func extractTextFromResponse(resp *genai.GenerateContentResponse) (string, error) {
	if resp == nil || len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no candidates in response")
	}
	cand := resp.Candidates[0]
	switch cand.FinishReason {
	case genai.FinishReasonStop, genai.FinishReasonUnspecified, "":
	default:
		return "", fmt.Errorf("finish_reason was %s (expected Stop)", cand.FinishReason)
	}
	text := strings.TrimSpace(resp.Text())
	if text == "" {
		return "", fmt.Errorf("empty text in response (finish_reason: %s)", cand.FinishReason)
	}
	return text, nil
}

func extractFunctionCalls(resp *genai.GenerateContentResponse) []*genai.FunctionCall {
	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return nil
	}
	var calls []*genai.FunctionCall
	for _, part := range resp.Candidates[0].Content.Parts {
		if part.FunctionCall != nil {
			calls = append(calls, part.FunctionCall)
		}
	}
	return calls
}

// remainingMissingKeys は Stage1 の missing_keys から、Stage1/2 で既に値が得られたキーを除いた一覧を返す。
func remainingMissingKeys(s1 *stage1Result, s2 *stage2Result) []string {
	if s1 == nil || len(s1.MissingKeys) == 0 {
		return nil
	}
	filled := make(map[string]bool)
	for _, f := range s1.Fields {
		if strings.TrimSpace(f.Value) != "" {
			filled[f.Key] = true
		}
	}
	if s2 != nil {
		for _, f := range s2.ImageFields {
			if strings.TrimSpace(f.Value) != "" {
				filled[f.Key] = true
			}
		}
	}
	var out []string
	for _, k := range s1.MissingKeys {
		if !filled[k] {
			out = append(out, k)
		}
	}
	return out
}

func searchQueriesFromGrounding(resp *genai.GenerateContentResponse) []string {
	if resp == nil || len(resp.Candidates) == 0 {
		return nil
	}
	meta := resp.Candidates[0].GroundingMetadata
	if meta == nil {
		return nil
	}
	var queries []string
	for _, q := range meta.WebSearchQueries {
		if strings.TrimSpace(q) != "" {
			queries = append(queries, q)
		}
	}
	return queries
}
