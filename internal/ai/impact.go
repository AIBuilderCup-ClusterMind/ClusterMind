package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/clustermind/clustermind/internal/graph"
)

// ImpactLevel represents how severely a resource is affected.
type ImpactLevel string

const (
	ImpactCritical ImpactLevel = "critical"
	ImpactDirect   ImpactLevel = "direct"
	ImpactIndirect ImpactLevel = "indirect"
	ImpactNone     ImpactLevel = "none"
)

// ImpactResult is the output of a failure impact analysis.
type ImpactResult struct {
	Target            string              `json:"target"`
	AffectedResources []string            `json:"affectedResources"` // node IDs
	PropagationPaths  [][]string          `json:"propagationPaths"`
	Severity          string              `json:"severity"` // critical|high|medium|low
	Explanation       string              `json:"explanation"`
	ImpactMap         map[string]string   `json:"impactMap"` // nodeID → critical|direct|indirect|none
}

// Analyzer performs failure impact analysis using graph traversal + LLM reasoning.
type Analyzer struct {
	engine *Engine
}

// NewAnalyzer creates an Analyzer backed by the given Engine.
func NewAnalyzer(engine *Engine) *Analyzer {
	return &Analyzer{engine: engine}
}

// AnalyzeFailure performs BFS on the dependency graph starting from target,
// then asks the LLM to reason about severity and business impact.
func (a *Analyzer) AnalyzeFailure(ctx context.Context, target string, g *graph.DependencyGraph) (*ImpactResult, error) {
	// Build adjacency map: source → []target (downstream)
	adj := map[string][]string{}
	for _, e := range g.Edges {
		if e.Confidence >= 0.3 {
			adj[e.Source] = append(adj[e.Source], e.Target)
		}
	}

	// BFS from target (depth ≤ 5)
	const maxDepth = 5
	impactMap := map[string]string{target: string(ImpactCritical)}
	depthMap := map[string]int{target: 0}
	queue := []string{target}
	paths := [][]string{}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		depth := depthMap[cur]
		if depth >= maxDepth {
			continue
		}
		for _, neighbor := range adj[cur] {
			if _, visited := depthMap[neighbor]; visited {
				continue
			}
			depthMap[neighbor] = depth + 1
			queue = append(queue, neighbor)

			// Propagation path
			paths = append(paths, []string{cur, neighbor})

			// Assign impact level
			if depth+1 == 1 {
				impactMap[neighbor] = string(ImpactDirect)
			} else {
				impactMap[neighbor] = string(ImpactIndirect)
			}
		}
	}

	// Collect affected resource IDs (excluding the target itself)
	affected := []string{}
	for id := range impactMap {
		if id != target {
			affected = append(affected, id)
		}
	}

	slog.Info("bfs complete", "target", target, "affected", len(affected))

	// LLM reasoning for severity + explanation
	severity := "medium"
	explanation := fmt.Sprintf("%s 장애로 인해 %d개 리소스가 영향받습니다.", target, len(affected))

	if a.engine != nil {
		system, user := ImpactAnalysisPrompt(target, affected, g)
		raw, err := a.engine.ChatComplete(ctx, system, user)
		if err != nil {
			slog.Warn("llm impact analysis failed, using fallback", "error", err)
		} else {
			// Strip markdown fences if present
			raw = strings.TrimSpace(raw)
			raw = strings.TrimPrefix(raw, "```json")
			raw = strings.TrimPrefix(raw, "```")
			raw = strings.TrimSuffix(raw, "```")
			raw = strings.TrimSpace(raw)

			var llmResp struct {
				Severity    string `json:"severity"`
				Explanation string `json:"explanation"`
			}
			if err := json.Unmarshal([]byte(raw), &llmResp); err == nil {
				severity = llmResp.Severity
				explanation = llmResp.Explanation
			}
		}
	}

	// Mark remaining nodes as none
	nodeSet := map[string]bool{}
	for _, n := range g.Nodes {
		nodeSet[n.ID] = true
	}
	for id := range nodeSet {
		if _, ok := impactMap[id]; !ok {
			impactMap[id] = string(ImpactNone)
		}
	}

	return &ImpactResult{
		Target:            target,
		AffectedResources: affected,
		PropagationPaths:  paths,
		Severity:          severity,
		Explanation:       explanation,
		ImpactMap:         impactMap,
	}, nil
}
