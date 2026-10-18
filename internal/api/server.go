package api

import (
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	aiPkg "github.com/clustermind/clustermind/internal/ai"
	"github.com/clustermind/clustermind/internal/graph"
	"github.com/clustermind/clustermind/internal/probe"
	"github.com/clustermind/clustermind/internal/render"
	"github.com/clustermind/clustermind/webdist"
)

// Server is the ClusterMind REST API server.
type Server struct {
	router      *gin.Engine
	collector   probe.Collector
	builder     *graph.GraphBuilder
	renderer    *render.Renderer
	analyzer    *aiPkg.Analyzer
	engine      *aiPkg.Engine
	clusterName string
}

// NewServer wires up all dependencies and registers routes.
func NewServer(
	collector probe.Collector,
	builder *graph.GraphBuilder,
	renderer *render.Renderer,
	analyzer *aiPkg.Analyzer,
	engine *aiPkg.Engine,
	clusterName string,
) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	if clusterName == "" {
		clusterName = "unknown"
	}

	s := &Server{
		router:      r,
		collector:   collector,
		builder:     builder,
		renderer:    renderer,
		analyzer:    analyzer,
		engine:      engine,
		clusterName: clusterName,
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
		v1.GET("/twin/snapshot", s.handleTwinSnapshot)
		v1.GET("/namespaces", s.handleListNamespaces)
		v1.GET("/namespaces/:namespace/resources", s.handleNamespaceResources)
		v1.POST("/impact", s.handleImpact)
		v1.POST("/investigate", s.handleInvestigate)
		v1.GET("/watch", s.handleWatch)
	}

	// Health check
	s.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	s.registerSPA()
}

// registerSPA serves the embedded Web UI (webdist) with an SPA fallback:
// unknown non-/api paths return index.html so client-side routes work.
// When no frontend bundle is embedded the server stays API-only.
func (s *Server) registerSPA() {
	sub, err := fs.Sub(webdist.FS, "dist")
	if err != nil {
		slog.Info("web UI not embedded; running API-only")
		return
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		slog.Info("web UI bundle not found; running API-only")
		return
	}
	indexHTML, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		slog.Warn("cannot read embedded index.html", "error", err)
		return
	}
	files := http.FileServer(http.FS(sub))
	s.router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "not found"})
			return
		}
		name := strings.TrimPrefix(c.Request.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}
		if f, err := sub.Open(name); err == nil {
			_ = f.Close()
			files.ServeHTTP(c.Writer, c.Request)
			return
		}
		// SPA fallback: unknown path → index.html (client router decides).
		c.Header("Cache-Control", "no-cache")
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
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
