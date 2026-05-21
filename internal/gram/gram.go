package gram

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tmc/langchaingo/llms"
)

type LLMResponse struct {
	Sentence    string `json:"sentence"`
	IsCorrect   bool   `json:"is_correct"`
	Explanation string `json:"explanation"`
	Correction  string `json:"correction"`
}

type Result struct {
	Sentence    string
	IsCorrect   bool
	Explanation string
	Correction  string
	Err         error
}

const SystemPrompt = `You are an English grammar checker. Analyze the user's sentence for grammatical correctness.
Check for: subject-verb agreement, tense consistency, article usage, preposition usage, word order, punctuation, spelling, and overall clarity.
Respond ONLY with a single JSON object, no markdown, no explanation:
{
  "sentence": "the original sentence",
  "is_correct": bool,
  "explanation": "Brief explanation in simple English. If the sentence is correct, explain why its structure and grammar are valid. If incorrect, clearly point out each grammar error and why it is wrong.",
  "correction": "corrected version of the sentence if it was incorrect, otherwise empty string"
}`

func Check(ctx context.Context, llm llms.Model, sentence string) Result {
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, SystemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, sentence),
	}

	resp, err := llm.GenerateContent(ctx, messages)
	if err != nil {
		return Result{Sentence: sentence, Err: fmt.Errorf("llm error: %w", err)}
	}

	if len(resp.Choices) == 0 {
		return Result{Sentence: sentence, Err: fmt.Errorf("empty response from model")}
	}

	content := resp.Choices[0].Content

	var llmResp LLMResponse
	if err := json.Unmarshal([]byte(content), &llmResp); err != nil {
		content = extractJSON(content)
		if err := json.Unmarshal([]byte(content), &llmResp); err != nil {
			return Result{Sentence: sentence, Err: fmt.Errorf("failed to parse response")}
		}
	}

	return Result{
		Sentence:    llmResp.Sentence,
		IsCorrect:   llmResp.IsCorrect,
		Explanation: llmResp.Explanation,
		Correction:  llmResp.Correction,
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
