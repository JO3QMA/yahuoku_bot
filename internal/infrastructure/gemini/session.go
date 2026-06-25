package gemini

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
	"golang.org/x/sync/errgroup"
	"google.golang.org/genai"
)

type session struct {
	api    *genAIAPI
	opts   Options
	images *imageFetcher
}

func newSession(api *genAIAPI, opts Options) *session {
	return &session{
		api:    api,
		opts:   opts.Normalize(),
		images: newImageFetcher(),
	}
}

func (s *session) run(ctx context.Context, in product.ExtractInput) (*product.Product, error) {
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
		text, err := s.api.generateJSON(gctx, s.opts.FastModel, buildStage1Prompt(title, plainDesc), stage1Schema())
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
			log.Printf("[gemini] vision supplement failed: %v", err)
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
	if len(mirror.unresolvedKeys()) > 0 {
		if err := s.runSearchSupplement(ctx, title, plainDesc, mirror); err != nil {
			supplementErr = err
		}
	}

	pd, err := s.finalize(ctx, title, plainDesc, mirror)
	if err != nil {
		return nil, err
	}
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
	text, err := s.api.generateJSONWithImages(ctx, s.opts.VisionModel, buildStage2Prompt(title, plainDesc), imgs, stage2Schema())
	if err != nil {
		return nil, fmt.Errorf("vision supplement: %w", err)
	}
	return parseStage2JSON(text)
}

func (s *session) runSearchSupplement(ctx context.Context, title, plainDesc string, mirror *productMirror) error {
	s1 := mirror.asStage1()
	contents := []*genai.Content{
		genai.NewContentFromText(buildStage3Prompt(title, plainDesc, s1, mirror.vision), genai.RoleUser),
	}

	searchCount := 0
	maxTurns := s.opts.MaxSearchCalls*2 + 3

	for turn := 0; turn < maxTurns; turn++ {
		resp, err := s.api.generateWithTools(ctx, s.opts.AgentModel, contents, agentToolConfig())
		if err != nil {
			return fmt.Errorf("turn %d: %w", turn, err)
		}
		if len(resp.Candidates) == 0 {
			return fmt.Errorf("turn %d: no candidates", turn)
		}
		cand := resp.Candidates[0]

		if calls := extractFunctionCalls(resp); len(calls) > 0 {
			if cand.Content != nil {
				contents = append(contents, cand.Content)
			}
			var notes []string
			for _, call := range calls {
				if call.Name != "lookup_spec" {
					continue
				}
				fr := runLookupSpec(ctx, s.api, s.opts, call, &searchCount, &notes)
				contents = append(contents, genai.NewContentFromFunctionResponse(call.Name, fr, genai.RoleUser))
			}
			for _, n := range notes {
				mirror.appendSearchNote(n)
			}
			continue
		}

		text := strings.TrimSpace(resp.Text())
		if text != "" {
			parsed, err := parseAgentFieldsJSON(text)
			if err == nil {
				mirror.applyFields(parsed.Fields)
				if parsed.Done {
					if remaining := mirror.unresolvedKeys(); len(remaining) > 0 {
						contents = append(contents, cand.Content)
						contents = append(contents, genai.NewContentFromText(
							"done:true ですが未解決フィールドが残っています: "+strings.Join(remaining, ", ")+
								"。lookup_spec で補完するか、確実な値だけ fields に入れてください。",
							genai.RoleUser,
						))
						continue
					}
					return nil
				}
			}
		}
		break
	}

	finalPrompt := buildStage3FinalPrompt(title, plainDesc, s1, mirror.vision, mirror.searchNotes)
	text, err := s.api.generateJSON(ctx, s.opts.AgentModel, finalPrompt, agentFieldsSchema())
	if err != nil {
		return fmt.Errorf("final: %w", err)
	}
	parsed, err := parseAgentFieldsJSON(text)
	if err != nil {
		return err
	}
	mirror.applyFields(parsed.Fields)
	if remaining := mirror.unresolvedKeys(); len(remaining) > 0 {
		return fmt.Errorf("unresolved after search: %s", strings.Join(remaining, ", "))
	}
	return nil
}

func (s *session) finalize(ctx context.Context, title, plainDesc string, mirror *productMirror) (*product.Product, error) {
	s1 := mirror.asStage1()
	text, err := s.api.generateJSON(ctx, s.opts.FastModel, buildMergePrompt(title, plainDesc, s1, mirror.vision, nil, mirror.searchNotes), productSchema())
	if err != nil {
		return nil, fmt.Errorf("finalize: %w", err)
	}
	raw, err := parseProductJSON(text)
	if err != nil {
		return nil, err
	}
	return toProduct(raw), nil
}
