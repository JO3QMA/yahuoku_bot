package gemini

import (
	"context"
	"testing"
	"time"

	appauction "jo3qma.com/yahoo_auctions_bot/internal/application/auction"
)

func TestNewClient_emptyAPIKey(t *testing.T) {
	_, err := NewClient("", Options{FastModel: "gemini-2.5-flash-lite"})
	if err == nil {
		t.Fatal("expected error for empty API key")
	}
}

func TestClient_Extract_realAPIRejectsInvalidKey(t *testing.T) {
	if testing.Short() {
		t.Skip("network")
	}
	c, err := NewClient("invalid-api-key-for-unit-test", Options{FastModel: "gemini-2.5-flash-lite"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_, err = c.Extract(ctx, appauction.ExtractInput{Title: "title", Description: "description body"})
	if err == nil {
		t.Fatal("expected error from Gemini API")
	}
}
