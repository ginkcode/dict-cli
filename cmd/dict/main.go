package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/gink/dict-cli/internal/dict"
	"github.com/gink/dict-cli/internal/output"
	"github.com/tmc/langchaingo/llms/ollama"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "Print version and exit")
	model := flag.String("model", envOr("DICT_MODEL", "glm-5.1:cloud"), "Ollama model name")
	host := flag.String("host", envOr("OLLAMA_HOST", "http://localhost:11434"), "Ollama server URL")
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

	llm, err := ollama.New(
		ollama.WithModel(*model),
		ollama.WithServerURL(*host),
		ollama.WithFormat("json"),
		ollama.WithPullModel(),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to initialize Ollama: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	results := dict.LookupAll(ctx, llm, words)

	valid, invalid := dict.Partition(results)
	output.PrintResults(valid, invalid)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
