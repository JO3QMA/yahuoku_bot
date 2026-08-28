package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

// flexString は LLM が文字列フィールドをオブジェクトで返すことがある（例: condition, category）。
type flexString string

func (s *flexString) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		*s = ""
		return nil
	}
	switch data[0] {
	case '"':
		var str string
		if err := json.Unmarshal(data, &str); err != nil {
			return err
		}
		*s = flexString(str)
		return nil
	case '{':
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(data, &obj); err != nil {
			return fmt.Errorf("flexString object: %w", err)
		}
		for _, key := range []string{"value", "label", "text", "name", "condition", "category", "key"} {
			if raw, ok := obj[key]; ok {
				var str string
				if err := json.Unmarshal(raw, &str); err == nil && strings.TrimSpace(str) != "" {
					*s = flexString(str)
					return nil
				}
			}
		}
		return fmt.Errorf("flexString object: no string value")
	default:
		return fmt.Errorf("flexString: expected string or object")
	}
}

// flexStringSlice は LLM が文字列配列を単一文字列やオブジェクト配列で返すことがある。
type flexStringSlice []string

func (sl *flexStringSlice) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		*sl = nil
		return nil
	}
	switch data[0] {
	case '"':
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		if s == "" {
			*sl = nil
			return nil
		}
		*sl = []string{s}
		return nil
	case '[':
		var raw []json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			var fs flexString
			if err := json.Unmarshal(item, &fs); err != nil {
				return err
			}
			if s := strings.TrimSpace(string(fs)); s != "" {
				out = append(out, s)
			}
		}
		*sl = out
		return nil
	default:
		return fmt.Errorf("flexStringSlice: expected string or array")
	}
}

// flexBool は LLM が bool を文字列やオブジェクトで返すことがある。
type flexBool bool

func (b *flexBool) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		*b = false
		return nil
	}
	switch data[0] {
	case 't', 'f':
		var v bool
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		*b = flexBool(v)
		return nil
	case '"':
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "true", "1", "yes":
			*b = true
		case "false", "0", "no", "":
			*b = false
		default:
			return fmt.Errorf("flexBool string: %q", s)
		}
		return nil
	case '{':
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(data, &obj); err != nil {
			return fmt.Errorf("flexBool object: %w", err)
		}
		for _, key := range []string{"value", "shipping_free", "done", "free", "free_shipping"} {
			if raw, ok := obj[key]; ok {
				var nested flexBool
				if err := json.Unmarshal(raw, &nested); err == nil {
					*b = nested
					return nil
				}
			}
		}
		return fmt.Errorf("flexBool object: no bool value")
	default:
		return fmt.Errorf("flexBool: unexpected json")
	}
}

// nullableFlexBool は null と false を区別する *bool 用。
type nullableFlexBool struct {
	set   bool
	value bool
}

func (n *nullableFlexBool) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		n.set = false
		return nil
	}
	var b flexBool
	if err := b.UnmarshalJSON(data); err != nil {
		return err
	}
	n.set = true
	n.value = bool(b)
	return nil
}

func (n nullableFlexBool) ptr() *bool {
	if !n.set {
		return nil
	}
	v := n.value
	return &v
}

func nullableBoolFromPtr(p *bool) nullableFlexBool {
	if p == nil {
		return nullableFlexBool{}
	}
	return nullableFlexBool{set: true, value: *p}
}

type flexField struct {
	Key   flexString `json:"key"`
	Value flexString `json:"value"`
}

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
		var arr []flexField
		if err := json.Unmarshal(data, &arr); err != nil {
			return err
		}
		out := make([]product.Field, 0, len(arr))
		for _, f := range arr {
			out = append(out, product.Field{Key: string(f.Key), Value: string(f.Value)})
		}
		*fl = out
		return nil
	case '{':
		var obj map[string]json.RawMessage
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
			var val flexString
			if err := json.Unmarshal(obj[k], &val); err != nil {
				return fmt.Errorf("fields object value: %w", err)
			}
			out = append(out, product.Field{Key: k, Value: string(val)})
		}
		*fl = out
		return nil
	default:
		return fmt.Errorf("fields: expected array or object")
	}
}

type stage1Result struct {
	Category         flexString      `json:"category"`
	Condition        flexString      `json:"condition"`
	FreeShipping     nullableFlexBool `json:"shipping_free"`
	Fields           fieldList       `json:"fields"`
	MissingKeys flexStringSlice `json:"missing_keys"`
}

type imageField struct {
	Key        flexString `json:"key"`
	Value      flexString `json:"value"`
	Confidence flexString `json:"confidence"`
}

type stage2Result struct {
	ImageFields         []imageField    `json:"image_fields"`
	VisibleModelNumbers flexStringSlice `json:"visible_model_numbers"`
}

type agentFieldsResult struct {
	Fields fieldList `json:"fields"`
	Done   flexBool  `json:"done"`
}

func parseStage1JSON(text string) (*stage1Result, error) {
	return parseJSON[stage1Result](text, "stage1")
}

func parseStage2JSON(text string) (*stage2Result, error) {
	return parseJSON[stage2Result](text, "stage2")
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
