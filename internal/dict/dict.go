package dict

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/tmc/langchaingo/llms"
)

type Meaning struct {
	Type     string   `json:"type"`
	Grammar  string   `json:"grammar"`
	Meaning  string   `json:"meaning"`
	Examples []string `json:"examples"`
}

type LLMResponse struct {
	IsValid       bool      `json:"is_valid"`
	Word          string    `json:"word"`
	Pronunciation string    `json:"pronunciation"`
	Meanings      []Meaning `json:"meanings"`
}

type Result struct {
	Index         int
	Word          string
	IsValid       bool
	Pronunciation string
	Meanings      []Meaning
	Err           error
}

const SystemPrompt = `You are an English-Vietnamese dictionary. Check if the user's input is a valid English word, phrase, or idiom.
Respond ONLY with a single JSON object, no markdown, no explanation:
{
  "is_valid": bool,
  "word": "string",
  "pronunciation": "IPA notation here (leave empty for phrases or idioms if unsure)",
  "meanings": [
    {
      "type": "noun/verb/adjective/adverb/preposition/phrasal verb/idiom/phrase/etc",
      "grammar": "for nouns: countable/uncountable/both; for verbs: transitive/intransitive/both; for adjectives: attributive/predicative/both; otherwise leave empty",
      "meaning": "Vietnamese definition here",
      "examples": ["example sentence 1", "example sentence 2", "example sentence 3"]
    }
  ]
}
For invalid or non-English words, respond: {"is_valid": false, "word": "..."}`

func Lookup(ctx context.Context, llm llms.Model, word string) Result {
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, SystemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, word),
	}

	resp, err := llm.GenerateContent(ctx, messages, llms.WithTemperature(0))
	if err != nil {
		return Result{Word: word, Err: fmt.Errorf("llm error: %w", err)}
	}

	if len(resp.Choices) == 0 {
		return Result{Word: word, Err: fmt.Errorf("empty response from model")}
	}

	content := resp.Choices[0].Content

	var llmResp LLMResponse
	if err := json.Unmarshal([]byte(content), &llmResp); err != nil {
		content = extractJSON(content)
		if err := json.Unmarshal([]byte(content), &llmResp); err != nil {
			return Result{Word: word, Err: fmt.Errorf("failed to parse response")}
		}
	}

	if llmResp.Word == "" {
		llmResp.Word = word
	}

	return Result{
		Word:          llmResp.Word,
		IsValid:       llmResp.IsValid,
		Pronunciation: llmResp.Pronunciation,
		Meanings:      llmResp.Meanings,
	}
}

func extractJSON(s string) string {
	depth := 0
	var start, end int

	for i, c := range s {
		if c == '{' {
			if depth == 0 {
				start = i
			}
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 {
				end = i + 1
				break
			}
		}
	}

	if start >= 0 && end > start {
		return s[start:end]
	}
	return s
}

func LookupAll(ctx context.Context, llm llms.Model, words []string) []Result {
	results := make([]Result, len(words))
	var wg sync.WaitGroup

	for i, word := range words {
		wg.Add(1)
		go func(idx int, w string) {
			defer wg.Done()
			r := Lookup(ctx, llm, w)
			r.Index = idx
			results[idx] = r
		}(i, word)
	}

	wg.Wait()
	return results
}

func Partition(results []Result) (valid, invalid []Result) {
	for _, r := range results {
		if r.Err != nil || !r.IsValid {
			invalid = append(invalid, r)
		} else {
			valid = append(valid, r)
		}
	}
	return
}
