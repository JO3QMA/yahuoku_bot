package gemini

import (
	"context"
	"fmt"
	"log"
	"strings"

	"google.golang.org/genai"
)

func lookupSpecDeclaration() *genai.FunctionDeclaration {
	return &genai.FunctionDeclaration{
		Name:        "lookup_spec",
		Description: "商品の型番やスペックをWeb検索で調べる。不足フィールドの補完にのみ使用する。",
		Parameters: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"query": {
					Type:        genai.TypeString,
					Description: "検索クエリ（日本語または型番）",
				},
				"field_key": {
					Type:        genai.TypeString,
					Description: "補完対象のテンプレートフィールドキー",
				},
			},
			Required: []string{"query", "field_key"},
		},
	}
}

func agentToolConfig() *genai.GenerateContentConfig {
	return &genai.GenerateContentConfig{
		Tools: []*genai.Tool{{
			FunctionDeclarations: []*genai.FunctionDeclaration{lookupSpecDeclaration()},
		}},
		ToolConfig: &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode: genai.FunctionCallingConfigModeAuto,
			},
		},
	}
}

func buildStage3FinalPrompt(title, plainDesc string, s1 *stage1Result, s2 *stage2Result, searchNotes []string) string {
	var b strings.Builder
	b.WriteString(`Web検索結果を踏まえ、補完した fields と done:true を JSON で返してください。

`)
	b.WriteString(buildStage3Prompt(title, plainDesc, s1, s2))
	if len(searchNotes) > 0 {
		b.WriteString("\n\n【検索結果】\n")
		for _, n := range searchNotes {
			b.WriteString(n)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func runLookupSpec(ctx context.Context, api *genAIAPI, opts Options, call *genai.FunctionCall, searchCount *int, searchNotes *[]string) map[string]any {
	if *searchCount >= opts.MaxSearchCalls {
		return map[string]any{"error": "search limit reached"}
	}
	query, _ := call.Args["query"].(string)
	fieldKey, _ := call.Args["field_key"].(string)
	if strings.TrimSpace(query) == "" {
		return map[string]any{"error": "empty query"}
	}
	summary, queries, err := api.groundedSearch(ctx, opts.AgentModel, query)
	if err != nil {
		log.Printf("[gemini] lookup_spec search failed: %v", err)
		return map[string]any{"error": err.Error()}
	}
	*searchCount++
	if len(queries) > 0 {
		log.Printf("[gemini] grounding search queries for %q: %v", fieldKey, queries)
	}
	*searchNotes = append(*searchNotes, fmt.Sprintf("[%s] %s", fieldKey, summary))
	return map[string]any{
		"summary":   summary,
		"field_key": fieldKey,
		"sources":   queries,
	}
}
