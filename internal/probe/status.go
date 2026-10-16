package probe

import (
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
)

// StatusPhase is a normalized health status for a single K8s resource.
type StatusPhase string

// Status phase constants.
const (
	StatusHealthy   StatusPhase = "Healthy"
	StatusDegraded  StatusPhase = "Degraded"
	StatusWarning   StatusPhase = "Warning"
	StatusPending   StatusPhase = "Pending"
	StatusCompleted StatusPhase = "Completed"
	StatusUnknown   StatusPhase = "Unknown"
)

// ResourceStatus is a normalized, kind-agnostic view of a single resource's health.
type ResourceStatus struct {
	Kind      string      `json:"kind"`
	Name      string      `json:"name"`
	Namespace string      `json:"namespace"`
	Phase     StatusPhase `json:"phase"`
	Ready     string      `json:"ready"`   // e.g. "3/3", "0/1", "—"
	Reason    string      `json:"reason"`  // short reason e.g. "CrashLoopBackOff", "Available"
	Message   string      `json:"message"` // human-readable detail
	Age       string      `json:"age"`     // human-readable age e.g. "2d3h"
}

// NamespaceResourceStats aggregates resource counts and phase breakdown for one namespace.
type NamespaceResourceStats struct {
	Name          string              `json:"name"`
	Phase         StatusPhase         `json:"phase"`         // overall namespace phase
	ResourceCount int                 `json:"resourceCount"` // total non-namespace resources
	Counts        map[string]int      `json:"counts"`        // kind → count
	PhaseCounts   map[StatusPhase]int `json:"phaseCounts"`   // phase → count
	Age           string              `json:"age"`
}

// ExtractResourceStatuses returns a flat list of ResourceStatus for every
// workload/config resource in the snapshot (excluding Namespace objects themselves).
func ExtractResourceStatuses(snap *ClusterSnapshot) []ResourceStatus {
	now := time.Now()
	var out []ResourceStatus

	for i := range snap.Deployments {
		out = append(out, extractDeploymentStatus(&snap.Deployments[i], now))
	}
	for i := range snap.StatefulSets {
		out = append(out, extractStatefulSetStatus(&snap.StatefulSets[i], now))
	}
	for i := range snap.DaemonSets {
		out = append(out, extractDaemonSetStatus(&snap.DaemonSets[i], now))
	}
	for i := range snap.Pods {
		out = append(out, extractPodStatus(&snap.Pods[i], now))
	}
	for i := range snap.Services {
		out = append(out, extractServiceStatus(&snap.Services[i], now))
	}
	for i := range snap.Ingresses {
		out = append(out, extractIngressStatus(&snap.Ingresses[i], now))
	}
	for i := range snap.ConfigMaps {
		cm := &snap.ConfigMaps[i]
		out = append(out, staticStatus("ConfigMap", cm.Name, cm.Namespace,
			cm.CreationTimestamp.Time, now, fmt.Sprintf("%d keys", len(cm.Data))))
	}
	for i := range snap.Secrets {
		sec := &snap.Secrets[i]
		out = append(out, staticStatus("Secret", sec.Name, sec.Namespace,
			sec.CreationTimestamp.Time, now, "values not stored"))
	}
	for i := range snap.HPAs {
		out = append(out, extractHPAStatus(&snap.HPAs[i], now))
	}
	for i := range snap.NetworkPolicies {
		np := &snap.NetworkPolicies[i]
		out = append(out, staticStatus("NetworkPolicy", np.Name, np.Namespace,
			np.CreationTimestamp.Time, now, ""))
	}

	return out
}

// BuildNamespaceStats returns per-namespace aggregated stats derived from
// the resource statuses extracted from the snapshot.
func BuildNamespaceStats(snap *ClusterSnapshot, statuses []ResourceStatus) []NamespaceResourceStats {
	stats := map[string]*NamespaceResourceStats{}
	now := time.Now()

	// Seed with every Namespace object (even if empty).
	for i := range snap.Namespaces {
		ns := &snap.Namespaces[i]
		phase := StatusUnknown
		switch ns.Status.Phase {
		case corev1.NamespaceActive:
			phase = StatusHealthy
		case corev1.NamespaceTerminating:
			phase = StatusWarning
		}
		stats[ns.Name] = &NamespaceResourceStats{
			Name:        ns.Name,
			Phase:       phase,
			Counts:      map[string]int{},
			PhaseCounts: map[StatusPhase]int{},
			Age:         formatAge(ns.CreationTimestamp.Time, now),
		}
	}

	// Aggregate per-resource statuses.
	for _, rs := range statuses {
		s, ok := stats[rs.Namespace]
		if !ok {
			s = &NamespaceResourceStats{
				Name:        rs.Namespace,
				Phase:       StatusUnknown,
				Counts:      map[string]int{},
				PhaseCounts: map[StatusPhase]int{},
			}
			stats[rs.Namespace] = s
		}
		s.ResourceCount++
		s.Counts[rs.Kind]++
		s.PhaseCounts[rs.Phase]++
		s.Phase = worstPhase(s.Phase, rs.Phase)
	}

	// Deterministic output order by namespace name (insertion sort, small N).
	seen := make([]string, 0, len(stats))
	for name := range stats {
		seen = append(seen, name)
	}
	for i := 1; i < len(seen); i++ {
		for j := i; j > 0 && seen[j] < seen[j-1]; j-- {
			seen[j], seen[j-1] = seen[j-1], seen[j]
		}
	}

	out := make([]NamespaceResourceStats, 0, len(seen))
	for _, name := range seen {
		out = append(out, *stats[name])
	}
	return out
}

// FilterStatuses returns only statuses whose Namespace matches ns and,
// if kindFilter is non-empty, whose Kind matches kindFilter (case-insensitive).
func FilterStatuses(statuses []ResourceStatus, ns, kindFilter string) []ResourceStatus {
	var out []ResourceStatus
	for _, rs := range statuses {
		if ns != "" && rs.Namespace != ns {
			continue
		}
		if kindFilter != "" && !strings.EqualFold(rs.Kind, kindFilter) {
			continue
		}
		out = append(out, rs)
	}
	return out
}

// ── per-kind status extractors ──────────────────────────────────────────────

func extractDeploymentStatus(d *appsv1.Deployment, now time.Time) ResourceStatus {
	desired := int32(1)
	if d.Spec.Replicas != nil {
		desired = *d.Spec.Replicas
	}
	ready := d.Status.ReadyReplicas
	available := d.Status.AvailableReplicas
	updated := d.Status.UpdatedReplicas

	rs := ResourceStatus{
		Kind:      "Deployment",
		Name:      d.Name,
		Namespace: d.Namespace,
		Ready:     fmt.Sprintf("%d/%d", ready, desired),
		Age:       formatAge(d.CreationTimestamp.Time, now),
	}

	switch {
	case desired == 0:
		rs.Phase = StatusCompleted
		rs.Reason = "ScaledToZero"
		rs.Message = "Replicas scaled to 0"
	case ready == 0 && desired > 0:
		rs.Phase = StatusDegraded
		rs.Reason = "MinimumReplicasUnavailable"
		rs.Message = fmt.Sprintf("0 of %d desired replicas ready", desired)
	case ready < desired:
		rs.Phase = StatusWarning
		rs.Reason = "PartialAvailability"
		rs.Message = fmt.Sprintf("%d of %d desired replicas ready", ready, desired)
	case updated < desired:
		rs.Phase = StatusWarning
		rs.Reason = "RollingUpdate"
		rs.Message = fmt.Sprintf("Rolling update in progress: %d/%d updated", updated, desired)
	default:
		rs.Phase = StatusHealthy
		rs.Reason = "Available"
		rs.Message = fmt.Sprintf("All %d replicas available", available)
	}
	return rs
}

func extractStatefulSetStatus(s *appsv1.StatefulSet, now time.Time) ResourceStatus {
	desired := int32(1)
	if s.Spec.Replicas != nil {
		desired = *s.Spec.Replicas
	}
	ready := s.Status.ReadyReplicas
	current := s.Status.CurrentReplicas
	updated := s.Status.UpdatedReplicas

	rs := ResourceStatus{
		Kind:      "StatefulSet",
		Name:      s.Name,
		Namespace: s.Namespace,
		Ready:     fmt.Sprintf("%d/%d", ready, desired),
		Age:       formatAge(s.CreationTimestamp.Time, now),
	}

	switch {
	case desired == 0:
		rs.Phase = StatusCompleted
		rs.Reason = "ScaledToZero"
		rs.Message = "Replicas scaled to 0"
	case ready == 0 && desired > 0:
		rs.Phase = StatusDegraded
		rs.Reason = "MinimumReplicasUnavailable"
		rs.Message = fmt.Sprintf("0 of %d desired replicas ready", desired)
	case ready < desired:
		rs.Phase = StatusWarning
		rs.Reason = "PartialAvailability"
		rs.Message = fmt.Sprintf("%d of %d desired replicas ready", ready, desired)
	case updated < desired:
		rs.Phase = StatusWarning
		rs.Reason = "RollingUpdate"
		rs.Message = fmt.Sprintf("Rolling update in progress: %d/%d updated", updated, desired)
	default:
		rs.Phase = StatusHealthy
		rs.Reason = "Available"
		rs.Message = fmt.Sprintf("All %d replicas ready (current: %d)", ready, current)
	}
	return rs
}

func extractDaemonSetStatus(ds *appsv1.DaemonSet, now time.Time) ResourceStatus {
	desired := ds.Status.DesiredNumberScheduled
	ready := ds.Status.NumberReady
	updated := ds.Status.UpdatedNumberScheduled

	rs := ResourceStatus{
		Kind:      "DaemonSet",
		Name:      ds.Name,
		Namespace: ds.Namespace,
		Ready:     fmt.Sprintf("%d/%d", ready, desired),
		Age:       formatAge(ds.CreationTimestamp.Time, now),
	}

	switch {
	case desired == 0:
		rs.Phase = StatusCompleted
		rs.Reason = "NoNodes"
		rs.Message = "No schedulable nodes match the node selector"
	case ready == 0:
		rs.Phase = StatusDegraded
		rs.Reason = "MinimumReplicasUnavailable"
		rs.Message = fmt.Sprintf("0 of %d desired pods ready", desired)
	case ready < desired:
		rs.Phase = StatusWarning
		rs.Reason = "PartialAvailability"
		rs.Message = fmt.Sprintf("%d of %d desired pods ready", ready, desired)
	case updated < desired:
		rs.Phase = StatusWarning
		rs.Reason = "RollingUpdate"
		rs.Message = fmt.Sprintf("Rolling update in progress: %d/%d updated", updated, desired)
	default:
		rs.Phase = StatusHealthy
		rs.Reason = "Available"
		rs.Message = fmt.Sprintf("All %d pods ready", ready)
	}
	return rs
}

func extractPodStatus(p *corev1.Pod, now time.Time) ResourceStatus {
	phase := p.Status.Phase
	total := len(p.Status.ContainerStatuses)
	readyCount := 0
	var crashReason, crashMsg string

	for _, cs := range p.Status.ContainerStatuses {
		if cs.Ready {
			readyCount++
		}
		if cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff" {
			crashReason = "CrashLoopBackOff"
			if cs.State.Waiting.Message != "" {
				crashMsg = cs.State.Waiting.Message
			} else {
				crashMsg = fmt.Sprintf("Container %s is in CrashLoopBackOff", cs.Name)
			}
		}
	}

	rs := ResourceStatus{
		Kind:      "Pod",
		Name:      p.Name,
		Namespace: p.Namespace,
		Ready:     fmt.Sprintf("%d/%d", readyCount, total),
		Age:       formatAge(p.CreationTimestamp.Time, now),
	}

	// CrashLoopBackOff overrides everything.
	if crashReason != "" {
		rs.Phase = StatusDegraded
		rs.Reason = crashReason
		rs.Message = crashMsg
		return rs
	}

	switch phase {
	case corev1.PodRunning:
		if readyCount == total && total > 0 {
			rs.Phase = StatusHealthy
			rs.Reason = "Running"
			rs.Message = "All containers ready"
		} else {
			rs.Phase = StatusWarning
			rs.Reason = "ContainersNotReady"
			rs.Message = fmt.Sprintf("%d of %d containers ready", readyCount, total)
		}
	case corev1.PodPending:
		rs.Phase = StatusPending
		rs.Reason = "Pending"
		rs.Message = "Pod is waiting to be scheduled or containers to start"
	case corev1.PodSucceeded:
		rs.Phase = StatusCompleted
		rs.Reason = "Succeeded"
		rs.Message = "All containers terminated successfully"
	case corev1.PodFailed:
		rs.Phase = StatusDegraded
		rs.Reason = "Failed"
		if p.Status.Message != "" {
			rs.Message = p.Status.Message
		} else {
			rs.Message = "One or more containers terminated with an error"
		}
	default:
		rs.Phase = StatusUnknown
		rs.Reason = string(phase)
		rs.Message = "Unknown pod phase"
	}
	return rs
}

func extractServiceStatus(svc *corev1.Service, now time.Time) ResourceStatus {
	rs := ResourceStatus{
		Kind:      "Service",
		Name:      svc.Name,
		Namespace: svc.Namespace,
		Ready:     "—",
		Age:       formatAge(svc.CreationTimestamp.Time, now),
	}

	ports := len(svc.Spec.Ports)
	switch svc.Spec.Type {
	case corev1.ServiceTypeClusterIP:
		if svc.Spec.ClusterIP == "None" {
			rs.Phase = StatusHealthy
			rs.Reason = "Headless"
			rs.Message = fmt.Sprintf("Headless service with %d port(s)", ports)
		} else {
			rs.Phase = StatusHealthy
			rs.Reason = "ClusterIP"
			rs.Message = fmt.Sprintf("ClusterIP %s, %d port(s)", svc.Spec.ClusterIP, ports)
		}
	case corev1.ServiceTypeNodePort:
		rs.Phase = StatusHealthy
		rs.Reason = "NodePort"
		rs.Message = fmt.Sprintf("NodePort service, %d port(s)", ports)
	case corev1.ServiceTypeLoadBalancer:
		if len(svc.Status.LoadBalancer.Ingress) > 0 {
			rs.Phase = StatusHealthy
			rs.Reason = "LoadBalancerReady"
			rs.Message = fmt.Sprintf("LoadBalancer provisioned, %d port(s)", ports)
		} else {
			rs.Phase = StatusPending
			rs.Reason = "LoadBalancerPending"
			rs.Message = "Waiting for load balancer to be provisioned"
		}
	case corev1.ServiceTypeExternalName:
		rs.Phase = StatusHealthy
		rs.Reason = "ExternalName"
		rs.Message = fmt.Sprintf("ExternalName → %s", svc.Spec.ExternalName)
	default:
		rs.Phase = StatusUnknown
		rs.Reason = string(svc.Spec.Type)
		rs.Message = "Unknown service type"
	}
	return rs
}

func extractIngressStatus(ing *networkingv1.Ingress, now time.Time) ResourceStatus {
	rules := len(ing.Spec.Rules)
	loadBalancerReady := len(ing.Status.LoadBalancer.Ingress) > 0

	rs := ResourceStatus{
		Kind:      "Ingress",
		Name:      ing.Name,
		Namespace: ing.Namespace,
		Ready:     "—",
		Age:       formatAge(ing.CreationTimestamp.Time, now),
	}

	if loadBalancerReady {
		rs.Phase = StatusHealthy
		rs.Reason = "Ready"
		rs.Message = fmt.Sprintf("Load balancer provisioned, %d rule(s)", rules)
	} else {
		rs.Phase = StatusPending
		rs.Reason = "LoadBalancerPending"
		rs.Message = fmt.Sprintf("Waiting for load balancer, %d rule(s)", rules)
	}
	return rs
}

func extractHPAStatus(hpa *autoscalingv2.HorizontalPodAutoscaler, now time.Time) ResourceStatus {
	current := hpa.Status.CurrentReplicas
	desired := hpa.Status.DesiredReplicas
	minReplicas := int32(1)
	if hpa.Spec.MinReplicas != nil {
		minReplicas = *hpa.Spec.MinReplicas
	}
	maxReplicas := hpa.Spec.MaxReplicas

	rs := ResourceStatus{
		Kind:      "HorizontalPodAutoscaler",
		Name:      hpa.Name,
		Namespace: hpa.Namespace,
		Ready:     fmt.Sprintf("%d→%d", current, desired),
		Age:       formatAge(hpa.CreationTimestamp.Time, now),
	}

	if current < minReplicas {
		rs.Phase = StatusWarning
		rs.Reason = "BelowMinReplicas"
		rs.Message = fmt.Sprintf("Current replicas %d below minimum %d", current, minReplicas)
	} else if desired > maxReplicas {
		rs.Phase = StatusWarning
		rs.Reason = "AboveMaxReplicas"
		rs.Message = fmt.Sprintf("Desired replicas %d above maximum %d", desired, maxReplicas)
	} else {
		rs.Phase = StatusHealthy
		rs.Reason = "ScalingActive"
		rs.Message = fmt.Sprintf("Scaling between %d and %d, currently at %d", minReplicas, maxReplicas, current)
	}
	return rs
}

func staticStatus(kind, name, ns string, created, now time.Time, message string) ResourceStatus {
	return ResourceStatus{
		Kind:      kind,
		Name:      name,
		Namespace: ns,
		Phase:     StatusHealthy,
		Ready:     "—",
		Reason:    "Static",
		Message:   message,
		Age:       formatAge(created, now),
	}
}

// ── helpers ─────────────────────────────────────────────────────────────────

// worstPhase returns the more severe of two phases (for namespace rollup).
func worstPhase(a, b StatusPhase) StatusPhase {
	rank := func(p StatusPhase) int {
		switch p {
		case StatusDegraded:
			return 5
		case StatusWarning:
			return 4
		case StatusPending:
			return 3
		case StatusUnknown:
			return 2
		case StatusCompleted:
			return 1
		case StatusHealthy:
			return 0
		default:
			return 2
		}
	}
	if rank(b) > rank(a) {
		return b
	}
	return a
}

// formatAge renders a human-readable age string from a creation timestamp.
func formatAge(created, now time.Time) string {
	if created.IsZero() {
		return "—"
	}
	d := now.Sub(created)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh%dm", h, m)
	case d < 30*24*time.Hour:
		days := int(d.Hours()) / 24
		h := int(d.Hours()) % 24
		return fmt.Sprintf("%dd%dh", days, h)
	default:
		days := int(d.Hours()) / 24
		return fmt.Sprintf("%dd", days)
	}
}
