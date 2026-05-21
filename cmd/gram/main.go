package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/gink/dict-cli/internal/gram"
	"github.com/tmc/langchaingo/llms/ollama"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	showVersion := flag.Bool("version", false, "Print version and exit")
	model := flag.String("model", envOr("GRAM_MODEL", envOr("DICT_MODEL", "deepseek-v4-pro:cloud")), "Ollama model name")
	host := flag.String("host", envOr("OLLAMA_HOST", "http://localhost:11434"), "Ollama server URL")
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
	result := gram.Check(ctx, llm, sentence)

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
