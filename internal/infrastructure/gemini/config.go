package gemini

import "strings"

// ExtractionMode は Extraction の実行方式。
type ExtractionMode string

const (
	ExtractionModePipeline ExtractionMode = "pipeline"
	ExtractionModeSession  ExtractionMode = "session"
)

// ParseExtractionMode は環境変数値を ExtractionMode に変換する。未対応値は pipeline。
func ParseExtractionMode(s string) ExtractionMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "session":
		return ExtractionModeSession
	default:
		return ExtractionModePipeline
	}
}

const (
	defaultFastModel   = "gemini-2.5-flash-lite"
	defaultVisionModel = "gemini-2.5-flash"
	defaultAgentModel  = "gemini-2.5-flash"
	defaultMaxImages   = 3
	defaultMaxSearch   = 3
	pipelineTimeout    = 45 // seconds
)

// Options は Extraction（pipeline / session）の設定。
type Options struct {
	FastModel          string
	VisionModel        string
	AgentModel         string
	MaxImages          int
	MaxSearchCalls     int
	PipelineTimeoutSec int
	ExtractionMode     ExtractionMode
}

// NewOptions はモデル名と数値設定から Options を構築する。
func NewOptions(fastModel, visionModel, agentModel string, maxImages, maxSearchCalls, pipelineTimeoutSec int) Options {
	return Options{
		FastModel:          fastModel,
		VisionModel:        visionModel,
		AgentModel:         agentModel,
		MaxImages:          maxImages,
		MaxSearchCalls:     maxSearchCalls,
		PipelineTimeoutSec: pipelineTimeoutSec,
	}.Normalize()
}

// Normalize は空の設定を既定値で埋める。
func (o Options) Normalize() Options {
	if o.FastModel == "" {
		o.FastModel = defaultFastModel
	}
	if o.VisionModel == "" {
		o.VisionModel = defaultVisionModel
	}
	if o.AgentModel == "" {
		o.AgentModel = defaultAgentModel
	}
	if o.MaxImages <= 0 {
		o.MaxImages = defaultMaxImages
	}
	if o.MaxSearchCalls <= 0 {
		o.MaxSearchCalls = defaultMaxSearch
	}
	if o.PipelineTimeoutSec <= 0 {
		o.PipelineTimeoutSec = pipelineTimeout
	}
	if o.ExtractionMode == "" {
		o.ExtractionMode = ExtractionModePipeline
	}
	return o
}
