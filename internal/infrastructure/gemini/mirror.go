package gemini

import (
	"strings"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

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
	m.category = product.ParseCategory(s1.Category)
	m.condition = s1.Condition
	m.freeShipping = s1.FreeShipping
	m.pendingMissing = filterTemplateKeys(m.category, s1.MissingKeys)
	for _, f := range s1.Fields {
		if strings.TrimSpace(f.Value) != "" {
			m.fields[f.Key] = f.Value
		}
	}
}

func (m *productMirror) applyVision(s2 *stage2Result) {
	if s2 == nil {
		return
	}
	m.vision = s2
	for _, f := range s2.ImageFields {
		if strings.TrimSpace(f.Value) == "" {
			continue
		}
		if _, ok := m.fields[f.Key]; !ok {
			m.fields[f.Key] = f.Value
		}
	}
}

func (m *productMirror) applyFields(fields []product.Field) {
	for _, f := range fields {
		if strings.TrimSpace(f.Value) != "" {
			m.fields[f.Key] = f.Value
		}
	}
}

func (m *productMirror) appendSearchNote(note string) {
	if strings.TrimSpace(note) != "" {
		m.searchNotes = append(m.searchNotes, note)
	}
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
	allowed := make(map[string]struct{}, len(product.TemplatesFor(cat)))
	for _, t := range product.TemplatesFor(cat) {
		allowed[t.Key] = struct{}{}
	}
	var out []string
	for _, k := range keys {
		if _, ok := allowed[k]; ok {
			out = append(out, k)
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
		Category:     string(m.category),
		Condition:    m.condition,
		FreeShipping: m.freeShipping,
		Fields:       m.fieldsSlice(),
		MissingKeys:  m.unresolvedKeys(),
	}
}
