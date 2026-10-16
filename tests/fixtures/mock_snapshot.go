// Package fixtures provides test helpers for ClusterMind unit tests.
package fixtures

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/clustermind/clustermind/internal/probe"
)

// MockSnapshot returns a demo ClusterSnapshot modelling the payment-svc topology.
func MockSnapshot() *probe.ClusterSnapshot {
	ns := "demo"
	return &probe.ClusterSnapshot{
		Services: []corev1.Service{
			makeService(ns, "frontend-svc", map[string]string{"app": "frontend"}),
			makeService(ns, "api-svc", map[string]string{"app": "api"}),
			makeService(ns, "payment-svc", map[string]string{"app": "payment"}),
			makeService(ns, "auth-svc", map[string]string{"app": "auth"}),
			makeService(ns, "postgres-svc", map[string]string{"app": "postgres"}),
		},
		Deployments: []appsv1.Deployment{
			makeDeployment(ns, "frontend-deployment", map[string]string{"app": "frontend"}),
			makeDeployment(ns, "api-deployment", map[string]string{"app": "api"}),
			makeDeployment(ns, "payment-deployment", map[string]string{"app": "payment"}),
			makeDeployment(ns, "auth-deployment", map[string]string{"app": "auth"}),
		},
		Ingresses: []networkingv1.Ingress{
			makeIngress(ns, "ingress-nginx", "frontend-svc"),
		},
	}
}

func makeService(ns, name string, selector map[string]string) corev1.Service {
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec:       corev1.ServiceSpec{Selector: selector},
	}
}

func makeDeployment(ns, name string, labels map[string]string) appsv1.Deployment {
	replicas := int32(1)
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
			},
		},
	}
}

func makeIngress(ns, name, svcName string) networkingv1.Ingress {
	pathType := networkingv1.PathTypePrefix
	return networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "demo.local",
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: svcName,
											Port: networkingv1.ServiceBackendPort{Number: 80},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
