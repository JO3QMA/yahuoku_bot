package product

import (
	"encoding/json"
	"testing"
)

func TestCategory_DisplayName(t *testing.T) {
	if CategoryServer.DisplayName() != "サーバー" {
		t.Fatal(CategoryServer.DisplayName())
	}
	if Category("invalid").DisplayName() != "その他" {
		t.Fatal()
	}
}

func TestParseCategory(t *testing.T) {
	if ParseCategory("gpu") != CategoryGPU {
		t.Fatal()
	}
	if ParseCategory("unknown") != CategoryOther {
		t.Fatal()
	}
}

func TestValidateFields_orderAndFilter(t *testing.T) {
	in := []Field{
		{Key: "other_notes", Value: "note"},
		{Key: "bogus", Value: "x"},
		{Key: "model", Value: "RTX"},
		{Key: "model", Value: "dup"},
	}
	out := ValidateFields(CategoryGPU, in)
	if len(out) != 2 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].Key != "model" || out[0].Value != "RTX" {
		t.Fatalf("first: %+v", out[0])
	}
	if out[1].Key != "other_notes" {
		t.Fatal(out[1])
	}
}

func TestProductDetail_JSONRoundTrip(t *testing.T) {
	sf := true
	p := &ProductDetail{
		Category: CategoryServer, Condition: "新品", ShippingFree: &sf,
		Fields: []Field{{Key: "cpu_model_line", Value: "Xeon"}},
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var out ProductDetail
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Category != CategoryServer || out.Fields[0].Value != "Xeon" {
		t.Fatal(out)
	}
}

func TestProductDetail_IsEffectivelyEmpty(t *testing.T) {
	if !EmptyProductDetail().IsEffectivelyEmpty() {
		t.Fatal("empty")
	}
	if (&ProductDetail{Category: CategoryGPU}).IsEffectivelyEmpty() {
		t.Fatal("has category")
	}
	if (&ProductDetail{Category: CategoryOther, Condition: "中古"}).IsEffectivelyEmpty() {
		t.Fatal("has condition")
	}
}
