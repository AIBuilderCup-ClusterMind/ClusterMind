package probe

import (
	"context"
	"fmt"
	"log/slog"
	"time"

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
// Essential resources (Deployments, Pods, Services) return an error on failure.
// Optional resources log a warning and allow collection to proceed.
// Secrets are never collected to protect sensitive data and prevent RBAC failures.
func (c *ClusterCollector) Collect(ctx context.Context) (*ClusterSnapshot, error) {
	snap := &ClusterSnapshot{
		CapturedAt: time.Now().UTC(),
	}
	ns := c.namespace

	// Namespaces (cluster-scoped, optional)
	nsList, err := c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		slog.Warn("optional resource collection skipped", "kind", "Namespace", "error", err)
	} else {
		snap.Namespaces = nsList.Items
		slog.Info("collected", "kind", "Namespace", "count", len(snap.Namespaces))
	}

	// Deployments (essential)
	deplList, err := c.clientset.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("collect Deployment: %w", err)
	}
	snap.Deployments = deplList.Items
	slog.Info("collected", "kind", "Deployment", "count", len(snap.Deployments))

	// StatefulSets (optional)
	if ssList, err := c.clientset.AppsV1().StatefulSets(ns).List(ctx, metav1.ListOptions{}); err != nil {
		slog.Warn("optional resource collection skipped", "kind", "StatefulSet", "namespace", ns, "error", err)
	} else {
		snap.StatefulSets = ssList.Items
	}

	// DaemonSets (optional)
	if dsList, err := c.clientset.AppsV1().DaemonSets(ns).List(ctx, metav1.ListOptions{}); err != nil {
		slog.Warn("optional resource collection skipped", "kind", "DaemonSet", "namespace", ns, "error", err)
	} else {
		snap.DaemonSets = dsList.Items
	}

	// Pods (essential)
	podList, err := c.clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("collect Pod: %w", err)
	}
	snap.Pods = podList.Items
	slog.Info("collected", "kind", "Pod", "count", len(snap.Pods))

	// Services (essential)
	svcList, err := c.clientset.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("collect Service: %w", err)
	}
	snap.Services = svcList.Items
	slog.Info("collected", "kind", "Service", "count", len(snap.Services))

	// Ingresses (optional)
	if ingList, err := c.clientset.NetworkingV1().Ingresses(ns).List(ctx, metav1.ListOptions{}); err != nil {
		slog.Warn("optional resource collection skipped", "kind", "Ingress", "namespace", ns, "error", err)
	} else {
		snap.Ingresses = ingList.Items
	}

	// ConfigMaps (optional — names only, no data needed for graph)
	if cmList, err := c.clientset.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{}); err != nil {
		slog.Warn("optional resource collection skipped", "kind", "ConfigMap", "namespace", ns, "error", err)
	} else {
		snap.ConfigMaps = cmList.Items
	}

	// Secrets: Deliberately omitted. Secrets are not collected for security and RBAC reasons.

	// HPAs (optional)
	if hpaList, err := c.clientset.AutoscalingV2().HorizontalPodAutoscalers(ns).List(ctx, metav1.ListOptions{}); err != nil {
		slog.Warn("optional resource collection skipped", "kind", "HPA", "namespace", ns, "error", err)
	} else {
		snap.HPAs = hpaList.Items
	}

	// NetworkPolicies (optional)
	if npList, err := c.clientset.NetworkingV1().NetworkPolicies(ns).List(ctx, metav1.ListOptions{}); err != nil {
		slog.Warn("optional resource collection skipped", "kind", "NetworkPolicy", "namespace", ns, "error", err)
	} else {
		snap.NetworkPolicies = npList.Items
	}

	// Events (optional — recent timeline context)
	if evList, err := c.clientset.CoreV1().Events(ns).List(ctx, metav1.ListOptions{Limit: 500}); err != nil {
		slog.Warn("optional resource collection skipped", "kind", "Event", "namespace", ns, "error", err)
	} else {
		snap.Events = evList.Items
	}

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
