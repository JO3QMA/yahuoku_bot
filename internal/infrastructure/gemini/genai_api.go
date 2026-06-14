package gemini

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

// generateHook はテストが GenerateContent を差し替えるためのフック（本番では nil）。
var generateHook func(
	ctx context.Context,
	client *genai.Client,
	model string,
	contents []*genai.Content,
	config *genai.GenerateContentConfig,
) (*genai.GenerateContentResponse, error)

type genAIAPI struct {
	client *genai.Client
}

func newGenAIAPI(apiKey string) (*genAIAPI, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("gemini api key is required")
	}
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("genai client: %w", err)
	}
	return &genAIAPI{client: client}, nil
}

func (a *genAIAPI) generate(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	if generateHook != nil {
		return generateHook(ctx, a.client, model, contents, config)
	}
	resp, err := a.client.Models.GenerateContent(ctx, model, contents, config)
	if err != nil {
		return nil, fmt.Errorf("gemini generate: %w", err)
	}
	return resp, nil
}

func (a *genAIAPI) generateJSON(ctx context.Context, model, prompt string, schema *genai.Schema) (string, error) {
	contents := []*genai.Content{
		genai.NewContentFromText(prompt, genai.RoleUser),
	}
	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   schema,
	}
	resp, err := a.generate(ctx, model, contents, config)
	if err != nil {
		return "", err
	}
	return extractTextFromResponse(resp)
}

func (a *genAIAPI) generateJSONWithImages(ctx context.Context, model, prompt string, images []fetchedImage, schema *genai.Schema) (string, error) {
	parts := []*genai.Part{genai.NewPartFromText(prompt)}
	for _, img := range images {
		parts = append(parts, genai.NewPartFromBytes(img.Data, img.MIMEType))
	}
	contents := []*genai.Content{{Role: genai.RoleUser, Parts: parts}}
	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   schema,
	}
	resp, err := a.generate(ctx, model, contents, config)
	if err != nil {
		return "", err
	}
	return extractTextFromResponse(resp)
}

func (a *genAIAPI) generateWithTools(ctx context.Context, model string, contents []*genai.Content, config *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	return a.generate(ctx, model, contents, config)
}

// groundedSearchHook はテストが groundedSearch を差し替えるためのフック（本番では nil）。
var groundedSearchHook func(ctx context.Context, api *genAIAPI, model, query string) (summary string, queries []string, err error)

func (a *genAIAPI) groundedSearch(ctx context.Context, model, query string) (summary string, queries []string, err error) {
	if groundedSearchHook != nil {
		return groundedSearchHook(ctx, a, model, query)
	}
	contents := []*genai.Content{
		genai.NewContentFromText(
			"次のクエリについて、商品スペックの補完に使える事実だけを簡潔に日本語でまとめてください。\n\nクエリ: "+query,
			genai.RoleUser,
		),
	}
	config := &genai.GenerateContentConfig{
		Tools: []*genai.Tool{{GoogleSearch: &genai.GoogleSearch{}}},
	}
	resp, err := a.generate(ctx, model, contents, config)
	if err != nil {
		return "", nil, err
	}
	text, err := extractTextFromResponse(resp)
	if err != nil {
		return "", nil, err
	}
	return text, searchQueriesFromGrounding(resp), nil
}
