// Package ai wraps LLM calls for cluster analysis.
package ai

import (
	"context"
	"fmt"
	"os"

	genai "google.golang.org/genai"
)

// DefaultModel is the recommended Gemini flash-lite model for low-latency reasoning in new projects.
const DefaultModel = "gemini-3.5-flash-lite"

// Engine wraps a Gemini generative AI client.
type Engine struct {
	client *genai.Client
	model  string
}

// NewEngine creates an Engine with the given Gemini API key and model name.
// If model is empty, it reads GEMINI_MODEL from the environment, defaulting
// to DefaultModel ("gemini-3.5-flash-lite").
func NewEngine(apiKey, model string) *Engine {
	if model == "" {
		model = os.Getenv("GEMINI_MODEL")
	}
	if model == "" {
		model = DefaultModel
	}
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}

	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		// Return engine with nil client; calls will fallback gracefully.
		return &Engine{client: nil, model: model}
	}
	return &Engine{client: client, model: model}
}

// Model returns the configured Gemini model name.
func (e *Engine) Model() string {
	if e == nil {
		return DefaultModel
	}
	return e.model
}

// IsConfigured returns true if the Gemini client was successfully initialized.
func (e *Engine) IsConfigured() bool {
	return e != nil && e.client != nil
}

// ChatComplete sends a system + user message to Gemini and returns the response text.
func (e *Engine) ChatComplete(ctx context.Context, system, user string) (string, error) {
	if e == nil || e.client == nil {
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

// ChatJSON sends a system + user prompt with ResponseMIMEType set to "application/json".
func (e *Engine) ChatJSON(ctx context.Context, system, user string) (string, error) {
	if e == nil || e.client == nil {
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
		ResponseMIMEType: "application/json",
	}

	resp, err := e.client.Models.GenerateContent(ctx, e.model, contents, config)
	if err != nil {
		return "", fmt.Errorf("gemini generate json: %w", err)
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned no content")
	}
	return resp.Candidates[0].Content.Parts[0].Text, nil
}
