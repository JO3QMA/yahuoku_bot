package market

import (
	"testing"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

func TestIdentityValue_gpu(t *testing.T) {
	p := &product.Product{
		Category: product.CategoryGPU,
		Fields:   []product.Field{{Key: "model", Value: "GTX 1080"}},
	}
	k, v, ok := IdentityValue(p)
	if !ok || k != "model" || v != "GTX 1080" {
		t.Fatalf("got %q %q %v", k, v, ok)
	}
}

func TestIdentityValue_missing(t *testing.T) {
	p := &product.Product{Category: product.CategoryGPU, Fields: nil}
	_, _, ok := IdentityValue(p)
	if ok {
		t.Fatal("expected false")
	}
}
