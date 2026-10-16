package tests

import (
	"testing"

	"github.com/clustermind/clustermind/internal/graph"
	"github.com/clustermind/clustermind/tests/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphBuilder_LabelSelectorMatch(t *testing.T) {
	snap := fixtures.MockSnapshot()
	builder := graph.NewGraphBuilder()
	g := builder.Build(snap)

	require.NotNil(t, g)
	assert.Greater(t, len(g.Nodes), 0, "should have nodes")
	assert.Greater(t, len(g.Edges), 0, "should have edges")

	// Verify selector_match edges exist between services and deployments
	selectorEdges := 0
	for _, e := range g.Edges {
		if e.EdgeType == "selector_match" {
			selectorEdges++
		}
	}
	assert.Greater(t, selectorEdges, 0, "should have selector_match edges")
}

func TestGraphBuilder_IngressToService(t *testing.T) {
	snap := fixtures.MockSnapshot()
	builder := graph.NewGraphBuilder()
	g := builder.Build(snap)

	// ingress-nginx → frontend-svc
	ingID := graph.NodeID("Ingress", "demo", "ingress-nginx")
	svcID := graph.NodeID("Service", "demo", "frontend-svc")

	found := false
	for _, e := range g.Edges {
		if e.Source == ingID && e.Target == svcID && e.EdgeType == "ingress_backend" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected ingress_backend edge from ingress-nginx to frontend-svc")
}

func TestGraphBuilder_ServiceDeploymentMatch(t *testing.T) {
	snap := fixtures.MockSnapshot()
	builder := graph.NewGraphBuilder()
	g := builder.Build(snap)

	// payment-svc (selector: app=payment) → payment-deployment (label: app=payment)
	svcID := graph.NodeID("Service", "demo", "payment-svc")
	deplID := graph.NodeID("Deployment", "demo", "payment-deployment")

	found := false
	for _, e := range g.Edges {
		if e.Source == svcID && e.Target == deplID {
			found = true
			assert.GreaterOrEqual(t, e.Confidence, 0.9, "selector_match should have high confidence")
			break
		}
	}
	assert.True(t, found, "expected edge from payment-svc to payment-deployment")
}

func TestGraphBuilder_NodeIDFormat(t *testing.T) {
	id := graph.NodeID("Service", "demo", "payment-svc")
	assert.Equal(t, "Service/demo/payment-svc", id)

	clusterScoped := graph.NodeID("Namespace", "", "kube-system")
	assert.Equal(t, "Namespace/kube-system", clusterScoped)
}
