package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
)

type session struct {
	api    *apiClient
	opts   Options
	images *imageFetcher
}

func newSession(api *apiClient, opts Options) *session {
	return &session{
		api:    api,
		opts:   opts.Normalize(),
		images: newImageFetcher(),
	}
}

// Extract はタイトル・説明・画像から Category 判別と Field 抽出（Extraction）を行う。
func (s *session) Extract(ctx context.Context, in product.ExtractInput) (*product.Product, error) {
	timeout := s.opts.PipelineTimeoutSec
	if timeout <= 0 {
		timeout = extractionTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	title := sanitizeUTF8(in.Title)
	plainDesc := plainDescription(in.Description)
	mirror := newProductMirror()

	var s1 *stage1Result
	var s2 *stage2Result
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		text, err := s.api.generateJSON(gctx, s.opts.FastModel, buildStage1Prompt(title, plainDesc))
		if err != nil {
			return fmt.Errorf("text extract: %w", err)
		}
		parsed, err := parseStage1JSON(text)
		if err != nil {
			return err
		}
		s1 = parsed
		return nil
	})
	g.Go(func() error {
		if len(in.ImageURLs) == 0 {
			return nil
		}
		r, err := s.runVisionSupplement(gctx, title, plainDesc, in.ImageURLs)
		if err != nil {
			log.Printf("[openai] vision supplement failed: %v", err)
			return nil
		}
		s2 = r
		return nil
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}
	mirror.applyStage1(s1)
	mirror.applyVision(s2)

	var supplementErr error
	if len(product.FilterSupplementEligibleKeys(mirror.category, mirror.unresolvedKeys())) > 0 {
		if err := s.runSearchSupplement(ctx, title, plainDesc, mirror); err != nil {
			supplementErr = err
		}
	}

	pd := mirror.toProduct()
	if supplementErr != nil {
		return pd, fmt.Errorf("search supplement: %w", supplementErr)
	}
	return pd, nil
}

func (s *session) runVisionSupplement(ctx context.Context, title, plainDesc string, imageURLs []string) (*stage2Result, error) {
	imgs := s.images.fetch(ctx, imageURLs, s.opts.MaxImages)
	if len(imgs) == 0 {
		return nil, nil
	}
	text, err := s.api.generateJSONWithImages(ctx, s.opts.VisionModel, buildStage2Prompt(title, plainDesc), imgs)
	if err != nil {
		return nil, fmt.Errorf("vision supplement: %w", err)
	}
	return parseStage2JSON(text)
}

func (s *session) runSearchSupplement(ctx context.Context, title, plainDesc string, mirror *productMirror) error {
	s1 := mirror.asStage1()
	messages := []chatMessage{{Role: "user", Content: buildStage3Prompt(title, plainDesc, s1, mirror.vision)}}
	tools := []tool{lookupSpecTool()}

	searchCount := 0
	maxTurns := s.opts.MaxSearchCalls*2 + 3

	for turn := 0; turn < maxTurns; turn++ {
		resp, err := s.api.generateWithTools(ctx, s.opts.AgentModel, messages, tools)
		if err != nil {
			return fmt.Errorf("turn %d: %w", turn, err)
		}
		if len(resp.Choices) == 0 {
			return fmt.Errorf("turn %d: no choices", turn)
		}
		ch := resp.Choices[0]

		if calls := ch.Message.ToolCalls; len(calls) > 0 {
			messages = append(messages, chatMessage{
				Role:      "assistant",
				Content:   textContent(ch.Message),
				ToolCalls: calls,
			})
			for _, call := range calls {
				if call.Function.Name != "lookup_spec" {
					continue
				}
				fr := runLookupSpec(ctx, s.api, s.opts, call, &searchCount, &mirror.searchNotes)
				frJSON, err := json.Marshal(fr)
				if err != nil {
					return fmt.Errorf("encode tool response: %w", err)
				}
				messages = append(messages, chatMessage{
					Role:       "tool",
					ToolCallID: call.ID,
					Content:    string(frJSON),
				})
			}
			continue
		}

		text := strings.TrimSpace(textContent(ch.Message))
		if text != "" {
			parsed, err := parseAgentFieldsJSON(text)
			if err == nil {
				mirror.applySupplementFields([]product.Field(parsed.Fields))
				if bool(parsed.Done) {
					if remaining := product.FilterSupplementEligibleKeys(mirror.category, mirror.unresolvedKeys()); len(remaining) > 0 {
						messages = append(messages, chatMessage{Role: "assistant", Content: text})
						messages = append(messages, chatMessage{
							Role:    "user",
							Content: "done:true ですが Supplement 対象の未解決フィールドが残っています: " + strings.Join(remaining, ", ") + "。lookup_spec で補完するか、確実な値だけ fields に入れてください。",
						})
						continue
					}
					return nil
				}
			}
		}
		break
	}

	finalPrompt := buildStage3FinalPrompt(title, plainDesc, s1, mirror.vision, mirror.searchNotes)
	text, err := s.api.generateJSON(ctx, s.opts.AgentModel, finalPrompt)
	if err != nil {
		return fmt.Errorf("final: %w", err)
	}
	parsed, err := parseAgentFieldsJSON(text)
	if err != nil {
		return err
	}
	mirror.applySupplementFields([]product.Field(parsed.Fields))
	if remaining := product.FilterSupplementEligibleKeys(mirror.category, mirror.unresolvedKeys()); len(remaining) > 0 {
		return fmt.Errorf("unresolved after search: %s", strings.Join(remaining, ", "))
	}
	return nil
}
