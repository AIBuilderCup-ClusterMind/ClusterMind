package ai

import (
	"testing"

	"github.com/clustermind/clustermind/internal/graph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// demoGraph mirrors the demo cluster chain:
//
//	ingress → frontend-svc ↔ frontend-deployment
//	  frontend-deployment --env--> api-svc ↔ api-deployment
//	  api-deployment --env--> payment-svc ↔ payment-deployment
//	  api-deployment --env--> postgres-svc ↔ postgres-deployment
func demoGraph() *graph.DependencyGraph {
	edge := func(src, tgt, kind string, conf float64) graph.DependencyEdge {
		return graph.DependencyEdge{Source: src, Target: tgt, EdgeType: kind, Confidence: conf}
	}
	node := func(kind, ns, name string) graph.ResourceNode {
		return graph.ResourceNode{ID: graph.NodeID(kind, ns, name), Kind: kind, Name: name, Namespace: ns}
	}
	return &graph.DependencyGraph{
		Nodes: []graph.ResourceNode{
			node("Ingress", "demo", "ingress-nginx"),
			node("Service", "demo", "frontend-svc"),
			node("Deployment", "demo", "frontend-deployment"),
			node("Service", "demo", "api-svc"),
			node("Deployment", "demo", "api-deployment"),
			node("Service", "demo", "payment-svc"),
			node("Deployment", "demo", "payment-deployment"),
			node("Service", "demo", "postgres-svc"),
			node("Deployment", "demo", "postgres-deployment"),
		},
		Edges: []graph.DependencyEdge{
			edge("Ingress/demo/ingress-nginx", "Service/demo/frontend-svc", "ingress_backend", 1.0),
			edge("Service/demo/frontend-svc", "Deployment/demo/frontend-deployment", "selector_match", 0.95),
			edge("Deployment/demo/frontend-deployment", "Service/demo/api-svc", "env_reference", 0.7),
			edge("Service/demo/api-svc", "Deployment/demo/api-deployment", "selector_match", 0.95),
			edge("Deployment/demo/api-deployment", "Service/demo/payment-svc", "env_reference", 0.7),
			edge("Service/demo/payment-svc", "Deployment/demo/payment-deployment", "selector_match", 0.95),
			edge("Deployment/demo/api-deployment", "Service/demo/postgres-svc", "env_reference", 0.7),
			edge("Service/demo/postgres-svc", "Deployment/demo/postgres-deployment", "selector_match", 0.95),
		},
	}
}

func TestTraverse_PaymentFailureBlastRadius(t *testing.T) {
	impactMap, paths, affected := traverse(demoGraph(), "Service/demo/payment-svc")

	assert.Equal(t, "critical", impactMap["Service/demo/payment-svc"])
	// Direct: the failing unit's workload and its caller.
	assert.Equal(t, "direct", impactMap["Deployment/demo/payment-deployment"])
	assert.Equal(t, "direct", impactMap["Deployment/demo/api-deployment"])
	// Indirect: callers keep climbing to the ingress.
	assert.Equal(t, "indirect", impactMap["Service/demo/api-svc"])
	assert.Equal(t, "indirect", impactMap["Deployment/demo/frontend-deployment"])
	assert.Equal(t, "indirect", impactMap["Service/demo/frontend-svc"])
	assert.Equal(t, "indirect", impactMap["Ingress/demo/ingress-nginx"])
	// Unrelated data tier must not be flagged (forward env edges do not propagate).
	assert.Equal(t, "none", impactMap["Service/demo/postgres-svc"])
	assert.Equal(t, "none", impactMap["Deployment/demo/postgres-deployment"])
	// Target itself is excluded from the affected list.
	assert.NotContains(t, affected, "Service/demo/payment-svc")
	assert.Len(t, affected, 6)
	assert.NotEmpty(t, paths)
}

func TestTraverse_DeploymentFailureDegradesItsService(t *testing.T) {
	impactMap, _, _ := traverse(demoGraph(), "Deployment/demo/api-deployment")

	assert.Equal(t, "critical", impactMap["Deployment/demo/api-deployment"])
	// Reverse selector: workload down → its Service degraded.
	assert.Equal(t, "direct", impactMap["Service/demo/api-svc"])
	// Callers of api-svc keep climbing.
	assert.Equal(t, "indirect", impactMap["Deployment/demo/frontend-deployment"])
	assert.Equal(t, "indirect", impactMap["Service/demo/frontend-svc"])
	assert.Equal(t, "indirect", impactMap["Ingress/demo/ingress-nginx"])
	// Callees of api-deployment are not impacted.
	assert.Equal(t, "none", impactMap["Service/demo/payment-svc"])
	assert.Equal(t, "none", impactMap["Service/demo/postgres-svc"])
}

func TestTraverse_DepthCapAndConfidenceFilter(t *testing.T) {
	g := &graph.DependencyGraph{}
	// Caller chain a → b → c → d → e → f → g (a calls b, …, f calls g),
	// so a failure of g climbs back to a in six hops.
	ids := []string{"a", "b", "c", "d", "e", "f", "g"}
	for _, id := range ids {
		g.Nodes = append(g.Nodes, graph.ResourceNode{ID: id})
	}
	for i := 0; i < len(ids)-1; i++ {
		g.Edges = append(g.Edges, graph.DependencyEdge{
			Source: ids[i], Target: ids[i+1], EdgeType: "env_reference", Confidence: 0.9,
		})
	}
	// Low-confidence caller edge is never traversed.
	g.Edges = append(g.Edges, graph.DependencyEdge{
		Source: "h", Target: "c", EdgeType: "env_reference", Confidence: 0.1,
	})

	impactMap, _, _ := traverse(g, "g")

	require.Equal(t, "critical", impactMap["g"])
	assert.Equal(t, "direct", impactMap["f"])
	assert.Equal(t, "indirect", impactMap["e"])
	assert.Equal(t, "indirect", impactMap["b"])
	// Six hops beyond the target → capped out (left as none).
	assert.Equal(t, "none", impactMap["a"])
	// h is not even a graph node; the low-confidence edge never surfaces it.
	assert.NotContains(t, impactMap, "h")
}
