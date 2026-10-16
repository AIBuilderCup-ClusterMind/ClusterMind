// Package ai wraps LLM calls for cluster analysis.
package ai

import (
	"context"
	"fmt"

	genai "google.golang.org/genai"
)

// Engine wraps a Gemini generative AI client.
type Engine struct {
	client *genai.Client
	model  string
}

// NewEngine creates an Engine with the given Gemini API key and model name.
// model defaults to gemini-2.0-flash if empty.
func NewEngine(apiKey, model string) *Engine {
	if model == "" {
		model = "gemini-2.0-flash"
	}
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		// Return engine with nil client; ChatComplete will return error gracefully.
		return &Engine{client: nil, model: model}
	}
	return &Engine{client: client, model: model}
}

// ChatComplete sends a system + user message to Gemini and returns the response text.
func (e *Engine) ChatComplete(ctx context.Context, system, user string) (string, error) {
	if e.client == nil {
		return "", fmt.Errorf("gemini client not initialised (check GEMINI_API_KEY)")
	}

	contents := []*genai.Content{
		{Role: "user", Parts: []*genai.Part{
			{Text: user},
		}},
	}

	config := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: system}},
		},
	}

	resp, err := e.client.Models.GenerateContent(ctx, e.model, contents, config)
	if err != nil {
		return "", fmt.Errorf("gemini generate: %w", err)
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned no content")
	}
	return resp.Candidates[0].Content.Parts[0].Text, nil
}
