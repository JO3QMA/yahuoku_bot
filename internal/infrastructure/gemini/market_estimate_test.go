package gemini

import (
	"context"
	"testing"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
	"google.golang.org/genai"
)

func TestMarketEstimator_Estimate(t *testing.T) {
	api, err := newGenAIAPI("test-key")
	if err != nil {
		t.Fatal(err)
	}
	api.stubGroundedSearch = func(context.Context, string, string) (string, []string, error) {
		return "GTX 1080 中古 8000-12000円", nil, nil
	}
	api.stubGenerate = func(context.Context, string, []*genai.Content, *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
		text := `{"low_price":8000,"high_price":12000,"note":"Web検索・型番一致"}`
		return textResponse(text), nil
	}

	estimator := NewMarketEstimatorWithAPI(api, "test-model")

	p := &product.Product{
		Category: product.CategoryGPU,
		Fields:   []product.Field{{Key: "model", Value: "GTX 1080"}},
	}
	est, err := estimator.Estimate(context.Background(), "GPU", "desc", p, false)
	if err != nil {
		t.Fatal(err)
	}
	if est == nil || est.LowPrice != 8000 || est.HighPrice != 12000 {
		t.Fatalf("got %+v", est)
	}
}

func textResponse(text string) *genai.GenerateContentResponse {
	return &genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{{
			Content: &genai.Content{
				Parts: []*genai.Part{{Text: text}},
			},
		}},
	}
}
