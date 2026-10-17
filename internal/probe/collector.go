package probe

import (
	"context"
	"fmt"
	"log/slog"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ClusterCollector gathers K8s resources from a cluster (read-only).
type ClusterCollector struct {
	clientset kubernetes.Interface
	namespace string // empty = all namespaces
}

// NewClusterCollector creates a ClusterCollector.
// namespace="" collects from all namespaces.
func NewClusterCollector(clientset kubernetes.Interface, namespace string) *ClusterCollector {
	return &ClusterCollector{clientset: clientset, namespace: namespace}
}

// Namespace returns the namespace filter (empty string = all namespaces).
func (c *ClusterCollector) Namespace() string {
	return c.namespace
}

// Collect fetches all supported resource types and returns a ClusterSnapshot.
func (c *ClusterCollector) Collect(ctx context.Context) (*ClusterSnapshot, error) {
	snap := &ClusterSnapshot{}
	ns := c.namespace

	// Namespaces (cluster-scoped)
	nsList, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("collect Namespace: %w", err)
	}
	snap.Namespaces = nsList.Items
	slog.Info("collected", "kind", "Namespace", "count", len(snap.Namespaces))

	// Deployments
	deplList, err := c.clientset.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("collect Deployment: %w", err)
	}
	snap.Deployments = deplList.Items
	slog.Info("collected", "kind", "Deployment", "count", len(snap.Deployments))

	// StatefulSets
	ssList, err := c.clientset.AppsV1().StatefulSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("collect StatefulSet: %w", err)
	}
	snap.StatefulSets = ssList.Items

	// DaemonSets
	dsList, err := c.clientset.AppsV1().DaemonSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("collect DaemonSet: %w", err)
	}
	snap.DaemonSets = dsList.Items

	// Pods
	podList, err := c.clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("collect Pod: %w", err)
	}
	snap.Pods = podList.Items
	slog.Info("collected", "kind", "Pod", "count", len(snap.Pods))

	// Services
	svcList, err := c.clientset.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("collect Service: %w", err)
	}
	snap.Services = svcList.Items
	slog.Info("collected", "kind", "Service", "count", len(snap.Services))

	// Ingresses
	ingList, err := c.clientset.NetworkingV1().Ingresses(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("collect Ingress: %w", err)
	}
	snap.Ingresses = ingList.Items

	// ConfigMaps (names only — no data needed for graph)
	cmList, err := c.clientset.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("collect ConfigMap: %w", err)
	}
	snap.ConfigMaps = cmList.Items

	// Secrets — Data is cleared to avoid leaking values
	secList, err := c.clientset.CoreV1().Secrets(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("collect Secret: %w", err)
	}
	for i := range secList.Items {
		secList.Items[i].Data = nil
		secList.Items[i].StringData = nil
	}
	snap.Secrets = secList.Items

	// HPAs
	hpaList, err := c.clientset.AutoscalingV2().HorizontalPodAutoscalers(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("collect HPA: %w", err)
	}
	snap.HPAs = hpaList.Items

	// NetworkPolicies
	npList, err := c.clientset.NetworkingV1().NetworkPolicies(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("collect NetworkPolicy: %w", err)
	}
	snap.NetworkPolicies = npList.Items

	// Events (recent, both Warning and Normal — feeds incident timelines)
	evList, err := c.clientset.CoreV1().Events(ns).List(ctx, metav1.ListOptions{Limit: 500})
	if err != nil {
		return nil, fmt.Errorf("collect Event: %w", err)
	}
	snap.Events = evList.Items

	_ = autoscalingv2.SchemeGroupVersion // suppress unused import
	slog.Info("snapshot complete",
		"deployments", len(snap.Deployments),
		"services", len(snap.Services),
		"pods", len(snap.Pods),
		"ingresses", len(snap.Ingresses),
		"events", len(snap.Events),
	)
	return snap, nil
}
