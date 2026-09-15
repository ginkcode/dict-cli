package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/ginkcode/dict-cli/internal/config"
	"github.com/ginkcode/dict-cli/internal/gram"
	"github.com/ginkcode/dict-cli/internal/llm"
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
	provider := flag.String("provider", envOr("GRAM_PROVIDER", envOr("DICT_PROVIDER", cfg.Provider)), "LLM provider: ollama or openai")
	model := flag.String("model", "", "Model name (defaults to config for the selected provider)")
	host := flag.String("host", envOr("OLLAMA_HOST", ""), "Ollama server URL")
	baseURL := flag.String("base-url", envOr("OPENAI_BASE_URL", ""), "OpenAI-compatible API base URL")
	apiKey := flag.String("api-key", envOr("OPENAI_API_KEY", ""), "OpenAI-compatible API key")
	flag.Parse()

	if *showVersion {
		fmt.Printf("gram %s (%s) built %s\n", version, commit, date)
		return
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, `Usage: gram <sentence>

Examples:
  gram How are you today?
  gram "He go to school every day."`)
		os.Exit(1)
	}

	sentence := strings.Join(args, " ")

	opts := llm.Options{Provider: *provider}
	switch *provider {
	case config.ProviderOpenAI:
		opts.Model = firstNonEmpty(*model, envOr("GRAM_MODEL", envOr("DICT_MODEL", "")), cfg.OpenAI.Model)
		opts.BaseURL = firstNonEmpty(*baseURL, cfg.OpenAI.BaseURL)
		opts.APIKey = firstNonEmpty(*apiKey, cfg.OpenAI.APIKey)
	default:
		opts.Model = firstNonEmpty(*model, envOr("GRAM_MODEL", envOr("DICT_MODEL", "")), cfg.Ollama.Model)
		opts.Host = firstNonEmpty(*host, cfg.Ollama.Host)
	}

	llmModel, err := llm.New(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize LLM: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	result := gram.Check(ctx, llmModel, sentence)

	if result.Err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", result.Err)
		os.Exit(1)
	}

	printResult(result)
}

func printResult(r gram.Result) {
	green := "\033[32m"
	red := "\033[31m"
	bold := "\033[1m"
	reset := "\033[0m"
	if fi, err := os.Stdout.Stat(); err != nil || (fi.Mode()&os.ModeCharDevice) == 0 {
		green, red, bold, reset = "", "", "", ""
	}

	if r.IsCorrect {
		fmt.Printf("\n%s✅%s \"%s\"\n", green, reset, r.Sentence)
		fmt.Printf("%sis grammatically correct.%s\n\n", green, reset)
	} else {
		fmt.Printf("\n%s❌%s \"%s\"\n", red, reset, r.Sentence)
		fmt.Printf("%sis grammatically incorrect.%s\n\n", red, reset)
	}

	fmt.Printf("%sExplanation:%s\n%s\n", bold, reset, r.Explanation)

	if r.Correction != "" {
		fmt.Printf("\n%sCorrection:%s\n%s%s%s\n", bold, reset, green, r.Correction, reset)
		if err := clipboard.WriteAll(r.Correction); err == nil {
			fmt.Printf("(copied to clipboard)\n")
		}
	}
	fmt.Println()
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
