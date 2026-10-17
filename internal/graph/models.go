// Package graph builds and represents a dependency graph of K8s resources.
package graph

// ResourceNode represents a single K8s resource vertex in the dependency graph.
type ResourceNode struct {
	ID        string            `json:"id"`   // "{Kind}/{namespace}/{name}"
	Kind      string            `json:"kind"` // "Service", "Deployment", ...
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels"`
	Meta      map[string]any    `json:"meta"`
}

// DependencyEdge represents a directed relationship between two ResourceNodes.
type DependencyEdge struct {
	Source     string  `json:"source"`     // ResourceNode.ID
	Target     string  `json:"target"`     // ResourceNode.ID
	EdgeType   string  `json:"edgeType"`   // "selector_match", "ingress_backend", "hpa_target", ...
	Confidence float64 `json:"confidence"` // 0.0 ~ 1.0
	Direction  string  `json:"direction"`  // "upstream" | "downstream" | "bidirectional"
}

// DependencyGraph is the full graph of a cluster snapshot.
type DependencyGraph struct {
	Nodes []ResourceNode   `json:"nodes"`
	Edges []DependencyEdge `json:"edges"`
}

// NodeID returns a canonical ID for a resource.
func NodeID(kind, namespace, name string) string {
	if namespace == "" {
		return kind + "/" + name
	}
	return kind + "/" + namespace + "/" + name
}
