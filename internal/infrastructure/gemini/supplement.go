package gemini

import (
	"context"
	"fmt"
	"log"
	"strings"

	"google.golang.org/genai"
)

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
