package ai

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEngine_ModelConfiguration(t *testing.T) {
	// 1. Default model should be gemini-3.5-flash-lite
	origEnv := os.Getenv("GEMINI_MODEL")
	defer os.Setenv("GEMINI_MODEL", origEnv)

	os.Unsetenv("GEMINI_MODEL")
	eng := NewEngine("", "")
	assert.Equal(t, "gemini-3.5-flash-lite", eng.Model())
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

	engUnconfigured := &Engine{client: nil, model: "custom-model"}
	assert.False(t, engUnconfigured.IsConfigured())
	assert.Equal(t, "custom-model", engUnconfigured.Model())
}
