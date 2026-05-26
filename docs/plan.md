# dict-cli Plan

## Overview

A CLI app built with Go and LangChain, using Ollama as LLM backend to look up English words and return Vietnamese meanings, pronunciations, and usage examples.

```
dict word1 word2 word3
```

Each word is looked up concurrently via Ollama, results are displayed in tables grouped by validity.

## Project Structure

```
dict-cli/
├── main.go                  # Entrypoint: parse flags, init LLM, orchestrate
├── go.mod
├── internal/
│   ├── dict/
│   │   └── dict.go          # Meaning, Result, Lookup(), LookupAll()
│   └── output/
│       └── output.go        # renderValidTable(), renderInvalidList()
```

## Dependencies

| Package                             | Purpose                                                    |
| ----------------------------------- | ---------------------------------------------------------- |
| `github.com/tmc/langchaingo`        | LangChain Go — Ollama LLM client                           |
| `github.com/olekukonko/tablewriter` | Box-drawing tables (pure Go, no shell `column` dependency) |
| `encoding/json` (stdlib)            | Parse LLM JSON response                                    |
| `text/tabwriter` (stdlib)           | Fallback aligned columns                                   |
| `sync`, `sort` (stdlib)             | Concurrent lookups with ordered results                    |

## Flags / Environment

| Flag      | Env           | Default                  | Description       |
| --------- | ------------- | ------------------------ | ----------------- |
| `--model` | `DICT_MODEL`  | `deepseek-v4-pro:cloud`  | Ollama model name |
| `--host`  | `OLLAMA_HOST` | `http://localhost:11434` | Ollama server URL |

## Architecture Flow

```
dict word1 word2 word3
       │
       ▼
  main.go ── parse flags, init Ollama LLM (WithFormat("json"))
       │
       ▼
  dict.LookupAll(ctx, llm, words) ── goroutines with indexed results
       │
       ├──► [goroutine 0] Lookup("word1") ──► {idx:0, result}
       ├──► [goroutine 1] Lookup("word2") ──► {idx:1, result}
       └──► [goroutine 2] Lookup("word3") ──► {idx:2, result}
       │                                    (JSON mode)
       ▼
  Sort by index → partition into valid[] / invalid[]
       │
       ▼
  output.PrintResults(valid, invalid)
       ├── Green header: VALID WORDS (N)
       │   For each word: word + pronunciation → meanings table (TYPE | MEANING | EXAMPLES)
       └── Red header: INVALID WORDS (N)
           Single-column table listing invalid words
```

## Data Model

```go
type Meaning struct {
    Type     string   `json:"type"`     // noun, verb, adjective, adverb, preposition...
    Meaning  string   `json:"meaning"`  // Vietnamese definition
    Examples []string `json:"examples"` // usage examples
}

type Result struct {
    Index         int       // preserves input order
    Word          string    `json:"word"`
    IsValid       bool      `json:"is_valid"`
    Pronunciation string    `json:"pronunciation"`
    Meanings      []Meaning `json:"meanings"`
    Err           error     // network/model errors
}
```

## LLM Prompt

### System prompt

```
You are an English-Vietnamese dictionary. Check if the user's word is a valid English word.
Respond ONLY with a single JSON object, no markdown, no explanation:
{
  "is_valid": bool,
  "word": "string",
  "pronunciation": "IPA notation here",
  "meanings": [
    {
      "type": "noun/verb/adjective/adverb/preposition/etc",
      "meaning": "Vietnamese definition here",
      "examples": ["example sentence 1", "example sentence 2", "example sentence 3"]
    }
  ]
}
For invalid or non-English words, respond: {"is_valid": false, "word": "..."}
```

### User prompt

Just the word string, e.g. `"serendipity"`.

### Ollama Options

- `WithFormat("json")` — guarantees JSON output
- `WithPullModel()` — auto-pull model if not available

## Output Format

### Valid Words Section

```
✅ VALID WORDS (2)

  apple  /ˈæp.əl/
  ┌──────┬──────────────────────┬──────────────────────────────────────────┐
  │ TYPE │ MEANING              │ EXAMPLES                                 │
  ├──────┼──────────────────────┼──────────────────────────────────────────┤
  │ noun │ quả táo              │ 1. She ate a red apple.                  │
  │      │                      │ 2. Apple pie is his favorite dessert.    │
  ├──────┼──────────────────────┼──────────────────────────────────────────┤
  │ adj  │ (thuộc) về táo       │ 1. Apple orchard tours are popular.      │
  └──────┴──────────────────────┴──────────────────────────────────────────┘

  run  /rʌn/
  ┌──────┬──────────────────────┬──────────────────────────────────────────┐
  │ TYPE │ MEANING              │ EXAMPLES                                 │
  ├──────┼──────────────────────┼──────────────────────────────────────────┤
  │ verb │ chạy                 │ 1. She runs every morning.               │
  │      │                      │ 2. He ran to catch the bus.              │
  ├──────┼──────────────────────┼──────────────────────────────────────────┤
  │ noun │ cuộc đua / cuộc chạy │ 1. He went for a morning run.           │
  └──────┴──────────────────────┴──────────────────────────────────────────┘
```

### Invalid Words Section

```
❌ INVALID WORDS (2)

  ┌──────────┐
  │ WORD     │
  ├──────────┤
  │ xyzabc   │
  │ qwerty   │
  └──────────┘
```

## Color Scheme (ANSI)

| Element              | Color      | Code       |
| -------------------- | ---------- | ---------- |
| Valid words header   | Green      | `\033[32m` |
| Invalid words header | Red        | `\033[31m` |
| Word + pronunciation | Bold white | `\033[1m`  |
| Invalid word text    | Red        | `\033[31m` |
| Reset                | —          | `\033[0m`  |

## Edge Cases & Error Handling

| Scenario             | Handling                                                |
| -------------------- | ------------------------------------------------------- |
| Ollama unreachable   | Fatal: "Cannot connect to Ollama at <url>"              |
| Model not pulled     | `WithPullModel()` auto-pulls on first request           |
| LLM returns non-JSON | Parse error → retry once, then show error for that word |
| LLM timeout          | Show "timed out" for that word, continue with others    |
| Single word, invalid | Show only invalid section                               |
| All words valid      | Show only valid section (skip invalid header)           |
| No args              | Print usage: `Usage: dict <word> [word...]`             |

## Concurrency Strategy

- `LookupAll` spawns one goroutine per word
- Each goroutine sends `Result{Index: i, ...}` to a buffered channel
- Main goroutine collects `len(words)` results, sorts by `Index`
- Results always appear in the same order as input arguments
- Uses `sync.WaitGroup` + channel, no external deps needed
