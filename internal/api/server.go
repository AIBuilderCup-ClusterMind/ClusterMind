package api

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	aiPkg "github.com/clustermind/clustermind/internal/ai"
	"github.com/clustermind/clustermind/internal/graph"
	"github.com/clustermind/clustermind/internal/probe"
	"github.com/clustermind/clustermind/internal/render"
)

// Server is the ClusterMind REST API server.
type Server struct {
	router    *gin.Engine
	collector *probe.ClusterCollector
	builder   *graph.GraphBuilder
	renderer  *render.Renderer
	analyzer  *aiPkg.Analyzer
	engine    *aiPkg.Engine
}

// NewServer wires up all dependencies and registers routes.
func NewServer(
	collector *probe.ClusterCollector,
	builder *graph.GraphBuilder,
	renderer *render.Renderer,
	analyzer *aiPkg.Analyzer,
	engine *aiPkg.Engine,
) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	s := &Server{
		router:    r,
		collector: collector,
		builder:   builder,
		renderer:  renderer,
		analyzer:  analyzer,
		engine:    engine,
	}
	s.registerRoutes()
	return s
}

// registerRoutes sets up all API endpoints.
func (s *Server) registerRoutes() {
	s.router.Use(corsMiddleware())

	v1 := s.router.Group("/api/v1")
	{
		cluster := v1.Group("/cluster")
		{
			cluster.GET("/snapshot", s.handleSnapshot)
			cluster.GET("/graph", s.handleGraph)
		}
		v1.GET("/namespaces", s.handleListNamespaces)
		v1.GET("/namespaces/:namespace/resources", s.handleNamespaceResources)
		v1.POST("/impact", s.handleImpact)
		v1.GET("/watch", s.handleWatch)
	}

	// Health check
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
}

// Run starts the HTTP server on the given address (e.g. ":8080").
func (s *Server) Run(addr string) error {
	slog.Info("starting server", "addr", addr)
	return s.router.Run(addr)
}

// corsMiddleware allows all origins for development convenience.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
