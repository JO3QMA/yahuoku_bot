package gemini

import (
	"context"
	"fmt"
	"strings"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
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

func (p *pipeline) runStage3(ctx context.Context, title, plainDesc string, s1 *stage1Result, s2 *stage2Result) ([]product.Field, []string, error) {
	// lookup_spec の Function Calling ループ。検索上限に達するか done:true の JSON が返るまで繰り返す。
	contents := []*genai.Content{
		genai.NewContentFromText(buildStage3Prompt(title, plainDesc, s1, s2), genai.RoleUser),
	}

	var searchNotes []string
	searchCount := 0
	maxTurns := p.opts.MaxSearchCalls*2 + 3

	for turn := 0; turn < maxTurns; turn++ {
		resp, err := p.api.generateWithTools(ctx, p.opts.AgentModel, contents, agentToolConfig())
		if err != nil {
			return nil, searchNotes, fmt.Errorf("stage3 turn %d: %w", turn, err)
		}
		if len(resp.Candidates) == 0 {
			return nil, searchNotes, fmt.Errorf("stage3: no candidates")
		}
		cand := resp.Candidates[0]

		if calls := extractFunctionCalls(resp); len(calls) > 0 {
			if cand.Content != nil {
				contents = append(contents, cand.Content)
			}
			for _, call := range calls {
				if call.Name != "lookup_spec" {
					continue
				}
				fr := p.executeLookupSpec(ctx, call, &searchCount, &searchNotes)
				contents = append(contents, genai.NewContentFromFunctionResponse(call.Name, fr, genai.RoleUser))
			}
			continue
		}

		text := strings.TrimSpace(resp.Text())
		if text != "" {
			if parsed, err := parseAgentFieldsJSON(text); err == nil {
				return parsed.Fields, searchNotes, nil
			}
		}
		break
	}

	// FC ループで JSON が得られない場合、検索メモを踏まえた最終 JSON 生成にフォールバックする。
	finalPrompt := buildStage3FinalPrompt(title, plainDesc, s1, s2, searchNotes)
	text, err := p.api.generateJSON(ctx, p.opts.AgentModel, finalPrompt, agentFieldsSchema())
	if err != nil {
		return nil, searchNotes, fmt.Errorf("stage3 final: %w", err)
	}
	parsed, err := parseAgentFieldsJSON(text)
	if err != nil {
		return nil, searchNotes, err
	}
	return parsed.Fields, searchNotes, nil
}

func (p *pipeline) executeLookupSpec(ctx context.Context, call *genai.FunctionCall, searchCount *int, searchNotes *[]string) map[string]any {
	return runLookupSpec(ctx, p.api, p.opts, call, searchCount, searchNotes)
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
