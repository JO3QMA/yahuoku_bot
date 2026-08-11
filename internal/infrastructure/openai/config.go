package openai

const (
	defaultBaseURL      = "https://api.openai.com/v1"
	defaultFastModel    = "gpt-4o-mini"
	defaultVisionModel  = "gpt-4o"
	defaultAgentModel   = "gpt-4o"
	defaultMaxImages    = 3
	defaultMaxSearch    = 3
	extractionTimeout   = 45 // seconds
)

// Options は Extraction の設定。
type Options struct {
	BaseURL            string
	FastModel          string
	VisionModel        string
	AgentModel         string
	MaxImages          int
	MaxSearchCalls     int
	PipelineTimeoutSec int
}

// NewOptions はベース URL・モデル名と数値設定から Options を構築する。
func NewOptions(baseURL, fastModel, visionModel, agentModel string, maxImages, maxSearchCalls, pipelineTimeoutSec int) Options {
	return Options{
		BaseURL:            baseURL,
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
	if o.BaseURL == "" {
		o.BaseURL = defaultBaseURL
	}
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
		o.PipelineTimeoutSec = extractionTimeout
	}
	return o
}
