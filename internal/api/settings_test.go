package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	aiPkg "github.com/clustermind/clustermind/internal/ai"
	"github.com/clustermind/clustermind/internal/graph"
	"github.com/clustermind/clustermind/internal/probe"
	"github.com/clustermind/clustermind/internal/render"
)

func setupTestServer() (*Server, *aiPkg.Engine) {
	engine := aiPkg.NewEngine("", "")
	analyzer := aiPkg.NewAnalyzer(engine)
	server := NewServer(
		probe.NewClusterCollector(nil, "demo"),
		graph.NewGraphBuilder(),
		render.NewRenderer(),
		analyzer,
		engine,
		"test-cluster",
	)
	return server, engine
}

func TestSettingsEndpoints(t *testing.T) {
	server, _ := setupTestServer()

	// 1. GET /api/v1/settings
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var getResp SettingsResponse
	err := json.Unmarshal(w.Body.Bytes(), &getResp)
	require.NoError(t, err)
	assert.Equal(t, aiPkg.ProviderGemini, getResp.Provider)
	assert.Equal(t, "gemini-2.0-flash", getResp.Model)
	assert.False(t, getResp.HasAPIKey)
	assert.Len(t, getResp.SupportedProviders, 2)

	// 2. POST /api/v1/settings (update to OpenAI)
	updatePayload := UpdateSettingsRequest{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "sk-test-1234567890abcdef",
	}
	body, _ := json.Marshal(updatePayload)
	postReq, _ := http.NewRequest(http.MethodPost, "/api/v1/settings", bytes.NewReader(body))
	postReq.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.router.ServeHTTP(w, postReq)

	assert.Equal(t, http.StatusOK, w.Code)
	var postResp SettingsResponse
	err = json.Unmarshal(w.Body.Bytes(), &postResp)
	require.NoError(t, err)
	assert.Equal(t, "openai", postResp.Provider)
	assert.Equal(t, "gpt-4o", postResp.Model)
	assert.True(t, postResp.HasAPIKey)
	assert.Contains(t, postResp.MaskedAPIKey, "sk-t")

	// 3. POST /api/v1/settings/test without key error
	testPayload := TestSettingsRequest{
		Provider: "openai",
		Model:    "gpt-4o",
		APIKey:   "",
	}
	body, _ = json.Marshal(testPayload)
	testReq, _ := http.NewRequest(http.MethodPost, "/api/v1/settings/test", bytes.NewReader(body))
	testReq.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.router.ServeHTTP(w, testReq)

	assert.Equal(t, http.StatusOK, w.Code)
	var testResp TestSettingsResponse
	err = json.Unmarshal(w.Body.Bytes(), &testResp)
	require.NoError(t, err)
	// Server should use the currently configured key and fail against real API or succeed
	// Since sk-test is fake, it will fail but handle gracefully
	assert.False(t, testResp.Success)
	assert.NotEmpty(t, testResp.Error)

	// 4. POST /api/v1/settings/models without key error
	modelsPayload := ListModelsRequest{
		Provider: "gemini",
		APIKey:   "",
	}
	body, _ = json.Marshal(modelsPayload)
	modelsReq, _ := http.NewRequest(http.MethodPost, "/api/v1/settings/models", bytes.NewReader(body))
	modelsReq.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	server.router.ServeHTTP(w, modelsReq)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
