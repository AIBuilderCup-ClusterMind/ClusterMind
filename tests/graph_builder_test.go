package tests

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/clustermind/clustermind/internal/graph"
	"github.com/clustermind/clustermind/internal/probe"
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

func TestGraphBuilder_EnvURLReference(t *testing.T) {
	snap := &probe.ClusterSnapshot{
		Services: []corev1.Service{
			{ObjectMeta: metav1.ObjectMeta{Name: "payment-svc", Namespace: "demo"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "postgres-svc", Namespace: "demo"}},
		},
		Deployments: []appsv1.Deployment{{
			ObjectMeta: metav1.ObjectMeta{Name: "api-deployment", Namespace: "demo"},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Name: "api",
							Env: []corev1.EnvVar{
								{Name: "PAYMENT_SERVICE_URL", Value: "http://payment-svc:8080"},
								{Name: "DATABASE_URL", Value: "postgresql://postgres-svc:5432"},
								{Name: "GATEWAY_URL", Value: "https://api.payment-gateway.external"},
								{Name: "PLAIN", Value: "not-a-url"},
							},
						}},
					},
				},
			},
		}},
	}

	g := graph.NewGraphBuilder().Build(snap)

	src := graph.NodeID("Deployment", "demo", "api-deployment")
	found := map[string]float64{}
	for _, e := range g.Edges {
		if e.Source == src && e.EdgeType == "env_reference" {
			found[e.Target] = e.Confidence
		}
	}
	assert.Contains(t, found, "Service/demo/payment-svc", "PAYMENT_SERVICE_URL should link to payment-svc")
	assert.Contains(t, found, "Service/demo/postgres-svc", "DATABASE_URL should link to postgres-svc")
	assert.Equal(t, 0.7, found["Service/demo/payment-svc"])
	assert.NotContains(t, found, "Service/demo/api.payment-gateway.external", "external hosts are not services")
	assert.NotContains(t, found, "Service/demo/not-a-url", "non-URL values are ignored")
}
