// Package ai wraps LLM calls for cluster analysis.
package ai

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

// Engine wraps an OpenAI-compatible LLM client.
type Engine struct {
	client *openai.Client
	model  string
}

// NewEngine creates an Engine with the given API key and model name.
// model defaults to gpt-4o-mini if empty.
func NewEngine(apiKey, model string) *Engine {
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &Engine{
		client: openai.NewClient(apiKey),
		model:  model,
	}
}

// ChatComplete sends a system + user message and returns the assistant response.
func (e *Engine) ChatComplete(ctx context.Context, system, user string) (string, error) {
	resp, err := e.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: e.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: system},
			{Role: openai.ChatMessageRoleUser, Content: user},
		},
	})
	if err != nil {
		return "", fmt.Errorf("llm chat: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("llm returned no choices")
	}
	return resp.Choices[0].Message.Content, nil
}
