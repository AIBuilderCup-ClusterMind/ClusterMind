// Command clustermind is the CLI and server entrypoint for ClusterMind.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	aiPkg "github.com/clustermind/clustermind/internal/ai"
	"github.com/clustermind/clustermind/internal/api"
	"github.com/clustermind/clustermind/internal/graph"
	"github.com/clustermind/clustermind/internal/probe"
	"github.com/clustermind/clustermind/internal/render"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel(),
	})))

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		runServe()
	case "analyze":
		runAnalyze()
	case "impact":
		runImpact()
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`ClusterMind — AI Infrastructure Digital Twin

Usage:
  clustermind serve   [--port 8080] [--namespace demo]
  clustermind analyze [--namespace demo]
  clustermind impact  --target Service/demo/payment-svc

Environment variables:
  GEMINI_API_KEY          Gemini API key (required for AI features)
  KUBECONFIG              kubeconfig path (default: ~/.kube/config)
  CLUSTERMIND_PORT        API server port (default: 8080)
  CLUSTERMIND_NAMESPACE   namespace to collect (default: all)
  CLUSTERMIND_LOG_LEVEL   log level: DEBUG|INFO|WARN|ERROR (default: INFO)`)
}

// --- serve ---

func runServe() {
	port := flagValue("--port", "CLUSTERMIND_PORT", "8080")
	ns := flagValue("--namespace", "CLUSTERMIND_NAMESPACE", "")

	clientset := mustKubeClient()
	collector := probe.NewClusterCollector(clientset, ns)
	builder := graph.NewGraphBuilder()
	renderer := render.NewRenderer()
	engine := aiPkg.NewEngine(os.Getenv("GEMINI_API_KEY"), "")
	analyzer := aiPkg.NewAnalyzer(engine)

	srv := api.NewServer(collector, builder, renderer, analyzer, engine)
	if err := srv.Run(":" + port); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

// --- analyze ---

func runAnalyze() {
	ns := flagValue("--namespace", "CLUSTERMIND_NAMESPACE", "")
	clientset := mustKubeClient()
	collector := probe.NewClusterCollector(clientset, ns)

	ctx := context.Background()
	snap, err := collector.Collect(ctx)
	if err != nil {
		slog.Error("collect failed", "error", err)
		os.Exit(1)
	}

	builder := graph.NewGraphBuilder()
	g := builder.Build(snap)

	fmt.Printf("=== ClusterMind Cluster Analysis ===\n")
	fmt.Printf("Nodes: %d  Edges: %d\n\n", len(g.Nodes), len(g.Edges))

	for _, n := range g.Nodes {
		fmt.Printf("  [%s] %s/%s\n", n.Kind, n.Namespace, n.Name)
	}
	fmt.Println()

	engine := aiPkg.NewEngine(os.Getenv("GEMINI_API_KEY"), "")
	system, user := aiPkg.ClusterSummaryPrompt(snap, g)
	summary, err := engine.ChatComplete(ctx, system, user)
	if err != nil {
		slog.Warn("AI summary unavailable", "error", err)
		fmt.Println("[AI summary unavailable — check GEMINI_API_KEY]")
		return
	}
	fmt.Println("=== AI Summary ===")
	fmt.Println(summary)
}

// --- impact ---

func runImpact() {
	target := flagValue("--target", "", "")
	if target == "" {
		fmt.Fprintln(os.Stderr, "error: --target is required")
		os.Exit(1)
	}
	ns := flagValue("--namespace", "CLUSTERMIND_NAMESPACE", "")
	clientset := mustKubeClient()
	collector := probe.NewClusterCollector(clientset, ns)

	ctx := context.Background()
	snap, err := collector.Collect(ctx)
	if err != nil {
		slog.Error("collect failed", "error", err)
		os.Exit(1)
	}

	builder := graph.NewGraphBuilder()
	g := builder.Build(snap)

	engine := aiPkg.NewEngine(os.Getenv("GEMINI_API_KEY"), "")
	analyzer := aiPkg.NewAnalyzer(engine)

	result, err := analyzer.AnalyzeFailure(ctx, target, g)
	if err != nil {
		slog.Error("impact analysis failed", "error", err)
		os.Exit(1)
	}

	fmt.Printf("[%s] %s 장애 영향 범위\n\n", result.Severity, target)

	direct := []string{}
	indirect := []string{}
	for id, lvl := range result.ImpactMap {
		if id == target {
			continue
		}
		switch lvl {
		case "direct":
			direct = append(direct, id)
		case "indirect":
			indirect = append(indirect, id)
		}
	}

	fmt.Printf("├── 직접 영향 (depth 1): %d개\n", len(direct))
	for _, id := range direct {
		fmt.Printf("│   └── %s\n", id)
	}
	fmt.Printf("└── 간접 영향 (depth 2+): %d개\n", len(indirect))
	for _, id := range indirect {
		fmt.Printf("    └── %s\n", id)
	}

	fmt.Printf("\n=== AI 분석 ===\n%s\n", result.Explanation)

	// Save React Flow JSON
	renderer := render.NewRenderer()
	graphResult := renderer.RenderGraph(g, result.ImpactMap)
	outFile := "cluster_impact.json"
	data, _ := json.MarshalIndent(graphResult, "", "  ")
	if err := os.WriteFile(outFile, data, 0o644); err == nil {
		fmt.Printf("\nReact Flow JSON → %s\n", outFile)
	}
}

// --- helpers ---

func mustKubeClient() kubernetes.Interface {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		if home, err := os.UserHomeDir(); err == nil {
			kubeconfig = home + "/.kube/config"
		}
	}
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		slog.Error("kubeconfig error", "error", err)
		os.Exit(1)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		slog.Error("kubernetes client error", "error", err)
		os.Exit(1)
	}
	return clientset
}

// flagValue returns the CLI flag value, falling back to env var, then defaultVal.
func flagValue(flag, envKey, defaultVal string) string {
	args := os.Args[2:]
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	if envKey != "" {
		if v := os.Getenv(envKey); v != "" {
			return v
		}
	}
	return defaultVal
}

func logLevel() slog.Level {
	switch os.Getenv("CLUSTERMIND_LOG_LEVEL") {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
