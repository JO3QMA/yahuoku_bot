package openai

import (
	"log"
	"strings"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

func warnUnknownFieldKey(cat product.Category, key string) {
	log.Printf("[product] unknown field key %q for category %s", key, cat)
}

// productMirror は Extraction 中の Product ドラフト。構造と UnresolvedField は Go が保証する。
type productMirror struct {
	category       product.Category
	condition      string
	freeShipping   *bool
	fields         map[string]string
	pendingMissing []string // Stage1 が報告した UnresolvedField（テンプレートキーに限定）
	vision         *stage2Result
	searchNotes    []string
}

func newProductMirror() *productMirror {
	return &productMirror{fields: make(map[string]string)}
}

func (m *productMirror) applyStage1(s1 *stage1Result) {
	if s1 == nil {
		return
	}
	m.category = product.ParseCategory(string(s1.Category))
	m.condition = string(s1.Condition)
	m.freeShipping = s1.FreeShipping.ptr()
	m.pendingMissing = filterTemplateKeys(m.category, s1.MissingKeys)
	for _, f := range s1.Fields {
		m.setField(f.Key, f.Value)
	}
}

func (m *productMirror) applyVision(s2 *stage2Result) {
	if s2 == nil {
		return
	}
	m.vision = s2
	for _, f := range s2.ImageFields {
		key := string(f.Key)
		canon := product.CanonicalFieldKey(m.category, key)
		value := strings.TrimSpace(string(f.Value))
		if canon == "" {
			if value != "" {
				warnUnknownFieldKey(m.category, key)
			}
			continue
		}
		if value == "" || m.fields[canon] != "" {
			continue
		}
		m.fields[canon] = value
	}
}

func (m *productMirror) applySupplementFields(fields []product.Field) {
	for _, f := range fields {
		canon := product.CanonicalFieldKey(m.category, f.Key)
		if canon == "" {
			if strings.TrimSpace(f.Value) != "" {
				warnUnknownFieldKey(m.category, f.Key)
			}
			continue
		}
		if !product.IsSupplementEligibleKey(m.category, canon) {
			continue
		}
		if strings.TrimSpace(m.fields[canon]) != "" {
			continue
		}
		m.setField(canon, f.Value)
	}
}

func (m *productMirror) setField(key, value string) {
	if m.category == "" {
		return
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	canon := product.CanonicalFieldKey(m.category, key)
	if canon == "" {
		warnUnknownFieldKey(m.category, key)
		return
	}
	m.fields[canon] = value
}

// unresolvedKeys は pendingMissing のうち、まだ値が入っていないキー一覧を返す。
func (m *productMirror) unresolvedKeys() []string {
	if len(m.pendingMissing) == 0 {
		return nil
	}
	var out []string
	for _, k := range m.pendingMissing {
		if strings.TrimSpace(m.fields[k]) == "" {
			out = append(out, k)
		}
	}
	return out
}

func filterTemplateKeys(cat product.Category, keys []string) []string {
	if cat == "" || len(keys) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	var out []string
	for _, k := range keys {
		canon := product.CanonicalFieldKey(cat, k)
		if canon == "" {
			warnUnknownFieldKey(cat, k)
			continue
		}
		if _, ok := seen[canon]; !ok {
			seen[canon] = struct{}{}
			out = append(out, canon)
		}
	}
	return out
}

func (m *productMirror) fieldsSlice() []product.Field {
	if m.category == "" {
		return nil
	}
	defs := product.TemplatesFor(m.category)
	out := make([]product.Field, 0, len(defs))
	for _, d := range defs {
		if v := m.fields[d.Key]; strings.TrimSpace(v) != "" {
			out = append(out, product.Field{Key: d.Key, Value: v})
		}
	}
	return out
}

func (m *productMirror) asStage1() *stage1Result {
	if m.category == "" {
		return nil
	}
	return &stage1Result{
		Category:     flexString(m.category),
		Condition:    flexString(m.condition),
		FreeShipping: nullableBoolFromPtr(m.freeShipping),
		Fields:       m.fieldsSlice(),
		MissingKeys:  m.unresolvedKeys(),
	}
}

func (m *productMirror) toProduct() *product.Product {
	return &product.Product{
		Category:     m.category,
		Condition:    m.condition,
		FreeShipping: m.freeShipping,
		Fields:       product.ValidateFields(m.category, m.fieldsSlice()),
	}
}
