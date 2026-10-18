# ClusterMind — AI Infrastructure Digital Twin

[English](./README.md) | [한국어](./README.ko.md)

[![Go Version](https://img.shields.io/badge/Go-1.24-blue.svg)](https://golang.org)
[![Gemini](https://img.shields.io/badge/Gemini-3.5--flash--lite-4285F4.svg)](https://aistudio.google.com)
[![Tests](https://img.shields.io/badge/Tests-passing-brightgreen.svg)](#test-results)
[![React Flow](https://img.shields.io/badge/React%20Flow-v12-purple.svg)](https://reactflow.dev)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](./LICENSE)

`clustermind` is an open-source AI digital twin system that **real-time analyzes Kubernetes clusters** and automatically generates topology diagrams, dependency graphs, and failure blast-radius reports.

It collects cluster resources (read-only), builds a dependency graph via label-selector and ingress inference, and uses an LLM to reason about architecture structure and failure propagation — visualized as an interactive React Flow graph in the browser.

---

## Live Demo

| | URL |
|---|---|
| **Intro page** | https://clustermind-intro-607541401069.asia-northeast3.run.app |
| **Demo app** | https://clustermind-607541401069.asia-northeast3.run.app |

The demo app runs on Cloud Run in **demo mode** (`CLUSTERMIND_DEMO=failure`): a virtual cluster with a broken `payment-deployment` (ImagePullBackOff), a derived sev1 incident timeline, and Gemini-powered investigation — no Kubernetes cluster required.

---

## Key Features

- **Read-Only Collection**: ClusterMind never writes to the cluster; all operations are strictly read-only.
- **Automatic Dependency Graph**: Infers edges from Label Selectors, Ingress backends, HPA targets, ConfigMap mounts, and name-prefix heuristics — with per-edge confidence scores.
- **AI Failure Impact Analysis**: BFS graph traversal (depth ≤ 5) feeds into an LLM that reasons about propagation paths, business consequences, and severity (critical / high / medium / low).
- **Interactive React Flow Graph**: Drag, zoom, minimap — failure nodes highlighted with animated colour coding in real time.
- **Per-Namespace Resource Status**: Every namespace is listed with a status rollup (Healthy / Degraded / Warning / Pending); expanding a namespace shows each resource's live health, ready ratio, and reason.
- **Node Detail Panel**: Click any node on the graph to open a right-side detail panel with live status, metadata, upstream/downstream dependencies, and a one-click "Analyze Impact" action.
- **Incident Timeline**: Kubernetes Warning events are correlated into sev1–sev3 incidents per resource, each with an evidence timeline (recent Warning events are required for otherwise-healthy resources).
- **AI Investigation Reports**: `POST /api/v1/investigate` runs a Gemini-powered root-cause investigation returning a summary, observed facts, ranked possible causes with confidence, evidence links, and kubectl next checks — with a deterministic fallback when no LLM key is configured.
- **Clusterless Demo Mode**: `CLUSTERMIND_DEMO=failure|healthy` runs the entire stack against an embedded virtual cluster — no kubeconfig or cluster required.
- **One-Shot Runner**: `./run.sh` builds the Go backend, installs frontend deps, and launches both servers concurrently.
- **Safe Secret Handling**: Kubernetes Secrets are never collected at all — no secret names or values are read, stored, or exposed.
- **SSE Live Watch**: Streams cluster change events every 30 seconds for always-current topology.
- **CLI + REST API**: `serve`, `analyze`, `impact` CLI commands backed by a Gin HTTP server.

---

## CLI Command Reference

| Command | Usage | Description |
|---------|-------|-------------|
| `serve` | `clustermind serve --port 8080 --namespace demo` | Start the REST API server |
| `analyze` | `clustermind analyze --namespace demo` | Print cluster structure + AI summary to terminal |
| `impact` | `clustermind impact --target Service/demo/payment-svc` | Run failure blast-radius analysis and save React Flow JSON |

Every command also accepts `--demo [healthy|failure]` to run against the embedded virtual cluster (no kubeconfig needed).

---

## Quick Start & Demo Flow

### Option A — One-Shot Runner (recommended)

```bash
# Builds backend + frontend deps, then starts both servers
./run.sh

# Restrict to a single namespace
./run.sh --namespace demo

# Custom port / skip npm install
./run.sh --port 9090 --skip-install
```

`./run.sh` defaults to **all namespaces** (including `default`). Backend runs on `:8080`, frontend on `:5173`.

### Option B — Manual Steps

#### 1. Build the Binary

```bash
go mod tidy
make build
# → bin/clustermind
```

#### 2. Install Frontend Dependencies

```bash
cd web && npm install && cd ..
```

#### 3. Set Environment Variables
Copy `.env.example` to `.env`:

```bash
# Backend / CLI settings (see .env.example)
export GEMINI_API_KEY=AIza...          # Required for AI features (https://aistudio.google.com/apikey)
export GEMINI_MODEL=gemini-3.5-flash-lite # Optional model override (default: gemini-3.5-flash-lite)
export KUBECONFIG=~/.kube/config      # kubeconfig path (default: ~/.kube/config)
export CLUSTERMIND_NAMESPACE=demo     # Dedicated demo namespace to collect
export CLUSTERMIND_PORT=8080          # API server port (default: 8080)
export CLUSTERMIND_DEMO=failure       # Clusterless demo mode: 1|healthy | failure (default: off)
export CLUSTERMIND_LOG_LEVEL=INFO     # Log level: DEBUG|INFO|WARN|ERROR
```

#### 4. Create a Local Demo Cluster (kind)

```bash
# Spins up a kind cluster and applies mock resources (no real cluster needed)
make demo
```

#### 5. Start the Backend API Server

```bash
make run-backend
# → http://localhost:8080
```

#### 6. Start the Frontend Dev Server

```bash
make run-frontend
# → http://localhost:5173
```

Open `http://localhost:5173` in your browser to see the interactive cluster graph. The left nav has four views — **Overview** (fleet stats + namespace panel), **Topology** (interactive graph), **Incidents** (timeline + impact), and **Investigation** (AI report). The diagram below shows the Topology view:

```
┌──────────────────────────────────────────────────────────────────────────┐
│  ClusterMind   AI Infrastructure Digital Twin              [ns badges]  │
├────────────────┬───────────────────────────────────┬─────────────────────┤
│ Namespaces     │                                   │ Details | Impact    │
│ ▸ default (12) │   [ingress-nginx]                 │ ── Details tab ──   │
│   🟢 Healthy   │          │                        │ Status: 🟢 Healthy  │
│ ▸ demo (9)     │   [frontend-svc]                  │ Ready:  3/3         │
│   🔴 Degraded  │          │                        │ Reason: Available   │
│                │   [frontend-deploy]               │                     │
│ (expand a ns   │          │                        │ Metadata            │
│  to see each   │   [api-svc] ────> [payment-svc]   │  Kind:  Service     │
│  resource +    │          │            │           │  NS:    demo        │
│  status dot)   │   [api-deploy]     [payment]      │                     │
│                │                                   │ Dependencies        │
│                │   (click a node → detail panel)   │  ↑ api-svc          │
│                │                                   │  ↓ payment-deploy   │
│                │                                   │ [▶ Analyze Impact]  │
└────────────────┴───────────────────────────────────┴─────────────────────┘
```

### 7. Explore & Simulate a Failure (Web UI)

```
Namespace panel (left):
  1. Expand a namespace to see all resources with live status dots
     🟢 Healthy  🔴 Degraded  🟡 Warning  🔵 Pending

Node detail panel (right — Details tab):
  2. Click any node on the graph
     → Status: live phase, ready ratio, reason, age
     → Metadata: kind, name, namespace, resource ID, replicas
     → Dependencies: upstream / downstream connections (clickable)
     → [Analyze Impact] button runs failure analysis directly

Impact simulator (right — Impact tab):
  3. Click [Analyze Impact] (or select a target and click Analyze)
     → POST /api/v1/impact { "target": "Service/demo/payment-svc" }
     → BFS graph traversal (depth ≤ 5)
     → LLM severity & impact reasoning

  4. React Flow graph updates with colour highlighting
     [CRITICAL] payment-svc          <- failure origin   (red border + animate-pulse)
     [DIRECT]   checkout-deployment  <- direct impact    (orange, animated dashed edge)
     [INDIRECT] order-svc            <- indirect impact  (yellow)
     [INDIRECT] notification-svc     <- indirect impact  (yellow)
     [OK]       frontend-svc         <- unaffected       (white)

  5. AI natural-language explanation + severity badge appear in the Impact tab
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

### 9. Incidents & Investigation (Web UI)

```
Incidents view (left nav → Incidents):
  1. sev1 / sev2 / sev3 incidents derived from K8s Warning events + resource health
  2. Click an incident → evidence timeline + automatic impact overlay on the graph
  3. Investigation panel → POST /api/v1/investigate
     → summary, observed facts, ranked possible causes (confidence),
       supporting evidence, kubectl next checks
     → deterministic fallback report when no GEMINI_API_KEY is set

Investigation view (left nav → Investigation):
  4. Full-page investigation report for the selected incident
```

---

## Test Results

All tests pass — covering graph builder edge inference, impact traversal, incident derivation, investigation fallback, demo collector, status normalization, and layout:

```
$ go test ./...
ok  	github.com/clustermind/clustermind/internal/ai
ok  	github.com/clustermind/clustermind/internal/api
ok  	github.com/clustermind/clustermind/internal/probe
ok  	github.com/clustermind/clustermind/internal/render
ok  	github.com/clustermind/clustermind/tests
```

Run tests at any time with:

```bash
go test -v ./...        # Go backend tests
cd web && npm test      # Frontend tests (node --test)
```

---

## Safety & Design Principles

- **Read-only** — ClusterMind never writes to the cluster.
- **Zero Secret Collection** — ClusterMind deliberately omits Kubernetes Secrets, ensuring sensitive values are never read, stored, or exposed.
- **Dedicated Namespace Scoping** — Workload collection and RBAC can be scoped to a dedicated demo namespace (e.g., `demo`).
- **Least-Privilege RBAC** — Cloud Run and GKE deployments use namespace-scoped Roles and short-lived tokens, avoiding cluster-admin or static secret tokens.
- **Confidence filtering** — Edges with confidence < 0.3 are excluded from the graph.
- **BFS depth limit** — Traversal is capped at 5 hops to prevent runaway cycles.
- **Structured LLM input** — Context is JSON-serialised with validation and deterministic fallback if the LLM is unreachable.

---

## Directory Structure

```
clustermind/
├── run.sh                          # One-shot build & run (backend + frontend)
├── deploycmd.sh                    # Cloud Run deploy — clusterless demo mode
├── cloudrundeploy.sh               # Cloud Run deploy — real remote GKE cluster
├── cmd/clustermind/main.go         # CLI entrypoint (serve / analyze / impact)
├── internal/
│   ├── probe/                      # K8s resource collection + status extraction
│   │   ├── collector.go
│   │   ├── demo.go                 # Clusterless demo-mode virtual collector
│   │   ├── snapshot.go
│   │   └── status.go              # Per-resource health normalization
│   ├── graph/                      # Dependency edge inference
│   ├── ai/                         # LLM engine, prompts, impact analysis
│   │   └── investigation.go        # Incident root-cause investigation report
│   ├── render/                     # React Flow JSON conversion
│   └── api/
│       ├── server.go               # Gin HTTP server
│       ├── handlers.go             # REST handlers
│       └── incidents.go            # K8s events → incidents/changes derivation
├── web/src/
│   ├── App.tsx                     # 4-view routing (Overview/Topology/Incidents/Investigation)
│   ├── components/
│   │   ├── cm/                     # Design system (AppShell, primitives, icons)
│   │   ├── TopologyGraph.tsx       # React Flow canvas + impact overlay
│   │   ├── NamespacePanel.tsx      # Namespace tree with status dots
│   │   ├── NodeDetailPanel.tsx     # Right-side node detail panel
│   │   ├── ImpactPanel.tsx         # Failure impact simulator
│   │   ├── IncidentTimeline.tsx    # Incident evidence timeline
│   │   ├── InvestigationPanel.tsx  # Investigation report (facts / causes / next checks)
│   │   └── SeverityBadge.tsx
│   ├── routes/                     # Overview / Topology / Incidents / Investigation views
│   ├── hooks/                      # useTwin, useImpact, useSelection, useNamespaces, useIncident, useInvestigate
│   ├── api/client.ts               # Typed fetch wrappers
│   └── types/index.ts              # TypeScript types mirroring Go DTOs
├── tests/
│   ├── fixtures/                   # Mock cluster data
│   └── *_test.go                   # Graph builder, impact, demo collector tests
├── docs/index.html                 # Bilingual intro page (Cloud Run via docs/cloudeRunDeploy.sh)
└── demo/
    ├── mock_cluster.yaml           # Demo K8s resources
    └── run_demo.sh                 # One-shot setup script
```

---

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/cluster/snapshot` | Resource count summary by kind |
| `GET` | `/api/v1/cluster/graph` | React Flow nodes + edges JSON |
| `GET` | `/api/v1/twin/snapshot` | Digital twin view model (nodes, edges, health, changes, incidents) |
| `GET` | `/api/v1/namespaces` | All namespaces with resource counts + status rollup |
| `GET` | `/api/v1/namespaces/:namespace/resources?kind=Pod` | Resources in a namespace with live health status |
| `POST` | `/api/v1/impact` | Failure blast-radius analysis |
| `POST` | `/api/v1/investigate` | Incident root-cause investigation report (Gemini-backed) |
| `GET` | `/api/v1/watch` | SSE stream for live cluster changes |
| `GET` | `/health` | Health check |

---

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go 1.24 + Gin + client-go + Gemini API (`google.golang.org/genai`) |
| Frontend | TypeScript 5 + React 18 + React Flow v12 + Vite |
| Styling | Tailwind CSS |
| State | React Query (server) + Zustand (client) |
| Icons | lucide-react |
| Deployment | Google Cloud Run + Artifact Registry + Secret Manager |

---

## Deploying to Cloud Run

| Script | Purpose |
|--------|---------|
| `./deploycmd.sh` | Clusterless demo deploy (`CLUSTERMIND_DEMO`, GEMINI_API_KEY → Secret Manager) |
| `PROJECT_ID=… KUBE_CONTEXT=… ./cloudrundeploy.sh` | Deploy against a real remote GKE cluster (kubeconfig → Secret Manager) |
| `./docs/cloudeRunDeploy.sh` | Deploy the bilingual intro page (`docs/`) |

```bash
# Live demo (clusterless failure scenario) — see Live Demo section above
PROJECT_ID=my-project ./deploycmd.sh
```

---

## Known Limitations & Future Roadmap

1. **ELK Auto-layout on Frontend**: The current backend returns a simple grid-based `position` for each node. Full ELK hierarchical auto-layout is intended to run client-side via `elkjs`; this integration is planned for the next release.
2. **NetworkPolicy Edge Analysis**: NetworkPolicy `podSelector` / `namespaceSelector` rules are collected but not yet reflected as edges in the dependency graph. Full NetworkPolicy edge inference is on the roadmap.
3. **Real Cluster SSE Watch**: The `/api/v1/watch` SSE endpoint currently polls on a fixed 30-second interval rather than using Kubernetes informers/watchers. Event-driven streaming via `client-go` informers is planned.
