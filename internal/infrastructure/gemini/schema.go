package gemini

import (
	"google.golang.org/genai"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

func categoryEnumSchema() *genai.Schema {
	categoryEnums := make([]string, len(product.AllCategories))
	for i, c := range product.AllCategories {
		categoryEnums[i] = string(c)
	}
	return &genai.Schema{Type: genai.TypeString, Enum: categoryEnums}
}

func categoryLockedSchema(cat product.Category) *genai.Schema {
	return &genai.Schema{Type: genai.TypeString, Enum: []string{string(cat)}}
}

func fieldArraySchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeArray,
		Items: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"key":   {Type: genai.TypeString},
				"value": {Type: genai.TypeString},
			},
			Required: []string{"key", "value"},
		},
	}
}

func fieldArraySchemaForCategory(cat product.Category) *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeArray,
		Items: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"key":   {Type: genai.TypeString, Enum: product.TemplateKeys(cat)},
				"value": {Type: genai.TypeString},
			},
			Required: []string{"key", "value"},
		},
	}
}

func stage1Schema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"category":      categoryEnumSchema(),
			"condition":     {Type: genai.TypeString, Description: "商品の状態(新品/中古/不明)"},
			"shipping_free": {Type: genai.TypeBoolean, Description: "送料無料かどうか"},
			"fields":        fieldArraySchema(),
			"missing_keys": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeString,
				},
				Description: "テンプレート上まだ埋められていないフィールドキー",
			},
			"candidate_queries": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeString,
				},
				Description: "不足情報を補うための推定検索クエリ",
			},
		},
		Required: []string{"category", "condition", "fields", "missing_keys", "candidate_queries"},
	}
}

func stage2Schema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"image_fields": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"key":        {Type: genai.TypeString},
						"value":      {Type: genai.TypeString},
						"confidence": {Type: genai.TypeString, Description: "high/medium/low"},
					},
					Required: []string{"key", "value"},
				},
			},
			"visible_model_numbers": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeString,
				},
			},
		},
		Required: []string{"image_fields", "visible_model_numbers"},
	}
}

func productSchema(cat product.Category) *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"category":      categoryLockedSchema(cat),
			"condition":     {Type: genai.TypeString, Description: "商品の状態(新品/中古/不明)"},
			"shipping_free": {Type: genai.TypeBoolean, Description: "送料無料かどうか"},
			"fields":        fieldArraySchemaForCategory(cat),
		},
		Required: []string{"category", "condition", "fields"},
	}
}

func agentFieldsSchema(cat product.Category) *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"fields": fieldArraySchemaForCategory(cat),
			"done": {
				Type:        genai.TypeBoolean,
				Description: "補完が完了したら true",
			},
		},
		Required: []string{"fields", "done"},
	}
}
