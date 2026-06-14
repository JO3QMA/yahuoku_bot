package gemini

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const maxImageBytes = 4 << 20 // 4MB

type fetchedImage struct {
	MIMEType string
	Data     []byte
}

type imageFetcher struct {
	client *http.Client
}

func newImageFetcher() *imageFetcher {
	return &imageFetcher{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (f *imageFetcher) fetch(ctx context.Context, urls []string, maxImages int) []fetchedImage {
	if maxImages <= 0 || len(urls) == 0 {
		return nil
	}
	limit := maxImages
	if len(urls) < limit {
		limit = len(urls)
	}
	out := make([]fetchedImage, 0, limit)
	for _, u := range urls[:limit] {
		img, err := f.fetchOne(ctx, u)
		if err != nil {
			continue
		}
		out = append(out, img)
	}
	return out
}

func (f *imageFetcher) fetchOne(ctx context.Context, rawURL string) (fetchedImage, error) {
	u := strings.TrimSpace(rawURL)
	if u == "" {
		return fetchedImage{}, fmt.Errorf("empty url")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return fetchedImage{}, err
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return fetchedImage{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fetchedImage{}, fmt.Errorf("status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes+1))
	if err != nil {
		return fetchedImage{}, err
	}
	if len(body) > maxImageBytes {
		return fetchedImage{}, fmt.Errorf("image too large")
	}
	mime := resp.Header.Get("Content-Type")
	if i := strings.Index(mime, ";"); i >= 0 {
		mime = strings.TrimSpace(mime[:i])
	}
	if mime == "" {
		mime = guessImageMIME(u)
	}
	if !strings.HasPrefix(mime, "image/") {
		return fetchedImage{}, fmt.Errorf("not an image: %s", mime)
	}
	return fetchedImage{MIMEType: mime, Data: body}, nil
}

func guessImageMIME(url string) string {
	lower := strings.ToLower(url)
	switch {
	case strings.Contains(lower, ".png"):
		return "image/png"
	case strings.Contains(lower, ".webp"):
		return "image/webp"
	case strings.Contains(lower, ".gif"):
		return "image/gif"
	default:
		return "image/jpeg"
	}
}
