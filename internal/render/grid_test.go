package render

import (
	"testing"

	"github.com/clustermind/clustermind/internal/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGridPositions_GroupsByKindRows(t *testing.T) {
	nodes := []graph.ResourceNode{
		{ID: "Ingress/ns/ing", Kind: "Ingress"},
		{ID: "Service/ns/svc", Kind: "Service"},
		{ID: "Deployment/ns/a", Kind: "Deployment"},
		{ID: "Deployment/ns/b", Kind: "Deployment"},
		{ID: "CustomKind/ns/x", Kind: "CustomKind"},
	}

	pos := GridPositions(nodes)

	require.Len(t, pos, len(nodes))
	assert.Equal(t, Position{X: 0, Y: 0}, pos["Ingress/ns/ing"])
	assert.Equal(t, Position{X: 0, Y: 140}, pos["Service/ns/svc"])
	assert.Equal(t, Position{X: 0, Y: 280}, pos["Deployment/ns/a"])
	assert.Equal(t, Position{X: 220, Y: 280}, pos["Deployment/ns/b"])
	assert.Equal(t, Position{X: 0, Y: 700}, pos["CustomKind/ns/x"])
}

func TestGridPositions_Empty(t *testing.T) {
	assert.Empty(t, GridPositions(nil))
}
