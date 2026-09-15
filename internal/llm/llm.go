package llm

import (
	"fmt"

	"github.com/ginkcode/dict-cli/internal/config"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
)

// Options holds fully-resolved connection settings for the selected provider.
type Options struct {
	Provider string
	Model    string
	Host     string // ollama only
	BaseURL  string // openai only
	APIKey   string // openai only
}

// New builds an llms.Model for the configured provider.
func New(opts Options) (llms.Model, error) {
	switch opts.Provider {
	case config.ProviderOpenAI:
		if opts.APIKey == "" {
			return nil, fmt.Errorf("missing API key: set --api-key, OPENAI_API_KEY, or openai.api_key in config")
		}
		if opts.Model == "" {
			return nil, fmt.Errorf("missing model: set --model or openai.model in config")
		}

		oaiOpts := []openai.Option{
			openai.WithModel(opts.Model),
			openai.WithToken(opts.APIKey),
			openai.WithResponseFormat(openai.ResponseFormatJSON),
		}
		if opts.BaseURL != "" {
			oaiOpts = append(oaiOpts, openai.WithBaseURL(opts.BaseURL))
		}

		return openai.New(oaiOpts...)

	case config.ProviderOllama, "":
		return ollama.New(
			ollama.WithModel(opts.Model),
			ollama.WithServerURL(opts.Host),
			ollama.WithFormat("json"),
			ollama.WithPullModel(),
		)

	default:
		return nil, fmt.Errorf("unknown provider %q (expected %q or %q)", opts.Provider, config.ProviderOllama, config.ProviderOpenAI)
	}
}
