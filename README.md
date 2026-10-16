# ClusterMind — AI Infrastructure Digital Twin

[English](./README.md) | [한국어](./README.ko.md)

[![Go Version](https://img.shields.io/badge/Go-1.22+-blue.svg)](https://golang.org)
[![Gemini](https://img.shields.io/badge/Gemini-2.0--flash-4285F4.svg)](https://aistudio.google.com)
[![Tests](https://img.shields.io/badge/Tests-4%20passed-brightgreen.svg)](#test-results)
[![React Flow](https://img.shields.io/badge/React%20Flow-v12-purple.svg)](https://reactflow.dev)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](./LICENSE)

`clustermind` is an open-source AI digital twin system that **real-time analyzes Kubernetes clusters** and automatically generates topology diagrams, dependency graphs, and failure blast-radius reports.

It collects cluster resources (read-only), builds a dependency graph via label-selector and ingress inference, and uses an LLM to reason about architecture structure and failure propagation — visualized as an interactive React Flow graph in the browser.

---

## Key Features

- **Read-Only Collection**: ClusterMind never writes to the cluster; all operations are strictly read-only.
- **Automatic Dependency Graph**: Infers edges from Label Selectors, Ingress backends, HPA targets, ConfigMap mounts, and name-prefix heuristics — with per-edge confidence scores.
- **AI Failure Impact Analysis**: BFS graph traversal (depth ≤ 5) feeds into an LLM that reasons about propagation paths, business consequences, and severity (critical / high / medium / low).
- **Interactive React Flow Graph**: Drag, zoom, minimap — failure nodes highlighted with animated colour coding in real time.
- **Safe Secret Handling**: `Secret.Data` is set to `nil` immediately after collection; secret values never appear in API responses or logs.
- **SSE Live Watch**: Streams cluster change events every 30 seconds for always-current topology.
- **CLI + REST API**: `serve`, `analyze`, `impact` CLI commands backed by a Gin HTTP server.

---

## CLI Command Reference

| Command | Usage | Description |
|---------|-------|-------------|
| `serve` | `clustermind serve --port 8080 --namespace demo` | Start the REST API server |
| `analyze` | `clustermind analyze --namespace demo` | Print cluster structure + AI summary to terminal |
| `impact` | `clustermind impact --target Service/demo/payment-svc` | Run failure blast-radius analysis and save React Flow JSON |

---

## Quick Start & Demo Flow

### 1. Build the Binary

```bash
go mod tidy
make build
# → bin/clustermind
```

### 2. Install Frontend Dependencies

```bash
cd web && npm install && cd ..
```

### 3. Set Environment Variables

```bash
export GEMINI_API_KEY=AIza...          # Required for AI features (https://aistudio.google.com/apikey)
export KUBECONFIG=~/.kube/config      # kubeconfig path (default)
export CLUSTERMIND_NAMESPACE=demo     # Namespace to collect (empty = all)
export CLUSTERMIND_PORT=8080          # API server port
export CLUSTERMIND_LOG_LEVEL=INFO     # Log level: DEBUG|INFO|WARN|ERROR
```

### 4. Create a Local Demo Cluster (kind)

```bash
# Spins up a kind cluster and applies mock resources (no real cluster needed)
make demo
```

### 5. Start the Backend API Server

```bash
make run-backend
# → http://localhost:8080
```

### 6. Start the Frontend Dev Server

```bash
make run-frontend
# → http://localhost:5173
```

Open `http://localhost:5173` in your browser to see the interactive cluster graph:

```
┌─────────────────────────────────────────────────────────────────┐
|  ClusterMind   AI Infrastructure Digital Twin       [demo]   |
├──────────────┬──────────────────────────────────┬───────────────┤
| Resources    |                                  | Impact        |
|  Ingress (1) |   [ingress-nginx]                |  Simulator    |
│  Service (5) │          │                       │               │
|  Deployment  |   [frontend-svc]                 | Target:       |
│    (4)       │          │                       │ [Select ▼]   │
|              |   [frontend-deploy]               |               |
│              │          │                       │ [▶ Analyze]   │
|              |   [api-svc] ----> [payment-svc]  |               |
│              │          │           │            │               │
|              |   [api-deploy]    [payment]      |               |
└──────────────┴──────────────────────────────────┴───────────────┘
```

### 7. Simulate a Failure (Web UI)

```
1. Select a resource in the right-hand "Impact Simulator" panel
   e.g. [Service] payment-svc

2. Click [Analyze]
   → POST /api/v1/impact { "target": "Service/demo/payment-svc" }
   → BFS graph traversal (depth ≤ 5)
   → LLM severity & impact reasoning

3. React Flow graph updates with colour highlighting
   [CRITICAL] payment-svc          <- failure origin   (red border + animate-pulse)
   [DIRECT]   checkout-deployment  <- direct impact    (orange, animated dashed edge)
   [INDIRECT] order-svc            <- indirect impact  (yellow)
   [INDIRECT] notification-svc     <- indirect impact  (yellow)
   [OK]       frontend-svc         <- unaffected       (white)

4. AI natural-language explanation + severity badge appear in the right panel
```

### 8. CLI Blast-Radius Analysis

```bash
# Outputs propagation tree to terminal and saves React Flow JSON
./bin/clustermind impact --target Service/demo/payment-svc
```

```
[CRITICAL] Service/demo/payment-svc failure blast radius

├── Direct impact  (depth 1): 1 resource
│   └── Deployment/demo/payment-deployment
└── Indirect impact (depth 2+): 2 resources
    ├── Service/demo/auth-svc
    └── Service/demo/postgres-svc

=== AI Analysis ===
A failure in payment-svc will cause payment-deployment to fail on all
payment API calls, making order completion impossible. Consequently,
order-svc cannot update order status and notification-svc cannot send
payment confirmation alerts.

React Flow JSON → cluster_impact.json
```

---

## Test Results

All graph builder test specifications are implemented and passing:

```
=== RUN   TestGraphBuilder_LabelSelectorMatch    --- PASS
=== RUN   TestGraphBuilder_IngressToService      --- PASS
=== RUN   TestGraphBuilder_ServiceDeploymentMatch--- PASS
=== RUN   TestGraphBuilder_NodeIDFormat          --- PASS
PASS
ok  	github.com/clustermind/clustermind/tests	0.540s
```

Run tests at any time with:

```bash
go test -v ./...
```

---

## Safety & Design Principles

- **Read-only** — ClusterMind never writes to the cluster.
- **Secret safety** — `Secret.Data` is set to `nil` immediately after collection; values never appear in API responses.
- **Confidence filtering** — Edges with confidence < 0.3 are excluded from the graph.
- **BFS depth limit** — Traversal is capped at 5 hops to prevent runaway cycles.
- **Structured LLM input** — User input is never forwarded raw to the LLM; all context is JSON-serialised and truncated to ≤ 4 000 chars before sending.

---

## Directory Structure

```
clustermind/
├── cmd/clustermind/main.go        # CLI entrypoint (serve / analyze / impact)
├── internal/
│   ├── probe/                     # K8s resource collection
│   ├── graph/                     # Dependency edge inference
│   ├── ai/                        # LLM engine, prompts, impact analysis
│   ├── render/                    # React Flow JSON conversion
│   └── api/                       # Gin HTTP server + handlers + DTOs
├── web/src/
│   ├── components/                # ClusterGraph, ImpactPanel, ResourceTree, SeverityBadge
│   ├── hooks/                     # useCluster (React Query), useImpact (Zustand)
│   ├── api/client.ts              # Typed fetch wrappers
│   └── types/index.ts             # TypeScript types mirroring Go DTOs
├── tests/
│   ├── fixtures/mock_snapshot.go  # Mock cluster data
│   └── graph_builder_test.go      # Unit tests
└── demo/
    ├── mock_cluster.yaml          # Demo K8s resources
    └── run_demo.sh                # One-shot setup script
```

---

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/cluster/snapshot` | Resource count summary by kind |
| `GET` | `/api/v1/cluster/graph` | React Flow nodes + edges JSON |
| `POST` | `/api/v1/impact` | Failure blast-radius analysis |
| `GET` | `/api/v1/watch` | SSE stream for live cluster changes |
| `GET` | `/health` | Health check |

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.22 + Gin + client-go + Gemini API (`google.golang.org/genai`) |
| Frontend | TypeScript 5 + React 18 + React Flow v12 + ELK.js |
| Styling | Tailwind CSS |
| State | React Query (server) + Zustand (client) |
| Icons | lucide-react |

---

## Known Limitations & Future Roadmap

1. **ELK Auto-layout on Frontend**: The current backend returns a simple grid-based `position` for each node. Full ELK hierarchical auto-layout is intended to run client-side via `elkjs`; this integration is planned for the next release.
2. **NetworkPolicy Edge Analysis**: NetworkPolicy `podSelector` / `namespaceSelector` rules are collected but not yet reflected as edges in the dependency graph. Full NetworkPolicy edge inference is on the roadmap.
3. **Real Cluster SSE Watch**: The `/api/v1/watch` SSE endpoint currently polls on a fixed 30-second interval rather than using Kubernetes informers/watchers. Event-driven streaming via `client-go` informers is planned.
