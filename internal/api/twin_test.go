package api

import (
	"testing"

	"github.com/clustermind/clustermind/internal/probe"
	"github.com/stretchr/testify/assert"
)

func TestHealthFromPhase(t *testing.T) {
	cases := map[probe.StatusPhase]string{
		probe.StatusHealthy:    "healthy",
		probe.StatusCompleted:  "healthy",
		probe.StatusWarning:    "degraded",
		probe.StatusPending:    "degraded",
		probe.StatusDegraded:   "unhealthy",
		probe.StatusUnknown:    "unknown",
		probe.StatusPhase("?"): "unknown",
	}
	for phase, want := range cases {
		assert.Equal(t, want, healthFromPhase(phase), "phase %q", phase)
	}
}

func TestTwinMetrics(t *testing.T) {
	t.Run("deployment with ready and age", func(t *testing.T) {
		rs := probe.ResourceStatus{Ready: "3/3", Age: "2d3h", Reason: "Available"}
		metrics := twinMetrics(rs)
		assert.Equal(t, []TwinMetric{
			{Label: "Ready", Value: "3/3"},
			{Label: "Age", Value: "2d3h"},
		}, metrics)
	})

	t.Run("resource without ready info", func(t *testing.T) {
		rs := probe.ResourceStatus{Ready: "—", Age: "5h2m"}
		metrics := twinMetrics(rs)
		assert.Equal(t, []TwinMetric{
			{Label: "Age", Value: "5h2m"},
		}, metrics)
	})

	t.Run("empty status yields no metrics", func(t *testing.T) {
		assert.Empty(t, twinMetrics(probe.ResourceStatus{}))
	})
}
