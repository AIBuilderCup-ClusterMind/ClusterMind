package api

import (
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/clustermind/clustermind/internal/graph"
	"github.com/clustermind/clustermind/internal/probe"
	"github.com/clustermind/clustermind/tests/fixtures"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTwinIncidents_SyntheticAlertForDegradedResources(t *testing.T) {
	snap := fixtures.MockSnapshot()
	g := graph.NewGraphBuilder().Build(snap)

	incidents, _ := BuildTwinIncidents(snap, g)

	require.NotEmpty(t, incidents, "degraded resources should produce incidents")
	for _, inc := range incidents {
		assert.Equal(t, "open", inc.Status)
		assert.Contains(t, []string{"sev1", "sev2"}, inc.Severity)
		require.Len(t, inc.Timeline, 1)
		assert.Equal(t, "alert", inc.Timeline[0].Source)
		assert.Contains(t, inc.Timeline[0].Title, "is ")
	}
}

func TestBuildTwinIncidents_WarningEventMapsReplicaSetToDeployment(t *testing.T) {
	now := time.Now()
	snap := &probe.ClusterSnapshot{
		Deployments: []appsv1.Deployment{{
			ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "prod"},
			Spec: appsv1.DeploymentSpec{
				Replicas: ptrInt32(1),
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "web"}},
				},
			},
		}},
		Events: []corev1.Event{{
			ObjectMeta:     metav1.ObjectMeta{Name: "web-7d9f84b6c5.abc", Namespace: "prod"},
			Type:           corev1.EventTypeWarning,
			Reason:         "BackOff",
			Message:        "Back-off restarting failed container",
			InvolvedObject: corev1.ObjectReference{Kind: "ReplicaSet", Namespace: "prod", Name: "web-7d9f84b6c5"},
			Source:         corev1.EventSource{Component: "kubelet"},
			LastTimestamp:  metav1.NewTime(now.Add(-5 * time.Minute)),
		}},
	}
	g := graph.NewGraphBuilder().Build(snap)

	incidents, changes := BuildTwinIncidents(snap, g)

	require.Len(t, incidents, 1)
	inc := incidents[0]
	assert.Equal(t, "inc-deployment-prod-web", inc.ID)
	assert.Equal(t, "Deployment/prod/web", inc.PrimaryResourceID)
	require.Len(t, inc.Timeline, 1)
	assert.Equal(t, "ev-inc-deployment-prod-web-0", inc.Timeline[0].ID)
	assert.Equal(t, "events", inc.Timeline[0].Source)
	assert.Equal(t, "BackOff", inc.Timeline[0].Title)
	assert.Empty(t, changes, "warning events are not changes")
}

func TestBuildChanges_DeploymentRollout(t *testing.T) {
	now := time.Now()
	snap := &probe.ClusterSnapshot{
		Deployments: []appsv1.Deployment{{
			ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: "prod"},
			Spec: appsv1.DeploymentSpec{
				Replicas: ptrInt32(1),
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "api"}},
				},
			},
		}},
		Events: []corev1.Event{{
			ObjectMeta:     metav1.ObjectMeta{Name: "api.1", Namespace: "prod"},
			Type:           corev1.EventTypeNormal,
			Reason:         "ScalingReplicaSet",
			Message:        "Scaled up replica set api-6f7b8c to 1",
			InvolvedObject: corev1.ObjectReference{Kind: "Deployment", Namespace: "prod", Name: "api"},
			Source:         corev1.EventSource{Component: "deployment-controller"},
			LastTimestamp:  metav1.NewTime(now.Add(-10 * time.Minute)),
		}},
	}
	g := graph.NewGraphBuilder().Build(snap)

	_, changes := BuildTwinIncidents(snap, g)

	require.Len(t, changes, 1)
	assert.Contains(t, changes[0].Description, "ScalingReplicaSet")
	assert.Equal(t, "deployment-controller", changes[0].Actor)
	assert.Equal(t, "Deployment/prod/api", changes[0].ResourceID)
}

func TestBuildTwinIncidents_PodWarningEventMapsToDeployment(t *testing.T) {
	now := time.Now()
	snap := &probe.ClusterSnapshot{
		Deployments: []appsv1.Deployment{{
			ObjectMeta: metav1.ObjectMeta{Name: "checkout", Namespace: "demo"},
			Spec: appsv1.DeploymentSpec{
				Replicas: ptrInt32(1),
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "checkout"}},
				},
			},
		}},
		Events: []corev1.Event{{
			ObjectMeta:     metav1.ObjectMeta{Name: "checkout-7d9f84b6c5-x9y8z.abc", Namespace: "demo"},
			Type:           corev1.EventTypeWarning,
			Reason:         "BackOff",
			Message:        "Back-off restarting failed container",
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Namespace: "demo", Name: "checkout-7d9f84b6c5-x9y8z"},
			Source:         corev1.EventSource{Component: "kubelet"},
			LastTimestamp:  metav1.NewTime(now.Add(-2 * time.Minute)),
		}},
	}
	g := graph.NewGraphBuilder().Build(snap)

	incidents, _ := BuildTwinIncidents(snap, g)

	require.Len(t, incidents, 1)
	assert.Equal(t, "inc-deployment-demo-checkout", incidents[0].ID)
	require.Len(t, incidents[0].Timeline, 1)
	assert.Equal(t, "events", incidents[0].Timeline[0].Source)
	assert.Equal(t, "BackOff", incidents[0].Timeline[0].Title)
}

func TestBuildTwinIncidents_HealthyResourceWithWarningEvents(t *testing.T) {
	now := time.Now()
	ready, desired := int32(1), int32(1)
	snap := &probe.ClusterSnapshot{
		Deployments: []appsv1.Deployment{{
			ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "demo"},
			Spec: appsv1.DeploymentSpec{
				Replicas: &desired,
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "web"}},
				},
			},
			Status: appsv1.DeploymentStatus{ReadyReplicas: ready, Replicas: desired, UpdatedReplicas: desired},
		}},
		Events: []corev1.Event{{
			ObjectMeta:     metav1.ObjectMeta{Name: "web-x9y8z.1", Namespace: "demo"},
			Type:           corev1.EventTypeWarning,
			Reason:         "Failed",
			Message:        "Error: ImagePullBackOff",
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Namespace: "demo", Name: "web-7d9f84b6c5-x9y8z"},
			Source:         corev1.EventSource{Component: "kubelet"},
			LastTimestamp:  metav1.NewTime(now.Add(-1 * time.Minute)),
		}},
	}
	g := graph.NewGraphBuilder().Build(snap)

	incidents, _ := BuildTwinIncidents(snap, g)

	require.Len(t, incidents, 1)
	inc := incidents[0]
	assert.Equal(t, "sev3", inc.Severity, "healthy resource with warnings → sev3")
	assert.Contains(t, inc.Title, "— Failed")
	require.Len(t, inc.Timeline, 1)
	assert.Equal(t, "events", inc.Timeline[0].Source)
}

func TestTrimPodTemplateHash(t *testing.T) {
	cases := map[string]string{
		"web-7d9f84b6c5":       "web",
		"web-abc":              "web-abc",  // suffix too short
		"web":                  "web",      // no dash
		"-leading":             "-leading", // empty prefix handled
		"api-v2.1.0-x9y8z7w6q": "api-v2.1.0",
	}
	for input, want := range cases {
		assert.Equal(t, want, trimPodTemplateHash(input), "input %q", input)
	}
}

func ptrInt32(v int32) *int32 { return &v }

func TestBuildTwinIncidents_StaleWarningOnHealthyResourceIgnored(t *testing.T) {
	now := time.Now()
	ready, desired := int32(1), int32(1)
	snap := &probe.ClusterSnapshot{
		Deployments: []appsv1.Deployment{{
			ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: "demo"},
			Spec: appsv1.DeploymentSpec{
				Replicas: &desired,
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "web"}},
				},
			},
			Status: appsv1.DeploymentStatus{ReadyReplicas: ready, Replicas: desired, UpdatedReplicas: desired},
		}},
		Events: []corev1.Event{{
			ObjectMeta:     metav1.ObjectMeta{Name: "web-old.1", Namespace: "demo"},
			Type:           corev1.EventTypeWarning,
			Reason:         "Failed",
			Message:        "Error: ImagePullBackOff",
			InvolvedObject: corev1.ObjectReference{Kind: "Pod", Namespace: "demo", Name: "web-7d9f84b6c5-x9y8z"},
			LastTimestamp:  metav1.NewTime(now.Add(-30 * time.Minute)),
		}},
	}
	g := graph.NewGraphBuilder().Build(snap)

	incidents, _ := BuildTwinIncidents(snap, g)

	assert.Empty(t, incidents, "stale warnings on a recovered resource must not stay open")
}

func TestBuildTwinIncidents_IngressWithoutLoadBalancerIP_NoSpuriousIncident(t *testing.T) {
	pathType := networkingv1.PathTypePrefix
	snap := &probe.ClusterSnapshot{
		Deployments: []appsv1.Deployment{{
			ObjectMeta: metav1.ObjectMeta{Name: "frontend-deployment", Namespace: "demo"},
			Spec: appsv1.DeploymentSpec{
				Replicas: ptrInt32(2),
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "frontend"}},
				},
			},
			Status: appsv1.DeploymentStatus{ReadyReplicas: 2, Replicas: 2, UpdatedReplicas: 2, AvailableReplicas: 2},
		}},
		Services: []corev1.Service{{
			ObjectMeta: metav1.ObjectMeta{Name: "frontend-svc", Namespace: "demo"},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{"app": "frontend"},
				Type:     corev1.ServiceTypeClusterIP,
			},
		}},
		Ingresses: []networkingv1.Ingress{{
			ObjectMeta: metav1.ObjectMeta{Name: "ingress-nginx", Namespace: "demo"},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{{
					Host: "demo.local",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{{
								Path:     "/",
								PathType: &pathType,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "frontend-svc",
										Port: networkingv1.ServiceBackendPort{Number: 80},
									},
								},
							}},
						},
					},
				}},
			},
			Status: networkingv1.IngressStatus{
				LoadBalancer: networkingv1.IngressLoadBalancerStatus{
					Ingress: []networkingv1.IngressLoadBalancerIngress{}, // No LB IP in local kind cluster
				},
			},
		}},
	}
	g := graph.NewGraphBuilder().Build(snap)

	incidents, changes := BuildTwinIncidents(snap, g)

	assert.Empty(t, incidents, "healthy ingress with rules but without cloud LB IP must not produce an incident")
	assert.Empty(t, changes)
}

func TestBuildTwinIncidents_IngressWithGenuineWarningEvent(t *testing.T) {
	now := time.Now()
	pathType := networkingv1.PathTypePrefix
	snap := &probe.ClusterSnapshot{
		Ingresses: []networkingv1.Ingress{{
			ObjectMeta: metav1.ObjectMeta{Name: "ingress-nginx", Namespace: "demo"},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{{
					Host: "demo.local",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{{
								Path:     "/",
								PathType: &pathType,
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: "missing-svc",
										Port: networkingv1.ServiceBackendPort{Number: 80},
									},
								},
							}},
						},
					},
				}},
			},
		}},
		Events: []corev1.Event{{
			ObjectMeta:     metav1.ObjectMeta{Name: "ing.warn.1", Namespace: "demo"},
			Type:           corev1.EventTypeWarning,
			Reason:         "BackendNotFound",
			Message:        "Service missing-svc not found in namespace demo",
			InvolvedObject: corev1.ObjectReference{Kind: "Ingress", Namespace: "demo", Name: "ingress-nginx"},
			Source:         corev1.EventSource{Component: "ingress-controller"},
			LastTimestamp:  metav1.NewTime(now.Add(-2 * time.Minute)),
		}},
	}
	g := graph.NewGraphBuilder().Build(snap)

	incidents, _ := BuildTwinIncidents(snap, g)

	require.Len(t, incidents, 1, "genuine ingress warning events must produce an incident")
	assert.Equal(t, "inc-ingress-demo-ingress-nginx", incidents[0].ID)
	assert.Equal(t, "sev3", incidents[0].Severity)
	assert.Contains(t, incidents[0].Title, "BackendNotFound")
	require.Len(t, incidents[0].Timeline, 1)
	assert.Equal(t, "events", incidents[0].Timeline[0].Source)
	assert.Equal(t, "BackendNotFound", incidents[0].Timeline[0].Title)
}

func TestBuildTwinIncidents_StableTimestampAcrossPolls(t *testing.T) {
	fixedCreation := time.Date(2026, 10, 18, 1, 0, 0, 0, time.UTC)
	snap := &probe.ClusterSnapshot{
		Deployments: []appsv1.Deployment{{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "failed-worker",
				Namespace:         "demo",
				CreationTimestamp: metav1.NewTime(fixedCreation),
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: ptrInt32(2),
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "worker"}},
				},
			},
			Status: appsv1.DeploymentStatus{ReadyReplicas: 0, Replicas: 2, UpdatedReplicas: 2},
		}},
	}
	g := graph.NewGraphBuilder().Build(snap)

	// Poll 1
	incidentsPoll1, _ := BuildTwinIncidents(snap, g)
	require.Len(t, incidentsPoll1, 1)

	// Simulate passage of time between polling intervals
	time.Sleep(10 * time.Millisecond)

	// Poll 2
	incidentsPoll2, _ := BuildTwinIncidents(snap, g)
	require.Len(t, incidentsPoll2, 1)

	// Both polls must yield exact same stable StartedAt and timeline evidence time matching CreationTimestamp
	assert.Equal(t, incidentsPoll1[0].StartedAt, incidentsPoll2[0].StartedAt, "incident startedAt must not shift between polls")
	assert.Equal(t, fixedCreation.Format(time.RFC3339), incidentsPoll1[0].StartedAt)
	assert.Equal(t, incidentsPoll1[0].Timeline[0].Time, incidentsPoll2[0].Timeline[0].Time, "timeline evidence time must remain stable")
}
