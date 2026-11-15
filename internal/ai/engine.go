// Package ai wraps LLM calls for cluster analysis.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	genai "google.golang.org/genai"
)

const (
	// ProviderGemini identifier.
	ProviderGemini = "gemini"
	// ProviderOpenAI identifier.
	ProviderOpenAI = "openai"

	// DefaultModel is the recommended Gemini model for low-latency reasoning.
	DefaultModel = "gemini-3.8-flash"
)

// ModelItem represents a selectable AI model with display metadata.
type ModelItem struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Description string `json:"description,omitempty"`
}

// ProviderMeta describes an AI provider and its available models.
type ProviderMeta struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	DefaultModel string   `json:"defaultModel"`
	Models       []string `json:"models"`
	KeyHelpURL   string   `json:"keyHelpUrl"`
	KeyPrefix    string   `json:"keyPrefix"`
}

// SupportedProviders returns metadata for all supported AI providers.
func SupportedProviders() []ProviderMeta {
	return []ProviderMeta{
		{
			ID:           ProviderGemini,
			Name:         "Google Gemini",
			DefaultModel: "gemini-3.8-flash",
			Models: []string{
				"gemini-3.8-flash",
				"gemini-3.5-flash-lite",
				"gemini-3.5-pro",
				"gemini-2.5-flash",
			},
			KeyHelpURL: "https://aistudio.google.com/apikey",
			KeyPrefix:  "AIza",
		},
		{
			ID:           ProviderOpenAI,
			Name:         "OpenAI",
			DefaultModel: "gpt-4o",
			Models: []string{
				"gpt-4o",
				"gpt-4o-mini",
				"o3-mini",
				"gpt-4-turbo",
			},
			KeyHelpURL: "https://platform.openai.com/api-keys",
			KeyPrefix:  "sk-",
		},
	}
}

// Engine wraps AI client calls with thread-safe provider and model configuration.
type Engine struct {
	mu           sync.RWMutex
	provider     string
	apiKey       string
	model        string
	geminiClient *genai.Client
	httpClient   *http.Client
}

// NewEngine creates an Engine with the given Gemini API key and model name.
// If model is empty, it reads GEMINI_MODEL from the environment, defaulting
// to DefaultModel ("gemini-3.8-flash").
// If apiKey is empty, it reads GEMINI_API_KEY from the environment.
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

	e := &Engine{
		provider:   ProviderGemini,
		model:      model,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}

	if apiKey != "" {
		client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
			APIKey:  apiKey,
			Backend: genai.BackendGeminiAPI,
		})
		if err == nil {
			e.geminiClient = client
		}
	}
	return e
}

// Configure updates the provider, apiKey, and model at runtime.
// If provider is empty, the current provider is kept.
// If apiKey is empty, the current apiKey is kept.
// If model is empty, a sensible default for the provider is chosen.
func (e *Engine) Configure(provider, apiKey, model string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if provider != "" {
		provider = strings.ToLower(strings.TrimSpace(provider))
		if provider != ProviderGemini && provider != ProviderOpenAI {
			return fmt.Errorf("unsupported AI provider: %s", provider)
		}
		e.provider = provider
	}
	if e.provider == "" {
		e.provider = ProviderGemini
	}

	if apiKey != "" {
		e.apiKey = strings.TrimSpace(apiKey)
	}

	if model != "" {
		e.model = strings.TrimSpace(model)
	} else if e.model == "" {
		if e.provider == ProviderOpenAI {
			e.model = "gpt-4o"
		} else {
			e.model = "gemini-3.8-flash"
		}
	}

	if e.httpClient == nil {
		e.httpClient = &http.Client{Timeout: 60 * time.Second}
	}

	// Re-initialize provider clients
	if e.provider == ProviderGemini {
		if e.apiKey != "" {
			client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
				APIKey:  e.apiKey,
				Backend: genai.BackendGeminiAPI,
			})
			if err != nil {
				e.geminiClient = nil
				return fmt.Errorf("gemini init: %w", err)
			}
			e.geminiClient = client
		} else {
			e.geminiClient = nil
		}
	}

	return nil
}

// Provider returns the current provider.
func (e *Engine) Provider() string {
	if e == nil {
		return ProviderGemini
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.provider == "" {
		return ProviderGemini
	}
	return e.provider
}

// Model returns the configured model name.
func (e *Engine) Model() string {
	if e == nil {
		return DefaultModel
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.model == "" {
		return DefaultModel
	}
	return e.model
}

// IsConfigured returns true if the engine is initialized with an active API key or client.
func (e *Engine) IsConfigured() bool {
	if e == nil {
		return false
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.provider == ProviderOpenAI {
		return strings.TrimSpace(e.apiKey) != ""
	}
	return e.geminiClient != nil
}

// HasAPIKey reports whether an API key is currently configured.
func (e *Engine) HasAPIKey() bool {
	if e == nil {
		return false
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return strings.TrimSpace(e.apiKey) != ""
}

// APIKey returns the current API key.
func (e *Engine) APIKey() string {
	if e == nil {
		return ""
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.apiKey
}

// MaskedAPIKey returns a partially masked version of the active API key for safe UI display.
func (e *Engine) MaskedAPIKey() string {
	if e == nil {
		return ""
	}
	e.mu.RLock()
	defer e.mu.RUnlock()
	key := strings.TrimSpace(e.apiKey)
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "••••••••"
	}
	prefix := key[:4]
	suffix := key[len(key)-4:]
	return fmt.Sprintf("%s••••%s", prefix, suffix)
}

// ChatComplete sends a system + user message to the active AI provider and returns the response text.
func (e *Engine) ChatComplete(ctx context.Context, system, user string) (string, error) {
	if e == nil {
		return "", fmt.Errorf("engine not initialised")
	}
	e.mu.RLock()
	provider := e.provider
	apiKey := e.apiKey
	model := e.model
	geminiClient := e.geminiClient
	httpClient := e.httpClient
	e.mu.RUnlock()

	if provider == ProviderOpenAI {
		return chatCompleteOpenAI(ctx, httpClient, apiKey, model, system, user)
	}

	return chatCompleteGemini(ctx, geminiClient, model, system, user)
}

// ChatJSON sends a system + user prompt and returns the JSON string response.
// For Gemini, it sets ResponseMIMEType to "application/json".
// For OpenAI, it uses ChatComplete.
func (e *Engine) ChatJSON(ctx context.Context, system, user string) (string, error) {
	if e == nil {
		return "", fmt.Errorf("engine not initialised")
	}
	e.mu.RLock()
	provider := e.provider
	apiKey := e.apiKey
	model := e.model
	geminiClient := e.geminiClient
	httpClient := e.httpClient
	e.mu.RUnlock()

	if provider == ProviderOpenAI {
		return chatCompleteOpenAI(ctx, httpClient, apiKey, model, system, user)
	}

	if geminiClient == nil {
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

	resp, err := geminiClient.Models.GenerateContent(ctx, model, contents, config)
	if err != nil {
		return "", fmt.Errorf("gemini generate json: %w", err)
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned no content")
	}
	return resp.Candidates[0].Content.Parts[0].Text, nil
}

// Test verifies connection with the specified or current configuration.
func (e *Engine) Test(ctx context.Context, provider, apiKey, model string) error {
	e.mu.RLock()
	currProvider := e.provider
	currKey := e.apiKey
	currModel := e.model
	httpClient := e.httpClient
	e.mu.RUnlock()

	if provider == "" {
		provider = currProvider
	}
	if apiKey == "" {
		apiKey = currKey
	}
	if model == "" {
		if currModel != "" {
			model = currModel
		} else if provider == ProviderOpenAI {
			model = "gpt-4o"
		} else {
			model = "gemini-3.8-flash"
		}
	}
	if strings.TrimSpace(apiKey) == "" {
		return fmt.Errorf("API key is required")
	}

	testSystem := "You are a health check assistant."
	testUser := "Reply with 'PONG'."

	if provider == ProviderOpenAI {
		res, err := chatCompleteOpenAI(ctx, httpClient, apiKey, model, testSystem, testUser)
		if err != nil {
			return err
		}
		if strings.TrimSpace(res) == "" {
			return fmt.Errorf("empty response received from OpenAI")
		}
		return nil
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize Gemini client: %w", err)
	}
	res, err := chatCompleteGemini(ctx, client, model, testSystem, testUser)
	if err != nil {
		return err
	}
	if strings.TrimSpace(res) == "" {
		return fmt.Errorf("empty response received from Gemini")
	}
	return nil
}

func chatCompleteGemini(ctx context.Context, client *genai.Client, model, system, user string) (string, error) {
	if client == nil {
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

	resp, err := client.Models.GenerateContent(ctx, model, contents, config)
	if err != nil {
		return "", fmt.Errorf("gemini generate: %w", err)
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned no content")
	}
	return resp.Candidates[0].Content.Parts[0].Text, nil
}

func chatCompleteOpenAI(ctx context.Context, client *http.Client, apiKey, model, system, user string) (string, error) {
	if strings.TrimSpace(apiKey) == "" {
		return "", fmt.Errorf("openai client not initialised (API key not set)")
	}
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type requestBody struct {
		Model       string    `json:"model"`
		Messages    []message `json:"messages"`
		Temperature float64   `json:"temperature"`
	}

	reqPayload := requestBody{
		Model: model,
		Messages: []message{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature: 0.2,
	}

	jsonBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return "", fmt.Errorf("marshal openai request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(jsonBytes))
	if err != nil {
		return "", fmt.Errorf("create openai request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read openai response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		if json.Unmarshal(bodyBytes, &errResp) == nil && errResp.Error.Message != "" {
			return "", fmt.Errorf("openai api error (%d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return "", fmt.Errorf("openai api error (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var completion struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(bodyBytes, &completion); err != nil {
		return "", fmt.Errorf("unmarshal openai completion: %w", err)
	}

	if len(completion.Choices) == 0 || completion.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("openai returned empty response")
	}

	return completion.Choices[0].Message.Content, nil
}

// FetchAvailableModels queries the provider API in real-time to list currently available models.
func (e *Engine) FetchAvailableModels(ctx context.Context, provider, apiKey string) ([]ModelItem, error) {
	e.mu.RLock()
	if provider == "" {
		provider = e.provider
	}
	if apiKey == "" {
		apiKey = e.apiKey
	}
	httpClient := e.httpClient
	e.mu.RUnlock()

	if provider == "" {
		provider = ProviderGemini
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		return nil, fmt.Errorf("API key is required to fetch available models for %s", provider)
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	if provider == ProviderOpenAI {
		return fetchOpenAIModels(ctx, httpClient, apiKey)
	}
	return fetchGeminiModels(ctx, httpClient, apiKey)
}

func fetchGeminiModels(ctx context.Context, client *http.Client, apiKey string) ([]ModelItem, error) {
	reqURL := "https://generativelanguage.googleapis.com/v1beta/models?pageSize=100&key=" + url.QueryEscape(apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create gemini list request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch gemini models: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read gemini models response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			} `json:"error"`
		}
		if json.Unmarshal(bodyBytes, &errResp) == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("gemini api error (%d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return nil, fmt.Errorf("gemini api error (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res struct {
		Models []struct {
			Name                       string   `json:"name"`
			DisplayName                string   `json:"displayName"`
			Description                string   `json:"description"`
			SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
		} `json:"models"`
	}

	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return nil, fmt.Errorf("unmarshal gemini models: %w", err)
	}

	var items []ModelItem
	for _, m := range res.Models {
		// Only include models that support generateContent
		hasGenerateContent := false
		for _, method := range m.SupportedGenerationMethods {
			if method == "generateContent" {
				hasGenerateContent = true
				break
			}
		}
		if !hasGenerateContent {
			continue
		}

		id := strings.TrimPrefix(m.Name, "models/")
		if !strings.HasPrefix(id, "gemini-") {
			continue
		}

		displayName := m.DisplayName
		if displayName == "" {
			displayName = id
		}

		items = append(items, ModelItem{
			ID:          id,
			DisplayName: displayName,
			Description: m.Description,
		})
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no generateContent-capable Gemini models found")
	}

	sortGeminiModels(items)
	return items, nil
}

func sortGeminiModels(items []ModelItem) {
	priority := func(id string) int {
		switch id {
		case "gemini-3.8-flash":
			return 100
		case "gemini-3.5-flash-lite":
			return 95
		case "gemini-3.5-pro":
			return 90
		case "gemini-3.0-flash":
			return 85
		case "gemini-2.5-flash":
			return 80
		case "gemini-2.0-flash":
			return 40
		case "gemini-2.0-flash-lite":
			return 35
		case "gemini-1.5-flash":
			return 25
		case "gemini-1.5-flash-8b":
			return 20
		case "gemini-1.5-pro":
			return 15
		}
		if strings.HasPrefix(id, "gemini-3.8") {
			return 92
		}
		if strings.HasPrefix(id, "gemini-3.5") {
			return 88
		}
		if strings.HasPrefix(id, "gemini-3") {
			return 82
		}
		if strings.HasPrefix(id, "gemini-2.5") {
			return 70
		}
		if strings.HasPrefix(id, "gemini-2") {
			return 30
		}
		if strings.HasPrefix(id, "gemini-1.5") {
			return 20
		}
		return 10
	}

	sort.SliceStable(items, func(i, j int) bool {
		pi := priority(items[i].ID)
		pj := priority(items[j].ID)
		if pi != pj {
			return pi > pj
		}
		return items[i].ID < items[j].ID
	})
}

func fetchOpenAIModels(ctx context.Context, client *http.Client, apiKey string) ([]ModelItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/models", nil)
	if err != nil {
		return nil, fmt.Errorf("create openai list request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch openai models: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read openai models response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(bodyBytes, &errResp) == nil && errResp.Error.Message != "" {
			return nil, fmt.Errorf("openai api error (%d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return nil, fmt.Errorf("openai api error (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	var res struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return nil, fmt.Errorf("unmarshal openai models: %w", err)
	}

	var items []ModelItem
	seen := make(map[string]bool)
	for _, m := range res.Data {
		id := m.ID
		if seen[id] {
			continue
		}
		if !strings.HasPrefix(id, "gpt-") && !strings.HasPrefix(id, "o1") && !strings.HasPrefix(id, "o3") {
			continue
		}
		if strings.Contains(id, "audio") || strings.Contains(id, "realtime") || strings.Contains(id, "instruct") || strings.Contains(id, "embedding") {
			continue
		}

		seen[id] = true
		items = append(items, ModelItem{
			ID:          id,
			DisplayName: id,
		})
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no suitable OpenAI chat models found")
	}

	sortOpenAIModels(items)
	return items, nil
}

func sortOpenAIModels(items []ModelItem) {
	priority := func(id string) int {
		switch id {
		case "gpt-4o":
			return 100
		case "gpt-4o-mini":
			return 95
		case "o3-mini":
			return 90
		case "o1":
			return 85
		case "o1-mini":
			return 80
		case "gpt-4-turbo":
			return 75
		}
		if strings.HasPrefix(id, "gpt-4o") {
			return 65
		}
		if strings.HasPrefix(id, "gpt-4") {
			return 55
		}
		return 10
	}

	sort.SliceStable(items, func(i, j int) bool {
		pi := priority(items[i].ID)
		pj := priority(items[j].ID)
		if pi != pj {
			return pi > pj
		}
		return items[i].ID < items[j].ID
	})
}
