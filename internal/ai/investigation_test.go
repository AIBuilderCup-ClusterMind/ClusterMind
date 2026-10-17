package ai

import (
	"context"
	"strings"
	"testing"

	"github.com/clustermind/clustermind/internal/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testGraph() *graph.DependencyGraph {
	return &graph.DependencyGraph{
		Nodes: []graph.ResourceNode{
			{ID: "Service/demo/web-svc", Kind: "Service", Name: "web-svc", Namespace: "demo"},
			{ID: "Deployment/demo/web", Kind: "Deployment", Name: "web", Namespace: "demo"},
		},
		Edges: []graph.DependencyEdge{
			{Source: "Service/demo/web-svc", Target: "Deployment/demo/web", EdgeType: "selector_match", Confidence: 1.0},
		},
	}
}

func testTimeline() []EvidenceInput {
	return []EvidenceInput{
		{ID: "ev-1", Time: "2026-10-17T08:00:00Z", Source: "events", Title: "BackOff", Detail: "Back-off restarting failed container"},
		{ID: "ev-2", Time: "2026-10-17T08:01:00Z", Source: "alert", Title: "Deployment/demo/web is unhealthy", Detail: "0 of 1 desired replicas ready"},
	}
}

func TestInvestigate_FallbackWithoutEngine(t *testing.T) {
	a := NewAnalyzer(nil)
	result := a.Investigate(context.Background(), InvestigationRequest{
		IncidentID:    "inc-deployment-demo-web",
		PrimaryTarget: "Deployment/demo/web",
		Health:        "unhealthy",
		Reason:        "MinimumReplicasUnavailable",
		Timeline:      testTimeline(),
	}, testGraph())

	require.NotNil(t, result)
	assert.Equal(t, "inc-deployment-demo-web", result.IncidentID)
	assert.Equal(t, "fallback", result.GeneratedBy)
	assert.NotEmpty(t, result.Summary)
	assert.NotEmpty(t, result.ObservedFacts)
	assert.NotEmpty(t, result.PossibleCauses)
	assert.NotEmpty(t, result.NextChecks)
	// BackOff signature should be detected.
	found := false
	for _, c := range result.PossibleCauses {
		if c.Confidence == "medium" && strings.Contains(c.Cause, "back-off") {
			found = true
		}
	}
	assert.True(t, found, "expected a medium-confidence back-off cause from the evidence")
}

func TestValidateEvidence_DropsUnknownIDs(t *testing.T) {
	timeline := testTimeline()
	refs := []SupportingEvidenceRef{
		{EvidenceID: "ev-1", Note: "valid"},
		{EvidenceID: "ev-999", Note: "hallucinated"},
	}
	out := validateEvidence(refs, timeline)
	require.Len(t, out, 1)
	assert.Equal(t, "ev-1", out[0].EvidenceID)
}

func TestClampCauses(t *testing.T) {
	out := clampCauses([]PossibleCause{
		{Cause: "a", Confidence: "high"},
		{Cause: "b", Confidence: "certain"}, // invalid → low
		{Cause: "  ", Confidence: "medium"}, // dropped
	})
	require.Len(t, out, 2)
	assert.Equal(t, "high", out[0].Confidence)
	assert.Equal(t, "low", out[1].Confidence)
}

func TestDetectCauses_Signatures(t *testing.T) {
	causes := detectCauses(InvestigationRequest{
		Health: "unhealthy",
		Timeline: []EvidenceInput{
			{ID: "e1", Title: "OOMKilled", Detail: "Container was OOMKilled"},
			{ID: "e2", Title: "Unhealthy", Detail: "Readiness probe failed"},
		},
	})
	require.Len(t, causes, 2)
	assert.Equal(t, "high", causes[0].Confidence)
	assert.Equal(t, "medium", causes[1].Confidence)
}
