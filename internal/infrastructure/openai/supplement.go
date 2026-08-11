package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

func lookupSpecTool() tool {
	return tool{
		Type: "function",
		Function: functionDecl{
			Name:        "lookup_spec",
			Description: "商品の型番や固定的仕様を調べる。型番同定とModelInvariant（例: ベイサイズ・対応CPU世代）の補完にのみ使用。BTO構成・搭載スペックの推測には使わない。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "検索クエリ（日本語または型番）",
					},
					"field_key": map[string]any{
						"type":        "string",
						"description": "補完対象のテンプレートフィールドキー",
					},
				},
				"required": []string{"query", "field_key"},
			},
		},
	}
}

func buildStage3FinalPrompt(title, plainDesc string, s1 *stage1Result, s2 *stage2Result, searchNotes []string) string {
	return "Web検索結果を踏まえ、補完した fields と done:true を JSON で返してください。\n\n" +
		buildStage3Prompt(title, plainDesc, s1, s2) +
		stage3SearchNotesBlock(searchNotes)
}

func stage3SearchNotesBlock(searchNotes []string) string {
	if len(searchNotes) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n【検索結果】\n")
	for _, n := range searchNotes {
		b.WriteString(n)
		b.WriteString("\n")
	}
	return b.String()
}

func runLookupSpec(ctx context.Context, api *apiClient, opts Options, call toolCall, searchCount *int, searchNotes *[]string) map[string]any {
	if *searchCount >= opts.MaxSearchCalls {
		return map[string]any{"error": "search limit reached"}
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		return map[string]any{"error": "invalid arguments"}
	}
	query, _ := args["query"].(string)
	fieldKey, _ := args["field_key"].(string)
	if strings.TrimSpace(query) == "" {
		return map[string]any{"error": "empty query"}
	}
	summary, err := api.lookupSpec(ctx, opts.AgentModel, query)
	if err != nil {
		log.Printf("[openai] lookup_spec search failed: %v", err)
		return map[string]any{"error": err.Error()}
	}
	*searchCount++
	*searchNotes = append(*searchNotes, fmt.Sprintf("[%s] %s", fieldKey, summary))
	return map[string]any{
		"summary":   summary,
		"field_key": fieldKey,
	}
}
