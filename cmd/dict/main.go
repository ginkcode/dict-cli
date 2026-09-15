package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/ginkcode/dict-cli/internal/config"
	"github.com/ginkcode/dict-cli/internal/dict"
	"github.com/ginkcode/dict-cli/internal/llm"
	"github.com/ginkcode/dict-cli/internal/output"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	cfg, cfgErr := config.Load()
	if cfgErr != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", cfgErr)
	}

	showVersion := flag.Bool("version", false, "Print version and exit")
	provider := flag.String("provider", envOr("DICT_PROVIDER", cfg.Provider), "LLM provider: ollama or openai")
	model := flag.String("model", "", "Model name (defaults to config for the selected provider)")
	host := flag.String("host", envOr("OLLAMA_HOST", ""), "Ollama server URL")
	baseURL := flag.String("base-url", envOr("OPENAI_BASE_URL", ""), "OpenAI-compatible API base URL")
	apiKey := flag.String("api-key", envOr("OPENAI_API_KEY", ""), "OpenAI-compatible API key")
	flag.Parse()

	if *showVersion {
		fmt.Printf("dict %s (%s) built %s\n", version, commit, date)
		return
	}

	words := flag.Args()
	if len(words) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: dict <word> [word...]")
		os.Exit(1)
	}

	opts := llm.Options{Provider: *provider}
	switch *provider {
	case config.ProviderOpenAI:
		opts.Model = firstNonEmpty(*model, envOr("DICT_MODEL", ""), cfg.OpenAI.Model)
		opts.BaseURL = firstNonEmpty(*baseURL, cfg.OpenAI.BaseURL)
		opts.APIKey = firstNonEmpty(*apiKey, cfg.OpenAI.APIKey)
	default:
		opts.Model = firstNonEmpty(*model, envOr("DICT_MODEL", ""), cfg.Ollama.Model)
		opts.Host = firstNonEmpty(*host, cfg.Ollama.Host)
	}

	llmModel, err := llm.New(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize LLM: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	results := dict.LookupAll(ctx, llmModel, words)

	valid, invalid := dict.Partition(results)
	output.PrintResults(valid, invalid)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
