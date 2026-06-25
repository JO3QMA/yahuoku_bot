package gemini

import (
	"testing"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

func Test_remainingMissingKeys_stage2FillsKey(t *testing.T) {
	s1 := &stage1Result{
		MissingKeys: []string{"cpu_model_line", "memory"},
		Fields:      []product.Field{{Key: "memory", Value: "32GB"}},
	}
	s2 := &stage2Result{
		ImageFields: []imageField{{Key: "cpu_model_line", Value: "Xeon E5", Confidence: "high"}},
	}
	if keys := remainingMissingKeys(s1, s2); len(keys) != 0 {
		t.Fatalf("expected no remaining keys, got %v", keys)
	}
}

func Test_remainingMissingKeys_stage1FieldFillsKey(t *testing.T) {
	s1 := &stage1Result{
		MissingKeys: []string{"model"},
		Fields:      []product.Field{{Key: "model", Value: "RTX 3080"}},
	}
	if keys := remainingMissingKeys(s1, nil); len(keys) != 0 {
		t.Fatalf("got %v", keys)
	}
}
