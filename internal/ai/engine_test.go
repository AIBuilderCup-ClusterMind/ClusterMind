package ai

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEngine_ModelConfiguration(t *testing.T) {
	t.Setenv("CLUSTERMIND_DISABLE_PERSISTENCE", "1")
	// 1. Default model should be DefaultModel (gemini-3.8-flash)
	origEnv := os.Getenv("GEMINI_MODEL")
	defer os.Setenv("GEMINI_MODEL", origEnv)

	os.Unsetenv("GEMINI_MODEL")
	eng := NewEngine("", "")
	assert.Equal(t, DefaultModel, eng.Model())
	assert.False(t, eng.IsConfigured())

	// 2. GEMINI_MODEL env var override
	os.Setenv("GEMINI_MODEL", "gemini-1.5-pro")
	engEnv := NewEngine("", "")
	assert.Equal(t, "gemini-1.5-pro", engEnv.Model())

	// 3. Explicit parameter takes precedence over GEMINI_MODEL env var
	engExplicit := NewEngine("", "gemini-1.5-flash")
	assert.Equal(t, "gemini-1.5-flash", engExplicit.Model())
}

func TestEngine_IsConfigured_NilSafety(t *testing.T) {
	var eng *Engine
	assert.False(t, eng.IsConfigured())
	assert.Equal(t, DefaultModel, eng.Model())

	engUnconfigured := &Engine{geminiClient: nil, model: "custom-model"}
	assert.False(t, engUnconfigured.IsConfigured())
	assert.Equal(t, "custom-model", engUnconfigured.Model())
}

func TestEngineConfiguration(t *testing.T) {
	t.Setenv("CLUSTERMIND_DISABLE_PERSISTENCE", "1")
	e := NewEngine("", "")
	assert.Equal(t, ProviderGemini, e.Provider())
	assert.Equal(t, DefaultModel, e.Model())
	assert.False(t, e.HasAPIKey())
	assert.Equal(t, "", e.MaskedAPIKey())

	// Configure Gemini with key
	err := e.Configure(ProviderGemini, "AIzaSyTestKey1234567890", "gemini-1.5-pro")
	assert.NoError(t, err)
	assert.Equal(t, ProviderGemini, e.Provider())
	assert.Equal(t, "gemini-1.5-pro", e.Model())
	assert.True(t, e.HasAPIKey())
	assert.Equal(t, "AIza••••7890", e.MaskedAPIKey())

	// Switch to OpenAI without changing key
	err = e.Configure(ProviderOpenAI, "", "")
	assert.NoError(t, err)
	assert.Equal(t, ProviderOpenAI, e.Provider())
	assert.Equal(t, "gemini-1.5-pro", e.Model()) // kept previous model since empty
	assert.True(t, e.HasAPIKey())

	// Switch model to gpt-4o and key to sk-test
	err = e.Configure(ProviderOpenAI, "sk-proj-1234567890abcdef", "gpt-4o")
	assert.NoError(t, err)
	assert.Equal(t, "gpt-4o", e.Model())
	assert.Equal(t, "sk-p••••cdef", e.MaskedAPIKey())

	// Test invalid provider
	err = e.Configure("unknown-provider", "", "")
	assert.Error(t, err)
}

func TestEngineChatCompleteWithoutKey(t *testing.T) {
	e := NewEngine("", "")
	_, err := e.ChatComplete(context.Background(), "sys", "user")
	assert.Error(t, err)

	_ = e.Configure(ProviderOpenAI, "", "gpt-4o")
	_, err = e.ChatComplete(context.Background(), "sys", "user")
	assert.Error(t, err)
}

func TestSupportedProviders(t *testing.T) {
	providers := SupportedProviders()
	assert.Len(t, providers, 2)
	assert.Equal(t, ProviderGemini, providers[0].ID)
	assert.Equal(t, ProviderOpenAI, providers[1].ID)

	assert.Contains(t, providers[0].Models, "gemini-3.8-flash")
	assert.Contains(t, providers[0].Models, "gemini-3.5-flash-lite")
}

func TestFetchAvailableModelsWithoutKey(t *testing.T) {
	e := NewEngine("", "")
	_, err := e.FetchAvailableModels(context.Background(), ProviderGemini, "")
	assert.Error(t, err)

	_, err = e.FetchAvailableModels(context.Background(), ProviderOpenAI, "")
	assert.Error(t, err)
}

func TestSortGeminiModels(t *testing.T) {
	items := []ModelItem{
		{ID: "gemini-1.5-pro", DisplayName: "Gemini 1.5 Pro"},
		{ID: "gemini-3.8-flash", DisplayName: "Gemini 3.8 Flash"},
		{ID: "gemini-3.5-flash-lite", DisplayName: "Gemini 3.5 Flash Lite"},
		{ID: "gemini-2.5-flash", DisplayName: "Gemini 2.5 Flash"},
	}
	sortGeminiModels(items)
	assert.Equal(t, "gemini-3.8-flash", items[0].ID)
	assert.Equal(t, "gemini-3.5-flash-lite", items[1].ID)
	assert.Equal(t, "gemini-2.5-flash", items[2].ID)
	assert.Equal(t, "gemini-1.5-pro", items[3].ID)
}

func TestEnginePersistence(t *testing.T) {
	tempFile := t.TempDir() + "/settings.json"
	t.Setenv("CLUSTERMIND_CONFIG_PATH", tempFile)
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("GEMINI_MODEL", "")

	// 1. Initial engine has default model
	e1 := NewEngine("", "")
	assert.Equal(t, DefaultModel, e1.Model())

	// 2. Configure with custom model & key (persists to tempFile)
	err := e1.Configure(ProviderOpenAI, "sk-test-saved-key-1234", "gpt-4o-mini")
	assert.NoError(t, err)
	assert.FileExists(t, tempFile)

	// 3. New engine instance simulating server restart reads persisted settings
	e2 := NewEngine("", "")
	assert.Equal(t, ProviderOpenAI, e2.Provider())
	assert.Equal(t, "gpt-4o-mini", e2.Model())
	assert.Equal(t, "sk-test-saved-key-1234", e2.APIKey())
}
