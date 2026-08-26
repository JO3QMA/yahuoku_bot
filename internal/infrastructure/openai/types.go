package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

// fieldList は LLM が fields を配列でも {key:value} オブジェクトでも返せるようにする。
type fieldList []product.Field

func (fl *fieldList) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		*fl = nil
		return nil
	}
	switch data[0] {
	case '[':
		var arr []product.Field
		if err := json.Unmarshal(data, &arr); err != nil {
			return err
		}
		*fl = arr
		return nil
	case '{':
		var obj map[string]string
		if err := json.Unmarshal(data, &obj); err != nil {
			return fmt.Errorf("fields object: %w", err)
		}
		keys := make([]string, 0, len(obj))
		for k := range obj {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make([]product.Field, 0, len(keys))
		for _, k := range keys {
			out = append(out, product.Field{Key: k, Value: obj[k]})
		}
		*fl = out
		return nil
	default:
		return fmt.Errorf("fields: expected array or object")
	}
}

type stage1Result struct {
	Category         string    `json:"category"`
	Condition        string    `json:"condition"`
	FreeShipping     *bool     `json:"shipping_free"`
	Fields           fieldList `json:"fields"`
	MissingKeys      []string  `json:"missing_keys"`
	CandidateQueries []string  `json:"candidate_queries"`
}

type imageField struct {
	Key        string `json:"key"`
	Value      string `json:"value"`
	Confidence string `json:"confidence"`
}

type stage2Result struct {
	ImageFields         []imageField `json:"image_fields"`
	VisibleModelNumbers []string     `json:"visible_model_numbers"`
}

type agentFieldsResult struct {
	Fields fieldList `json:"fields"`
	Done   bool      `json:"done"`
}

type extractResponse struct {
	Category     string    `json:"category"`
	Condition    string    `json:"condition"`
	FreeShipping *bool     `json:"shipping_free"`
	Fields       fieldList `json:"fields"`
}

func parseStage1JSON(text string) (*stage1Result, error) {
	return parseJSON[stage1Result](text, "stage1")
}

func parseStage2JSON(text string) (*stage2Result, error) {
	return parseJSON[stage2Result](text, "stage2")
}

func parseProductJSON(text string) (*extractResponse, error) {
	return parseJSON[extractResponse](text, "product")
}

func parseAgentFieldsJSON(text string) (*agentFieldsResult, error) {
	return parseJSON[agentFieldsResult](text, "agent")
}

func parseJSON[T any](text, stage string) (*T, error) {
	jsonStr := extractJSONFromResponse(text)
	if strings.TrimSpace(jsonStr) == "" {
		return nil, fmt.Errorf("empty json in %s response", stage)
	}
	var raw T
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil, fmt.Errorf("parse %s json: %w", stage, err)
	}
	return &raw, nil
}

func toProduct(raw *extractResponse) *product.Product {
	cat := product.ParseCategory(raw.Category)
	return &product.Product{
		Category:     cat,
		Condition:    raw.Condition,
		FreeShipping: raw.FreeShipping,
		Fields:       product.ValidateFields(cat, []product.Field(raw.Fields)),
	}
}
