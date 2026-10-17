package graph

import (
	"net/url"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/clustermind/clustermind/internal/probe"
)

const minConfidence = 0.3

// GraphBuilder constructs a DependencyGraph from a ClusterSnapshot.
type GraphBuilder struct{}

// NewGraphBuilder creates a new GraphBuilder.
func NewGraphBuilder() *GraphBuilder { return &GraphBuilder{} }

// Build analyses a ClusterSnapshot and returns a DependencyGraph.
func (b *GraphBuilder) Build(snap *probe.ClusterSnapshot) *DependencyGraph {
	g := &DependencyGraph{}
	nodeIndex := map[string]*ResourceNode{}

	addNode := func(n ResourceNode) {
		if _, exists := nodeIndex[n.ID]; !exists {
			nodeIndex[n.ID] = &n
			g.Nodes = append(g.Nodes, n)
		}
	}

	addEdge := func(src, tgt, edgeType string, confidence float64) {
		if confidence < minConfidence {
			return
		}
		g.Edges = append(g.Edges, DependencyEdge{
			Source:     src,
			Target:     tgt,
			EdgeType:   edgeType,
			Confidence: confidence,
			Direction:  "downstream",
		})
	}

	// --- Register all nodes ---

	for _, ns := range snap.Namespaces {
		addNode(ResourceNode{
			ID: NodeID("Namespace", "", ns.Name), Kind: "Namespace",
			Name: ns.Name, Labels: ns.Labels, Meta: map[string]any{},
		})
	}

	for _, d := range snap.Deployments {
		meta := map[string]any{}
		if d.Spec.Replicas != nil {
			meta["replicas"] = *d.Spec.Replicas
		}
		addNode(ResourceNode{
			ID: NodeID("Deployment", d.Namespace, d.Name), Kind: "Deployment",
			Name: d.Name, Namespace: d.Namespace,
			Labels: d.Spec.Template.Labels, Meta: meta,
		})
	}

	for _, ss := range snap.StatefulSets {
		addNode(ResourceNode{
			ID: NodeID("StatefulSet", ss.Namespace, ss.Name), Kind: "StatefulSet",
			Name: ss.Name, Namespace: ss.Namespace,
			Labels: ss.Spec.Template.Labels, Meta: map[string]any{},
		})
	}

	for _, ds := range snap.DaemonSets {
		addNode(ResourceNode{
			ID: NodeID("DaemonSet", ds.Namespace, ds.Name), Kind: "DaemonSet",
			Name: ds.Name, Namespace: ds.Namespace,
			Labels: ds.Spec.Template.Labels, Meta: map[string]any{},
		})
	}

	for _, svc := range snap.Services {
		addNode(ResourceNode{
			ID: NodeID("Service", svc.Namespace, svc.Name), Kind: "Service",
			Name: svc.Name, Namespace: svc.Namespace,
			Labels: svc.Labels,
			Meta:   map[string]any{"selector": svc.Spec.Selector, "type": string(svc.Spec.Type)},
		})
	}

	for _, ing := range snap.Ingresses {
		addNode(ResourceNode{
			ID: NodeID("Ingress", ing.Namespace, ing.Name), Kind: "Ingress",
			Name: ing.Name, Namespace: ing.Namespace,
			Labels: ing.Labels, Meta: map[string]any{},
		})
	}

	for _, hpa := range snap.HPAs {
		addNode(ResourceNode{
			ID: NodeID("HPA", hpa.Namespace, hpa.Name), Kind: "HPA",
			Name: hpa.Name, Namespace: hpa.Namespace,
			Labels: hpa.Labels, Meta: map[string]any{},
		})
	}

	// --- Infer edges ---

	// Rule 1: Service selector → Deployment/StatefulSet/DaemonSet label match
	for _, svc := range snap.Services {
		if len(svc.Spec.Selector) == 0 {
			continue
		}
		svcID := NodeID("Service", svc.Namespace, svc.Name)

		for _, d := range snap.Deployments {
			if d.Namespace == svc.Namespace && labelsMatch(svc.Spec.Selector, d.Spec.Template.Labels) {
				addEdge(svcID, NodeID("Deployment", d.Namespace, d.Name), "selector_match", 0.95)
			}
		}
		for _, ss := range snap.StatefulSets {
			if ss.Namespace == svc.Namespace && labelsMatch(svc.Spec.Selector, ss.Spec.Template.Labels) {
				addEdge(svcID, NodeID("StatefulSet", ss.Namespace, ss.Name), "selector_match", 0.95)
			}
		}
		for _, ds := range snap.DaemonSets {
			if ds.Namespace == svc.Namespace && labelsMatch(svc.Spec.Selector, ds.Spec.Template.Labels) {
				addEdge(svcID, NodeID("DaemonSet", ds.Namespace, ds.Name), "selector_match", 0.95)
			}
		}
	}

	// Rule 2: Ingress → Service (backend reference)
	for _, ing := range snap.Ingresses {
		ingID := NodeID("Ingress", ing.Namespace, ing.Name)
		for _, rule := range ing.Spec.Rules {
			if rule.HTTP == nil {
				continue
			}
			for _, path := range rule.HTTP.Paths {
				if path.Backend.Service != nil {
					svcName := path.Backend.Service.Name
					addEdge(ingID, NodeID("Service", ing.Namespace, svcName), "ingress_backend", 1.0)
				}
			}
		}
		// DefaultBackend
		if ing.Spec.DefaultBackend != nil && ing.Spec.DefaultBackend.Service != nil {
			addEdge(ingID, NodeID("Service", ing.Namespace, ing.Spec.DefaultBackend.Service.Name), "ingress_backend", 1.0)
		}
	}

	// Rule 3: HPA → Deployment/StatefulSet (scaleTargetRef)
	for _, hpa := range snap.HPAs {
		hpaID := NodeID("HPA", hpa.Namespace, hpa.Name)
		ref := hpa.Spec.ScaleTargetRef
		targetID := NodeID(ref.Kind, hpa.Namespace, ref.Name)
		addEdge(hpaID, targetID, "hpa_target", 1.0)
	}

	// Rule 4: Name-prefix weak inference (confidence 0.4)
	// e.g. "api-svc" and "api-deployment" share prefix "api"
	b.inferByPrefix(snap, g, addEdge)

	// Rule 5: Env-var URL reference inference (confidence 0.7)
	// e.g. PAYMENT_SERVICE_URL="http://payment-svc:8080" → Deployment → Service
	b.inferFromEnv(snap, g, addEdge)

	return g
}

// labelsMatch returns true if all selector keys exist in podLabels with matching values.
func labelsMatch(selector, podLabels map[string]string) bool {
	if len(selector) == 0 {
		return false
	}
	for k, v := range selector {
		if podLabels[k] != v {
			return false
		}
	}
	return true
}

// inferByPrefix adds low-confidence edges between resources sharing a name prefix.
func (b *GraphBuilder) inferByPrefix(snap *probe.ClusterSnapshot, g *DependencyGraph, addEdge func(src, tgt, edgeType string, confidence float64)) {
	type entry struct{ kind, ns, name string }
	all := []entry{}

	for _, svc := range snap.Services {
		all = append(all, entry{"Service", svc.Namespace, svc.Name})
	}
	for _, d := range snap.Deployments {
		all = append(all, entry{"Deployment", d.Namespace, d.Name})
	}

	seen := map[string]bool{}
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			a, b2 := all[i], all[j]
			if a.ns != b2.ns || a.kind == b2.kind {
				continue
			}
			prefix := commonPrefix(a.name, b2.name)
			if len(prefix) < 3 {
				continue
			}
			key := NodeID(a.kind, a.ns, a.name) + "|" + NodeID(b2.kind, b2.ns, b2.name)
			if seen[key] {
				continue
			}
			seen[key] = true
			// Service → Deployment direction
			if a.kind == "Service" {
				addEdge(NodeID(a.kind, a.ns, a.name), NodeID(b2.kind, b2.ns, b2.name), "name_prefix", 0.4)
			} else {
				addEdge(NodeID(b2.kind, b2.ns, b2.name), NodeID(a.kind, a.ns, a.name), "name_prefix", 0.4)
			}
		}
	}
}

func commonPrefix(a, b string) string {
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	// strip trailing hyphen/dash
	p := a[:i]
	p = strings.TrimRight(p, "-_")
	return p
}

// inferFromEnv adds edges from workloads to Services referenced by
// URL-shaped container env values (e.g. "http://payment-svc:8080" or
// "postgresql://postgres-svc:5432").
func (b *GraphBuilder) inferFromEnv(snap *probe.ClusterSnapshot, g *DependencyGraph, addEdge func(src, tgt, edgeType string, confidence float64)) {
	services := map[string]bool{} // "namespace/name" → exists
	for _, svc := range snap.Services {
		services[svc.Namespace+"/"+svc.Name] = true
	}

	type key struct{ src, tgt string }
	seen := map[key]bool{}

	addTemplate := func(kind, ns, name string, tpl corev1.PodTemplateSpec) {
		for _, c := range tpl.Spec.Containers {
			for _, env := range c.Env {
				host, ok := urlHostname(env.Value)
				if !ok {
					continue
				}
				if !services[ns+"/"+host] {
					continue
				}
				src := NodeID(kind, ns, name)
				tgt := NodeID("Service", ns, host)
				if k := (key{src, tgt}); !seen[k] {
					seen[k] = true
					addEdge(src, tgt, "env_reference", 0.7)
				}
			}
		}
	}

	for _, d := range snap.Deployments {
		addTemplate("Deployment", d.Namespace, d.Name, d.Spec.Template)
	}
	for _, ss := range snap.StatefulSets {
		addTemplate("StatefulSet", ss.Namespace, ss.Name, ss.Spec.Template)
	}
	for _, ds := range snap.DaemonSets {
		addTemplate("DaemonSet", ds.Namespace, ds.Name, ds.Spec.Template)
	}
}

// urlHostname extracts the hostname from a URL-shaped string.
func urlHostname(v string) (string, bool) {
	v = strings.TrimSpace(v)
	if !strings.Contains(v, "://") {
		return "", false
	}
	u, err := url.Parse(v)
	if err != nil {
		return "", false
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return "", false
	}
	return host, true
}
