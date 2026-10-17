package probe

import (
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
)

func TestExtractIngressStatus_LocalKindWithRules(t *testing.T) {
	now := time.Now()
	created := now.Add(-10 * time.Minute)
	pathType := networkingv1.PathTypePrefix
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "ingress-nginx",
			Namespace:         "demo",
			CreationTimestamp: metav1.NewTime(created),
		},
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
				Ingress: []networkingv1.IngressLoadBalancerIngress{}, // Empty in kind
			},
		},
	}

	rs := extractIngressStatus(ing, now)

	assert.Equal(t, "Ingress", rs.Kind)
	assert.Equal(t, "ingress-nginx", rs.Name)
	assert.Equal(t, StatusHealthy, rs.Phase, "ingress with rules should be healthy in local cluster")
	assert.Equal(t, "RoutingConfigured", rs.Reason)
	assert.Contains(t, rs.Message, "1 rule(s) configured")
	assert.Equal(t, created, rs.CreatedAt)
}

func TestExtractIngressStatus_LoadBalancerReady(t *testing.T) {
	now := time.Now()
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "cloud-ing", Namespace: "prod"},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{Host: "app.example.com"}},
		},
		Status: networkingv1.IngressStatus{
			LoadBalancer: networkingv1.IngressLoadBalancerStatus{
				Ingress: []networkingv1.IngressLoadBalancerIngress{{IP: "34.120.1.2"}},
			},
		},
	}

	rs := extractIngressStatus(ing, now)

	assert.Equal(t, StatusHealthy, rs.Phase)
	assert.Equal(t, "Ready", rs.Reason)
}

func TestExtractIngressStatus_NoRules(t *testing.T) {
	now := time.Now()
	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: "empty-ing", Namespace: "prod"},
	}

	rs := extractIngressStatus(ing, now)

	assert.Equal(t, StatusPending, rs.Phase)
	assert.Equal(t, "NoRules", rs.Reason)
}

func TestExtractResourceStatuses_CreatedAtPopulated(t *testing.T) {
	now := time.Now()
	created := now.Add(-1 * time.Hour)
	snap := &ClusterSnapshot{
		Deployments: []appsv1.Deployment{{
			ObjectMeta: metav1.ObjectMeta{Name: "d1", Namespace: "default", CreationTimestamp: metav1.NewTime(created)},
		}},
		Services: []corev1.Service{{
			ObjectMeta: metav1.ObjectMeta{Name: "s1", Namespace: "default", CreationTimestamp: metav1.NewTime(created)},
		}},
	}

	statuses := ExtractResourceStatuses(snap)
	for _, rs := range statuses {
		assert.Equal(t, created, rs.CreatedAt, "CreatedAt should be populated on resource status")
	}
}
