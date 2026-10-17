// Package probe collects Kubernetes cluster resources.
package probe

import (
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// ClusterSnapshot holds all collected K8s resources from a single point in time.
type ClusterSnapshot struct {
	Namespaces      []corev1.Namespace
	Deployments     []appsv1.Deployment
	StatefulSets    []appsv1.StatefulSet
	DaemonSets      []appsv1.DaemonSet
	Pods            []corev1.Pod
	Services        []corev1.Service
	Ingresses       []networkingv1.Ingress
	ConfigMaps      []corev1.ConfigMap
	Secrets         []corev1.Secret // Data is always nil — values never stored
	HPAs            []autoscalingv2.HorizontalPodAutoscaler
	NetworkPolicies []networkingv1.NetworkPolicy
	Events          []corev1.Event
}
