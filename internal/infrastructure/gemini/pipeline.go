package gemini

import (
	"context"
	"fmt"
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

// run は Stage1/2 を並列実行し、不足キーがあれば Stage3 で補完して Stage4 で統合する。
func (p *pipeline) run(ctx context.Context, in appauction.ExtractInput) (*product.Product, error) {
	timeout := p.opts.PipelineTimeoutSec
	if timeout <= 0 {
		timeout = pipelineTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
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
	var stage3Err error
	if shouldRunStage3(s1, s2) {
		agentFields, searchNotes, stage3Err = p.runStage3(ctx, title, plainDesc, s1, s2)
	}

	detail, err := p.runStage4(ctx, title, plainDesc, s1, s2, agentFields, searchNotes)
	if err != nil {
		return nil, err
	}
	if stage3Err != nil {
		return detail, fmt.Errorf("stage3: %w", stage3Err)
	}
	return detail, nil
}

func shouldRunStage3(s1 *stage1Result, s2 *stage2Result) bool {
	return len(remainingMissingKeys(s1, s2)) > 0
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

func (p *pipeline) runStage4(ctx context.Context, title, plainDesc string, s1 *stage1Result, s2 *stage2Result, agentFields []product.Field, searchNotes []string) (*product.Product, error) {
	text, err := p.api.generateJSON(ctx, p.opts.FastModel, buildMergePrompt(title, plainDesc, s1, s2, agentFields, searchNotes), productSchema())
	if err != nil {
		return nil, fmt.Errorf("stage4: %w", err)
	}
	raw, err := parseProductJSON(text)
	if err != nil {
		return nil, err
	}
	return toProduct(raw), nil
}
