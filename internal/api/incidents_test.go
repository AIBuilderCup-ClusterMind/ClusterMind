package api

import (
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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
