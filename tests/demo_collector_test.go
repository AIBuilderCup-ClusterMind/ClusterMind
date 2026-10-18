package tests

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	demodata "github.com/clustermind/clustermind/demo"
	"github.com/clustermind/clustermind/internal/api"
	"github.com/clustermind/clustermind/internal/graph"
	"github.com/clustermind/clustermind/internal/probe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDemoMode_Parse(t *testing.T) {
	cases := []struct {
		in       string
		scenario string
		enabled  bool
		wantErr  bool
	}{
		{in: "", scenario: "", enabled: false},
		{in: "0", scenario: "", enabled: false},
		{in: "false", scenario: "", enabled: false},
		{in: "off", scenario: "", enabled: false},
		{in: "1", scenario: probe.DemoScenarioHealthy, enabled: true},
		{in: "true", scenario: probe.DemoScenarioHealthy, enabled: true},
		{in: "healthy", scenario: probe.DemoScenarioHealthy, enabled: true},
		{in: "failure", scenario: probe.DemoScenarioFailure, enabled: true},
		{in: "FAIL", scenario: probe.DemoScenarioFailure, enabled: true},
		{in: "incident", scenario: probe.DemoScenarioFailure, enabled: true},
		{in: "banana", wantErr: true},
	}
	for _, tc := range cases {
		scenario, enabled, err := probe.DemoMode(tc.in)
		if tc.wantErr {
			require.Error(t, err, "input %q", tc.in)
			continue
		}
		require.NoError(t, err, "input %q", tc.in)
		assert.Equal(t, tc.scenario, scenario, "input %q", tc.in)
		assert.Equal(t, tc.enabled, enabled, "input %q", tc.in)
	}
}

func TestDemoCollector_HealthySnapshot(t *testing.T) {
	c, err := probe.NewDemoCollector(demodata.MockClusterYAML, "demo", probe.DemoScenarioHealthy)
	require.NoError(t, err)
	require.Equal(t, "demo", c.Namespace())

	snap, err := c.Collect(context.Background())
	require.NoError(t, err)

	require.Len(t, snap.Namespaces, 1)
	assert.Len(t, snap.Services, 5)
	assert.Len(t, snap.Deployments, 5)
	assert.Len(t, snap.Ingresses, 1)
	require.Len(t, snap.Pods, 9, "one pod per desired replica: 2+3+1+2+1")

	for _, d := range snap.Deployments {
		require.NotNil(t, d.Spec.Replicas)
		assert.Equal(t, *d.Spec.Replicas, d.Status.ReadyReplicas,
			"healthy %s/%s must be fully ready", d.Namespace, d.Name)
	}
	for _, p := range snap.Pods {
		assert.Equal(t, corev1.PodRunning, p.Status.Phase, "pod %s", p.Name)
	}

	for _, ev := range snap.Events {
		assert.NotEqual(t, corev1.EventTypeWarning, ev.Type,
			"healthy scenario must not emit warning events (got %s: %s)", ev.Reason, ev.Message)
	}

	g := graph.NewGraphBuilder().Build(snap)
	require.NotEmpty(t, g.Nodes)

	incidents, changes := api.BuildTwinIncidents(snap, g)
	assert.Empty(t, incidents, "healthy demo must have no incidents")
	assert.NotEmpty(t, changes, "ScalingReplicaSet events must feed the changes timeline")
}

func TestDemoCollector_FailureIncident(t *testing.T) {
	c, err := probe.NewDemoCollector(demodata.MockClusterYAML, "demo", probe.DemoScenarioFailure)
	require.NoError(t, err)

	snap, err := c.Collect(context.Background())
	require.NoError(t, err)

	var payment *appsv1.Deployment
	for i := range snap.Deployments {
		if snap.Deployments[i].Name == "payment-deployment" {
			payment = &snap.Deployments[i]
		}
	}
	require.NotNil(t, payment, "payment-deployment must be in the manifest")
	assert.EqualValues(t, 0, payment.Status.ReadyReplicas)
	assert.EqualValues(t, 1, payment.Status.UnavailableReplicas)
	assert.Equal(t, "nginx:does-not-exist",
		payment.Spec.Template.Spec.Containers[0].Image, "failure injects a broken image")

	warnings := 0
	for _, ev := range snap.Events {
		if ev.Type == corev1.EventTypeWarning {
			warnings++
			assert.Equal(t, "payment-deployment", ev.InvolvedObject.Name)
			assert.Equal(t, "Failed", ev.Reason)
		}
	}
	assert.Equal(t, 3, warnings, "three ImagePullBackOff warning events")

	g := graph.NewGraphBuilder().Build(snap)
	require.NotEmpty(t, g.Nodes)

	incidents, changes := api.BuildTwinIncidents(snap, g)
	require.NotEmpty(t, incidents, "failure demo must derive an incident")

	var paymentIncident *api.TwinIncident
	for i := range incidents {
		if incidents[i].PrimaryResourceID == "Deployment/demo/payment-deployment" {
			paymentIncident = &incidents[i]
		}
	}
	require.NotNil(t, paymentIncident, "incident must target the broken deployment")
	assert.Equal(t, "open", paymentIncident.Status)
	assert.Equal(t, "sev1", paymentIncident.Severity, "0/1 ready → unhealthy → sev1")
	assert.GreaterOrEqual(t, len(paymentIncident.Timeline), 3,
		"timeline must carry the warning events as evidence")
	assert.NotEmpty(t, changes)
}

func TestDemoCollector_NamespaceFilter(t *testing.T) {
	all, err := probe.NewDemoCollector(demodata.MockClusterYAML, "", probe.DemoScenarioHealthy)
	require.NoError(t, err)
	allSnap, err := all.Collect(context.Background())
	require.NoError(t, err)
	assert.Len(t, allSnap.Services, 5, "empty scope = all namespaces")

	scoped, err := probe.NewDemoCollector(demodata.MockClusterYAML, "other", probe.DemoScenarioHealthy)
	require.NoError(t, err)
	scopedSnap, err := scoped.Collect(context.Background())
	require.NoError(t, err)
	assert.Empty(t, scopedSnap.Services, "unknown namespace yields empty snapshot")
	assert.Empty(t, scopedSnap.Deployments)
	assert.Empty(t, scopedSnap.Pods)
}

func TestDemoCollector_Validation(t *testing.T) {
	_, err := probe.NewDemoCollector(nil, "", probe.DemoScenarioHealthy)
	require.Error(t, err)

	_, err = probe.NewDemoCollector([]byte("not: valid: k8s: ["), "", probe.DemoScenarioHealthy)
	require.Error(t, err, "malformed manifest must fail at construction")

	_, err = probe.NewDemoCollector(demodata.MockClusterYAML, "", "zombie")
	require.Error(t, err, "unknown scenario must be rejected")
}
