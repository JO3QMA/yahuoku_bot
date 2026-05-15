package gemini

import (
	"context"
	"testing"
	"time"
)

func TestNewClient_emptyAPIKey(t *testing.T) {
	_, err := NewClient("", "gemini-2.5-flash-lite")
	if err == nil {
		t.Fatal("expected error for empty API key")
	}
}

func TestClient_ExtractSpec_realAPIRejectsInvalidKey(t *testing.T) {
	if testing.Short() {
		t.Skip("network")
	}
	c, err := NewClient("invalid-api-key-for-unit-test", "gemini-2.5-flash-lite")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_, err = c.ExtractSpec(ctx, "title", "description body")
	if err == nil {
		t.Fatal("expected error from Gemini API")
	}
}
