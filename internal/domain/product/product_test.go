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

func TestValidateFields_serverTemplateOrder(t *testing.T) {
	in := []Field{
		{Key: "gpu", Value: "AMD Radeon RX9070XT x2"},
		{Key: "server_model", Value: "Fujitsu Primergy RX1330 M4"},
		{Key: "storage_type", Value: "SSD"},
		{Key: "cpu_model_line", Value: "Intel Core Ultra 7 355 @4.25GHz x1"},
		{Key: "storage_info", Value: "SSD 256GB x1"},
	}
	out := ValidateFields(CategoryServer, in)
	if len(out) != 4 {
		t.Fatalf("len=%d, got %+v", len(out), out)
	}
	want := []string{"server_model", "cpu_model_line", "storage_info", "gpu"}
	for i, key := range want {
		if out[i].Key != key {
			t.Fatalf("out[%d].Key=%q, want %q", i, out[i].Key, key)
		}
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

func TestValidateFields_serverAliases(t *testing.T) {
	in := []Field{
		{Key: "model", Value: "DELL Precision 7920 Tower"},
		{Key: "cpu", Value: "Xeon Gold 6242 2.8GHz x2"},
		{Key: "memory", Value: "128GB"},
		{Key: "storage", Value: "1TB SSD"},
		{Key: "drive", Value: "DVD+-RW"},
		{Key: "os", Value: "Win11 Pro"},
		{Key: "power_supply", Value: "1400W"},
	}
	out := ValidateFields(CategoryServer, in)
	if len(out) != 5 {
		t.Fatalf("len=%d, got %+v", len(out), out)
	}
	if out[0].Key != "server_model" || out[0].Value != "DELL Precision 7920 Tower" {
		t.Fatalf("server_model: %+v", out[0])
	}
	if out[3].Key != "storage_info" || out[3].Value != "1TB SSD" {
		t.Fatalf("storage_info: %+v", out[3])
	}
	if out[4].Key != "other_notes" || out[4].Value != "Win11 Pro" {
		t.Fatalf("other_notes: %+v", out[4])
	}
}

func TestProduct_JSONRoundTrip(t *testing.T) {
	sf := true
	p := &Product{
		Category: CategoryServer, Condition: "新品", FreeShipping: &sf,
		Fields: []Field{{Key: "cpu_model_line", Value: "Xeon"}},
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var out Product
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Category != CategoryServer || out.Fields[0].Value != "Xeon" {
		t.Fatal(out)
	}
}

func TestProduct_IsEffectivelyEmpty(t *testing.T) {
	if !EmptyProduct().IsEffectivelyEmpty() {
		t.Fatal("empty")
	}
	if (&Product{Category: CategoryGPU}).IsEffectivelyEmpty() {
		t.Fatal("has category")
	}
	if (&Product{Category: CategoryOther, Condition: "中古"}).IsEffectivelyEmpty() {
		t.Fatal("has condition")
	}
}
