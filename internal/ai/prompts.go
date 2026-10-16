package ai

import (
	"encoding/json"
	"fmt"

	"github.com/clustermind/clustermind/internal/graph"
	"github.com/clustermind/clustermind/internal/probe"
)

const systemPersona = `You are a Kubernetes infrastructure expert with deep knowledge of
microservices architecture, service mesh, and cluster operations.
Always respond with precise, actionable insights based on the provided cluster data.`

// ClusterSummaryPrompt builds system+user prompts for cluster structure explanation.
func ClusterSummaryPrompt(snap *probe.ClusterSnapshot, g *graph.DependencyGraph) (system, user string) {
	summary := map[string]any{
		"deployments": summarizeDeployments(snap),
		"services":    summarizeServices(snap),
		"ingresses":   summarizeIngresses(snap),
		"edges":       summarizeEdges(g),
	}
	data, _ := json.Marshal(summary)

	// Truncate to ~4000 chars to stay within token limits
	dataStr := string(data)
	if len(dataStr) > 4000 {
		dataStr = dataStr[:4000] + "..."
	}

	system = systemPersona
	user = fmt.Sprintf(`Analyze this Kubernetes cluster and provide a concise summary in Korean.
Explain: (1) overall architecture, (2) external traffic flow, (3) key service dependencies.
Cluster data: %s`, dataStr)
	return
}

// ImpactAnalysisPrompt builds prompts for failure impact reasoning.
func ImpactAnalysisPrompt(target string, affected []string, g *graph.DependencyGraph) (system, user string) {
	data, _ := json.Marshal(map[string]any{
		"failedResource":    target,
		"affectedResources": affected,
		"totalNodes":        len(g.Nodes),
		"totalEdges":        len(g.Edges),
	})

	system = systemPersona + "\nAlways respond with valid JSON only. No markdown, no explanation outside JSON."
	user = fmt.Sprintf(`A Kubernetes resource has failed. Analyze the blast radius and provide a JSON response.

Failed resource: %s
BFS-detected affected resources: %v

Respond ONLY with this JSON structure:
{
  "severity": "critical|high|medium|low",
  "explanation": "Korean explanation of impact chain and business consequences"
}

Severity guide:
- critical: core user-facing service or payment system affected
- high: significant feature degradation
- medium: partial degradation, workarounds exist  
- low: minimal impact

Context: %s`, target, affected, string(data))
	return
}

func summarizeDeployments(snap *probe.ClusterSnapshot) []map[string]any {
	out := make([]map[string]any, 0, len(snap.Deployments))
	for _, d := range snap.Deployments {
		m := map[string]any{"name": d.Name, "namespace": d.Namespace}
		if d.Spec.Replicas != nil {
			m["replicas"] = *d.Spec.Replicas
		}
		out = append(out, m)
	}
	return out
}

func summarizeServices(snap *probe.ClusterSnapshot) []map[string]any {
	out := make([]map[string]any, 0, len(snap.Services))
	for _, s := range snap.Services {
		out = append(out, map[string]any{
			"name": s.Name, "namespace": s.Namespace,
			"type": string(s.Spec.Type), "selector": s.Spec.Selector,
		})
	}
	return out
}

func summarizeIngresses(snap *probe.ClusterSnapshot) []map[string]any {
	out := make([]map[string]any, 0, len(snap.Ingresses))
	for _, ing := range snap.Ingresses {
		rules := []string{}
		for _, r := range ing.Spec.Rules {
			rules = append(rules, r.Host)
		}
		out = append(out, map[string]any{"name": ing.Name, "namespace": ing.Namespace, "hosts": rules})
	}
	return out
}

func summarizeEdges(g *graph.DependencyGraph) []map[string]any {
	out := make([]map[string]any, 0, len(g.Edges))
	for _, e := range g.Edges {
		out = append(out, map[string]any{
			"src": e.Source, "tgt": e.Target, "type": e.EdgeType,
		})
	}
	return out
}
