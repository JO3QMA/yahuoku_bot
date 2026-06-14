package gemini

import (
	"encoding/json"
	"fmt"
	"strings"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

type stage1Result struct {
	Category         string          `json:"category"`
	Condition        string          `json:"condition"`
	ShippingFree     *bool           `json:"shipping_free"`
	Fields           []product.Field `json:"fields"`
	MissingKeys      []string        `json:"missing_keys"`
	CandidateQueries []string        `json:"candidate_queries"`
}

type imageField struct {
	Key        string `json:"key"`
	Value      string `json:"value"`
	Confidence string `json:"confidence"`
}

type stage2Result struct {
	ImageFields           []imageField `json:"image_fields"`
	VisibleModelNumbers   []string     `json:"visible_model_numbers"`
}

type agentFieldsResult struct {
	Fields []product.Field `json:"fields"`
	Done   bool            `json:"done"`
}

type extractResponse struct {
	Category     string          `json:"category"`
	Condition    string          `json:"condition"`
	ShippingFree *bool           `json:"shipping_free"`
	Fields       []product.Field `json:"fields"`
}

func parseStage1JSON(text string) (*stage1Result, error) {
	jsonStr := extractJSONFromResponse(text)
	if strings.TrimSpace(jsonStr) == "" {
		return nil, fmt.Errorf("empty json in stage1 response")
	}
	var raw stage1Result
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil, fmt.Errorf("parse stage1 json: %w", err)
	}
	return &raw, nil
}

func parseStage2JSON(text string) (*stage2Result, error) {
	jsonStr := extractJSONFromResponse(text)
	if strings.TrimSpace(jsonStr) == "" {
		return nil, fmt.Errorf("empty json in stage2 response")
	}
	var raw stage2Result
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil, fmt.Errorf("parse stage2 json: %w", err)
	}
	return &raw, nil
}

func parseProductJSON(text string) (*extractResponse, error) {
	jsonStr := extractJSONFromResponse(text)
	if strings.TrimSpace(jsonStr) == "" {
		return nil, fmt.Errorf("empty json in response")
	}
	var raw extractResponse
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil, fmt.Errorf("parse product json: %w", err)
	}
	return &raw, nil
}

func parseAgentFieldsJSON(text string) (*agentFieldsResult, error) {
	jsonStr := extractJSONFromResponse(text)
	if strings.TrimSpace(jsonStr) == "" {
		return nil, fmt.Errorf("empty json in agent response")
	}
	var raw agentFieldsResult
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil, fmt.Errorf("parse agent json: %w", err)
	}
	return &raw, nil
}

func toProductDetail(raw *extractResponse) *product.ProductDetail {
	cat := product.ParseCategory(raw.Category)
	return &product.ProductDetail{
		Category:     cat,
		Condition:    raw.Condition,
		ShippingFree: raw.ShippingFree,
		Fields:       product.ValidateFields(cat, raw.Fields),
	}
}
