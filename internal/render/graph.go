// Package render converts DependencyGraph to React Flow-compatible JSON.
package render

import (
	"fmt"

	"github.com/clustermind/clustermind/internal/graph"
)

// Position is a 2D coordinate for React Flow node placement.
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// FlowNodeData carries display data for each React Flow node.
type FlowNodeData struct {
	Label       string `json:"label"`
	Kind        string `json:"kind"`
	Namespace   string `json:"namespace"`
	ImpactLevel string `json:"impactLevel"` // critical|direct|indirect|none
	Replicas    *int32 `json:"replicas,omitempty"`
}

// FlowNode is a React Flow-compatible node.
type FlowNode struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Position Position     `json:"position"`
	Data     FlowNodeData `json:"data"`
}

// FlowEdge is a React Flow-compatible edge.
type FlowEdge struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	Type     string `json:"type"`
	Animated bool   `json:"animated"`
	Label    string `json:"label,omitempty"`
}

// GraphResult is the full React Flow payload returned by the API.
type GraphResult struct {
	Nodes []FlowNode `json:"nodes"`
	Edges []FlowEdge `json:"edges"`
}

// Renderer converts internal graph data to React Flow format.
type Renderer struct{}

// NewRenderer creates a Renderer.
func NewRenderer() *Renderer { return &Renderer{} }

// kindToType maps K8s resource Kind to React Flow custom node type name.
func kindToType(kind string) string {
	switch kind {
	case "Service":
		return "service"
	case "Deployment":
		return "deployment"
	case "Ingress":
		return "ingress"
	case "StatefulSet":
		return "statefulset"
	default:
		return "default"
	}
}

// GridPositions computes a deterministic grid layout for graph nodes.
// Nodes are grouped by Kind so upstream kinds (Ingress) sit on higher rows,
// and spread across columns within each row.
func GridPositions(nodes []graph.ResourceNode) map[string]Position {
	kindOrder := map[string]int{
		"Ingress":     0,
		"Service":     1,
		"Deployment":  2,
		"StatefulSet": 2,
		"DaemonSet":   2,
		"HPA":         3,
		"Namespace":   4,
		"default":     5,
	}
	kindCols := map[int]int{} // row → current column counter

	out := make(map[string]Position, len(nodes))
	for _, n := range nodes {
		row, ok := kindOrder[n.Kind]
		if !ok {
			row = kindOrder["default"]
		}
		col := kindCols[row]
		kindCols[row]++
		out[n.ID] = Position{
			X: float64(col) * 220,
			Y: float64(row) * 140,
		}
	}
	return out
}

// RenderGraph converts a DependencyGraph to a React Flow GraphResult.
// impactMap maps nodeID → "critical"|"direct"|"indirect"|"none".
// Pass nil or empty impactMap to render all nodes as "none".
func (r *Renderer) RenderGraph(g *graph.DependencyGraph, impactMap map[string]string) *GraphResult {
	if impactMap == nil {
		impactMap = map[string]string{}
	}

	positions := GridPositions(g.Nodes)

	nodes := make([]FlowNode, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		impactLevel := impactMap[n.ID]
		if impactLevel == "" {
			impactLevel = "none"
		}

		var replicas *int32
		if v, ok := n.Meta["replicas"]; ok {
			if f, ok := v.(float64); ok {
				r32 := int32(f)
				replicas = &r32
			}
		}

		nodes = append(nodes, FlowNode{
			ID:       n.ID,
			Type:     kindToType(n.Kind),
			Position: positions[n.ID],
			Data: FlowNodeData{
				Label:       n.Name,
				Kind:        n.Kind,
				Namespace:   n.Namespace,
				ImpactLevel: impactLevel,
				Replicas:    replicas,
			},
		})
	}

	// Build impact node set for animated edge detection.
	impactedNodes := map[string]bool{}
	for id, lvl := range impactMap {
		if lvl == "critical" || lvl == "direct" || lvl == "indirect" {
			impactedNodes[id] = true
		}
	}

	edges := make([]FlowEdge, 0, len(g.Edges))
	for i, e := range g.Edges {
		animated := impactedNodes[e.Source] || impactedNodes[e.Target]
		edges = append(edges, FlowEdge{
			ID:       fmt.Sprintf("e%d-%s-%s", i, e.Source, e.Target),
			Source:   e.Source,
			Target:   e.Target,
			Type:     "smoothstep",
			Animated: animated,
			Label:    e.EdgeType,
		})
	}

	return &GraphResult{Nodes: nodes, Edges: edges}
}
