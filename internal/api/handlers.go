package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/clustermind/clustermind/internal/probe"
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
