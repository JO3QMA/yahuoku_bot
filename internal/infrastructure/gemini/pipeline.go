package gemini

import (
	"context"
	"fmt"
	"log"
	"time"

	appauction "jo3qma.com/yahoo_auctions_bot/internal/application/auction"
	"jo3qma.com/yahoo_auctions_bot/internal/domain/product"
	"golang.org/x/sync/errgroup"
)

type pipeline struct {
	api    *genAIAPI
	opts   Options
	images *imageFetcher
}

func newPipeline(api *genAIAPI, opts Options) *pipeline {
	return &pipeline{
		api:    api,
		opts:   opts.Normalize(),
		images: newImageFetcher(),
	}
}

func (p *pipeline) run(ctx context.Context, in appauction.ExtractInput) (*product.ProductDetail, error) {
	ctx, cancel := context.WithTimeout(ctx, pipelineTimeout*time.Second)
	defer cancel()

	title := sanitizeUTF8(in.Title)
	plainDesc := plainDescription(in.Description)

	var s1 *stage1Result
	var s2 *stage2Result

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		s1, err = p.runStage1(gctx, title, plainDesc)
		return err
	})
	g.Go(func() error {
		if len(in.ImageURLs) == 0 {
			return nil
		}
		var err error
		s2, err = p.runStage2(gctx, title, plainDesc, in.ImageURLs)
		return err
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}

	var agentFields []product.Field
	var searchNotes []string
	if shouldRunStage3(s1) {
		fields, notes, err := p.runStage3(ctx, title, plainDesc, s1, s2)
		if err != nil {
			log.Printf("[gemini] stage3 failed, continuing with merge: %v", err)
		} else {
			agentFields = fields
			searchNotes = notes
		}
	}

	return p.runStage4(ctx, title, plainDesc, s1, s2, agentFields, searchNotes)
}

func shouldRunStage3(s1 *stage1Result) bool {
	if s1 == nil {
		return false
	}
	return len(s1.MissingKeys) > 0
}

func (p *pipeline) runStage1(ctx context.Context, title, plainDesc string) (*stage1Result, error) {
	text, err := p.api.generateJSON(ctx, p.opts.FastModel, buildStage1Prompt(title, plainDesc), stage1Schema())
	if err != nil {
		return nil, fmt.Errorf("stage1: %w", err)
	}
	return parseStage1JSON(text)
}

func (p *pipeline) runStage2(ctx context.Context, title, plainDesc string, imageURLs []string) (*stage2Result, error) {
	imgs := p.images.fetch(ctx, imageURLs, p.opts.MaxImages)
	if len(imgs) == 0 {
		return nil, nil
	}
	text, err := p.api.generateJSONWithImages(ctx, p.opts.VisionModel, buildStage2Prompt(title, plainDesc), imgs, stage2Schema())
	if err != nil {
		return nil, fmt.Errorf("stage2: %w", err)
	}
	return parseStage2JSON(text)
}

func (p *pipeline) runStage4(ctx context.Context, title, plainDesc string, s1 *stage1Result, s2 *stage2Result, agentFields []product.Field, searchNotes []string) (*product.ProductDetail, error) {
	text, err := p.api.generateJSON(ctx, p.opts.FastModel, buildMergePrompt(title, plainDesc, s1, s2, agentFields, searchNotes), productSchema())
	if err != nil {
		return nil, fmt.Errorf("stage4: %w", err)
	}
	raw, err := parseProductJSON(text)
	if err != nil {
		return nil, err
	}
	return toProductDetail(raw), nil
}
