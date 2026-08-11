package openai

import (
	"testing"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

func Test_productMirror_unresolvedKeys(t *testing.T) {
	m := newProductMirror()
	m.applyStage1(&stage1Result{
		Category:    "gpu",
		Fields:      []product.Field{{Key: "model", Value: "RTX 3080"}},
		MissingKeys: []string{},
	})
	if keys := m.unresolvedKeys(); len(keys) != 0 {
		t.Fatalf("expected no unresolved, got %v", keys)
	}

	m2 := newProductMirror()
	m2.applyStage1(&stage1Result{
		Category:    "gpu",
		Fields:      nil,
		MissingKeys: []string{"model"},
	})
	if keys := m2.unresolvedKeys(); len(keys) != 1 || keys[0] != "model" {
		t.Fatalf("got %v", keys)
	}
}

func Test_productMirror_applyVision_fillsUnresolved(t *testing.T) {
	m := newProductMirror()
	m.applyStage1(&stage1Result{
		Category:    "server",
		MissingKeys: []string{"cpu_model_line"},
	})
	m.applyVision(&stage2Result{
		ImageFields: []imageField{{Key: "cpu_model_line", Value: "Xeon E5", Confidence: "high"}},
	})
	if keys := m.unresolvedKeys(); len(keys) != 0 {
		t.Fatalf("vision should fill cpu_model_line, unresolved=%v", keys)
	}
}

func Test_productMirror_applySupplementFields_rejectsInstalledConfiguration(t *testing.T) {
	m := newProductMirror()
	m.applyStage1(&stage1Result{Category: "server"})
	m.applySupplementFields([]product.Field{
		{Key: "server_model", Value: "PowerEdge R740"},
		{Key: "cpu_model_line", Value: "Xeon Gold 6248"},
	})
	if m.fields["server_model"] != "PowerEdge R740" {
		t.Fatal("server_model should be applied")
	}
	if m.fields["cpu_model_line"] != "" {
		t.Fatalf("cpu_model_line should be rejected, got %q", m.fields["cpu_model_line"])
	}
}

func Test_productMirror_applySupplementFields_doesNotOverwrite(t *testing.T) {
	m := newProductMirror()
	m.applyStage1(&stage1Result{
		Category: "server",
		Fields:   []product.Field{{Key: "server_model", Value: "Dell R740"}},
	})
	m.applySupplementFields([]product.Field{{Key: "server_model", Value: "PowerEdge R740"}})
	if m.fields["server_model"] != "Dell R740" {
		t.Fatalf("listing evidence should win, got %q", m.fields["server_model"])
	}
}

func Test_filterTemplateKeys_rejectsUnknown(t *testing.T) {
	got := filterTemplateKeys(product.CategoryGPU, []string{"model", "not_a_key"})
	if len(got) != 1 || got[0] != "model" {
		t.Fatalf("got %v", got)
	}
}
