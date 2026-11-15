package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	aiPkg "github.com/clustermind/clustermind/internal/ai"
	"github.com/clustermind/clustermind/internal/graph"
	"github.com/clustermind/clustermind/internal/probe"
	"github.com/clustermind/clustermind/internal/render"
)

// handleSnapshot returns a lightweight summary of collected cluster resources.
func (s *Server) handleSnapshot(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	snap, err := s.collector.Collect(ctx)
	if err != nil {
		slog.Error("collect failed", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("collect: %s", err)})
		return
	}

	c.JSON(http.StatusOK, SnapshotSummary{
		Namespaces:   len(snap.Namespaces),
		Deployments:  len(snap.Deployments),
		StatefulSets: len(snap.StatefulSets),
		DaemonSets:   len(snap.DaemonSets),
		Pods:         len(snap.Pods),
		Services:     len(snap.Services),
		Ingresses:    len(snap.Ingresses),
		HPAs:         len(snap.HPAs),
	})
}

// handleGraph collects the cluster, builds the dependency graph,
// and returns React Flow-compatible nodes+edges JSON.
func (s *Server) handleGraph(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	snap, err := s.collector.Collect(ctx)
	if err != nil {
		slog.Error("collect failed", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("collect: %s", err)})
		return
	}

	g := s.builder.Build(snap)
	result := s.renderer.RenderGraph(g, nil)

	c.JSON(http.StatusOK, result)
}

// handleImpact runs failure impact analysis for the given target resource.
func (s *Server) handleImpact(c *gin.Context) {
	var req ImpactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("invalid request: %s", err)})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	snap, err := s.collector.Collect(ctx)
	if err != nil {
		slog.Error("collect failed", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("collect: %s", err)})
		return
	}

	g := s.builder.Build(snap)

	impact, err := s.analyzer.AnalyzeFailure(ctx, req.Target, g)
	if err != nil {
		slog.Error("impact analysis failed", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("impact: %s", err)})
		return
	}

	graphResult := s.renderer.RenderGraph(g, impact.ImpactMap)

	c.JSON(http.StatusOK, ImpactResponse{
		Target:            impact.Target,
		GeneratedBy:       impact.GeneratedBy,
		AffectedResources: impact.AffectedResources,
		PropagationPaths:  impact.PropagationPaths,
		Severity:          impact.Severity,
		Explanation:       impact.Explanation,
		Graph:             graphResult,
	})
}

// handleWatch streams cluster change events via Server-Sent Events.
func (s *Server) handleWatch(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Send initial event
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	sendSnapshot := func() {
		snap, err := s.collector.Collect(ctx)
		if err != nil {
			c.SSEvent("error", gin.H{"error": err.Error()})
			return
		}
		g := s.builder.Build(snap)
		result := s.renderer.RenderGraph(g, nil)
		c.SSEvent("graph", result)
		c.Writer.Flush()
	}

	sendSnapshot()

	for {
		select {
		case <-ticker.C:
			sendSnapshot()
		case <-ctx.Done():
			return
		}
	}
}

// handleListNamespaces returns every namespace with aggregated resource
// counts and a phase rollup derived from each namespace's resources.
func (s *Server) handleListNamespaces(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	snap, err := s.collector.Collect(ctx)
	if err != nil {
		slog.Error("collect failed", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("collect: %s", err)})
		return
	}

	statuses := probe.ExtractResourceStatuses(snap)
	stats := probe.BuildNamespaceStats(snap, statuses)

	c.JSON(http.StatusOK, NamespaceListResponse{Namespaces: stats})
}

// handleNamespaceResources returns every resource in the given namespace
// with its normalized health status. Supports an optional ?kind= filter.
func (s *Server) handleNamespaceResources(c *gin.Context) {
	ns := c.Param("namespace")
	if ns == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "namespace path parameter is required"})
		return
	}
	kindFilter := c.Query("kind")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	snap, err := s.collector.Collect(ctx)
	if err != nil {
		slog.Error("collect failed", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("collect: %s", err)})
		return
	}

	statuses := probe.ExtractResourceStatuses(snap)
	filtered := probe.FilterStatuses(statuses, ns, kindFilter)

	c.JSON(http.StatusOK, NamespaceResourcesResponse{
		Namespace: ns,
		Resources: filtered,
	})
}

// handleTwinSnapshot collects the cluster, builds the dependency graph, and
// returns the digital-twin view model (identity + health + metrics per node).
func (s *Server) handleTwinSnapshot(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	snap, err := s.collector.Collect(ctx)
	if err != nil {
		slog.Error("collect failed", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("collect: %s", err)})
		return
	}

	g := s.builder.Build(snap)
	statuses := probe.ExtractResourceStatuses(snap)

	statusByID := make(map[string]probe.ResourceStatus, len(statuses))
	for _, rs := range statuses {
		statusByID[graph.NodeID(rs.Kind, rs.Namespace, rs.Name)] = rs
	}

	positions := render.GridPositions(g.Nodes)

	nodes := make([]TwinNode, 0, len(g.Nodes))
	for _, n := range g.Nodes {
		node := TwinNode{
			ID:        n.ID,
			Name:      n.Name,
			Kind:      strings.ToLower(n.Kind),
			Namespace: n.Namespace,
			Status:    "unknown",
			Metrics:   []TwinMetric{},
			Labels:    n.Labels,
			Position:  positions[n.ID],
		}

		if rs, ok := statusByID[n.ID]; ok {
			node.Status = healthFromPhase(rs.Phase)
			node.Reason = rs.Reason
			node.Metrics = twinMetrics(rs)
		}

		if v, ok := n.Meta["replicas"]; ok {
			if f, ok := v.(float64); ok {
				r32 := int32(f)
				node.Replicas = &r32
			}
		}

		nodes = append(nodes, node)
	}

	edges := make([]TwinEdge, 0, len(g.Edges))
	for _, e := range g.Edges {
		edges = append(edges, TwinEdge{
			ID:       fmt.Sprintf("%s|%s|%s", e.EdgeType, e.Source, e.Target),
			Source:   e.Source,
			Target:   e.Target,
			Protocol: e.EdgeType,
		})
	}

	ns := s.collector.Namespace()
	if ns == "" {
		ns = "all"
	}

	incidents, changes := BuildTwinIncidents(snap, g)

	c.JSON(http.StatusOK, TwinSnapshot{
		Cluster:    s.clusterName,
		Namespace:  ns,
		CapturedAt: time.Now().UTC(),
		Nodes:      nodes,
		Edges:      edges,
		Changes:    changes,
		Incidents:  incidents,
	})
}

// healthFromPhase maps a probe StatusPhase to the digital-twin health status.
func healthFromPhase(p probe.StatusPhase) string {
	switch p {
	case probe.StatusHealthy, probe.StatusCompleted:
		return "healthy"
	case probe.StatusWarning, probe.StatusPending:
		return "degraded"
	case probe.StatusDegraded:
		return "unhealthy"
	default:
		return "unknown"
	}
}

// twinMetrics derives display metrics for a node card from its live status.
func twinMetrics(rs probe.ResourceStatus) []TwinMetric {
	metrics := make([]TwinMetric, 0, 3)
	if rs.Ready != "" && rs.Ready != "—" {
		metrics = append(metrics, TwinMetric{Label: "Ready", Value: rs.Ready})
	}
	if rs.Age != "" && rs.Age != "—" {
		metrics = append(metrics, TwinMetric{Label: "Age", Value: rs.Age})
	}
	return metrics
}

// handleInvestigate generates a structured investigation report for a
// derived incident (Gemini-backed with a deterministic fallback).
func (s *Server) handleInvestigate(c *gin.Context) {
	var req InvestigateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("invalid request: %s", err)})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()

	snap, err := s.collector.Collect(ctx)
	if err != nil {
		slog.Error("collect failed", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("collect: %s", err)})
		return
	}

	g := s.builder.Build(snap)
	incidents, _ := BuildTwinIncidents(snap, g)

	var incident *TwinIncident
	for i := range incidents {
		if incidents[i].ID == req.IncidentID {
			incident = &incidents[i]
			break
		}
	}
	if incident == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: fmt.Sprintf("incident not found: %s", req.IncidentID)})
		return
	}

	health := "healthy"
	switch incident.Severity {
	case "sev1":
		health = "unhealthy"
	case "sev2":
		health = "degraded"
	}

	// Progress conditions like "Available" on a healthy resource are not
	// failure causes — only surface the reason when the resource is degraded.
	reason := ""
	if health != "healthy" {
		statuses := probe.ExtractResourceStatuses(snap)
		for _, rs := range statuses {
			if graph.NodeID(rs.Kind, rs.Namespace, rs.Name) == incident.PrimaryResourceID {
				reason = rs.Reason
				break
			}
		}
	}

	timeline := make([]aiPkg.EvidenceInput, 0, len(incident.Timeline))
	for _, ev := range incident.Timeline {
		timeline = append(timeline, aiPkg.EvidenceInput{
			ID:     ev.ID,
			Time:   ev.Time,
			Source: ev.Source,
			Title:  ev.Title,
			Detail: ev.Detail,
		})
	}

	result := s.analyzer.Investigate(ctx, aiPkg.InvestigationRequest{
		IncidentID:    incident.ID,
		PrimaryTarget: incident.PrimaryResourceID,
		Health:        health,
		Reason:        reason,
		Timeline:      timeline,
	}, g)

	c.JSON(http.StatusOK, result)
}

// handleGetSettings returns current AI settings and available providers/models.
func (s *Server) handleGetSettings(c *gin.Context) {
	supported := aiPkg.SupportedProviders()
	infos := make([]ProviderInfo, 0, len(supported))
	for _, p := range supported {
		infos = append(infos, ProviderInfo{
			ID:           p.ID,
			Name:         p.Name,
			DefaultModel: p.DefaultModel,
			Models:       p.Models,
			KeyHelpURL:   p.KeyHelpURL,
			KeyPrefix:    p.KeyPrefix,
		})
	}

	provider := aiPkg.ProviderGemini
	model := aiPkg.DefaultModel
	hasKey := false
	maskedKey := ""

	if s.engine != nil {
		provider = s.engine.Provider()
		model = s.engine.Model()
		hasKey = s.engine.HasAPIKey()
		maskedKey = s.engine.MaskedAPIKey()
	}

	c.JSON(http.StatusOK, SettingsResponse{
		Provider:           provider,
		Model:              model,
		HasAPIKey:          hasKey,
		MaskedAPIKey:       maskedKey,
		SupportedProviders: infos,
	})
}

// handleUpdateSettings updates active AI provider, model, and/or API key.
func (s *Server) handleUpdateSettings(c *gin.Context) {
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("invalid request: %s", err)})
		return
	}

	if s.engine == nil {
		s.engine = aiPkg.NewEngine("", "")
	}

	if err := s.engine.Configure(req.Provider, req.APIKey, req.Model); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	s.handleGetSettings(c)
}

// handleTestSettings verifies connection with the specified provider and API key.
func (s *Server) handleTestSettings(c *gin.Context) {
	var req TestSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("invalid request: %s", err)})
		return
	}

	if s.engine == nil {
		s.engine = aiPkg.NewEngine("", "")
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	if err := s.engine.Test(ctx, req.Provider, req.APIKey, req.Model); err != nil {
		c.JSON(http.StatusOK, TestSettingsResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, TestSettingsResponse{
		Success: true,
		Message: "Connection test succeeded!",
	})
}

// handleListModels queries the AI provider API for live available models.
func (s *Server) handleListModels(c *gin.Context) {
	var req ListModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: fmt.Sprintf("invalid request: %s", err)})
		return
	}

	if s.engine == nil {
		s.engine = aiPkg.NewEngine("", "")
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()

	models, err := s.engine.FetchAvailableModels(ctx, req.Provider, req.APIKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	provider := req.Provider
	if provider == "" {
		provider = s.engine.Provider()
	}

	c.JSON(http.StatusOK, ListModelsResponse{
		Provider: provider,
		Models:   models,
	})
}


