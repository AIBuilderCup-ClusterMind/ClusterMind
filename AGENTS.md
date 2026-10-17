# ClusterMind — AI Infrastructure Digital Twin

## 프로젝트 개요

Kubernetes 클러스터를 실시간으로 분석해 **구조도**, **의존성 그래프**, **장애 영향 범위**를 자동 생성하는 AI 인프라 디지털 트윈 시스템.

### 핵심 가치
- **복잡한 클러스터 구조 파악 어려움** → AI가 리소스 관계를 자동 분석·시각화
- **장애 영향 범위 추론** → 특정 Service 장애 시 연결된 Deployment·외부 서비스까지 전파 경로 추적
- **React Flow + ELK 인터랙티브 그래프 + 장애 영향 범위 연결** → 드래그·줌·클릭 가능한 실시간 노드 그래프로 어디서든 렌더링 가능

---

## 아키텍처

```
┌──────────────────────────────────────────────────────────────┐
│                        ClusterMind                          │
│                                                              │
│  ┌──────────────┐   ┌──────────────┐   ┌────────────────┐  │
│  │  K8s Probe   │──▶│ Graph Builder│──▶│   AI Engine    │  │
│  │   (Go)       │   │   (Go)       │   │  (LLM via Go)  │  │
│  └──────────────┘   └──────────────┘   └────────────────┘  │
│         │                  │                    │           │
│         ▼                  ▼                    ▼           │
│  ┌──────────────┐   ┌──────────────┐   ┌────────────────┐  │
│  │ ClusterSnap  │   │  DepGraph    │   │  ReactFlow   │  │
│  │  (structs)   │   │  (nodes +    │   │  Impact JSON │  │
│  │              │   │   edges)     │   │                │  │
│  └──────────────┘   └──────────────┘   └────────────────┘  │
│                                                    │        │
│                          REST API (Go/Gin)         │        │
│                    ┌───────────────────────────────┘        │
│                    ▼                                        │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Web UI (TypeScript / React)            │   │
│  │   - 실시간 클러스터 구조도 (Mermaid 렌더링)           │   │
│  │   - 장애 시뮬레이션 패널                              │   │
│  │   - 영향 범위 인터랙티브 하이라이트                   │   │
│  └─────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

### 레이어 구성

| 레이어 | 역할 | 주요 기술 |
|--------|------|-----------|
| **K8s Probe** | 클러스터 리소스 수집 | Go + `client-go` |
| **Graph Builder** | 의존성 그래프 구성 | Go + `gonum/graph` |
| **AI Engine** | 구조 설명 + 장애 추론 | Go + OpenAI SDK |
| **REST API** | 프론트엔드 데이터 제공 | Go + Gin |
| **Web UI** | 인터랙티브 시각화 | TypeScript + React + React Flow + ELK |
| **Impact Analyzer** | 장애 영향 범위 계산 | Go BFS/DFS + LLM 추론 |

---

## 디렉터리 구조

```
clustermind/
├── AGENTS.md                       # 이 파일 — AI 에이전트 지침
├── go.mod
├── go.sum
├── Makefile
├── cmd/
│   └── clustermind/
│       └── main.go                 # CLI + 서버 진입점
├── internal/
│   ├── probe/
│   │   ├── collector.go            # K8s 리소스 수집 (Pod, Svc, Deploy, Ingress 등)
│   │   └── snapshot.go             # ClusterSnapshot 구조체
│   ├── graph/
│   │   ├── builder.go              # 리소스 간 edge 추론 (label selector 매칭 등)
│   │   └── models.go               # ResourceNode / DependencyEdge 타입
│   ├── ai/
│   │   ├── engine.go               # LLM 호출 추상화 (OpenAI 호환)
│   │   ├── prompts.go              # 구조 설명 / 장애 추론 / 조사 프롬프트 템플릿
│   │   ├── impact.go               # 장애 영향 범위 분석 로직 (BFS traverse 공용)
│   │   └── investigation.go        # 인시던트 조사 리포트 (Gemini + fallback)
│   ├── render/
│   │   └── graph.go               # React Flow용 nodes/edges JSON 변환
│   └── api/
│       ├── server.go               # Gin HTTP 서버
│       ├── handlers.go             # REST 엔드포인트 핸들러
│       ├── incidents.go            # K8s Events → changes/incidents 파생
│       └── dto.go                  # 요청/응답 DTO 구조체
├── web/                            # TypeScript + React 프론트엔드
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   └── src/
│       ├── main.tsx
│       ├── App.tsx                 # 4뷰 라우팅 (Overview/Topology/Incidents/Investigation)
│       ├── index.css               # Jexa07 oklch 토큰 + Tailwind v4 테마
│       ├── components/
│       │   ├── cm/                 # 디자인 시스템 (AppShell, primitives, kindIconFor)
│       │   ├── TopologyGraph.tsx   # React Flow 캔버스 (impact 오버레이 포함)
│       │   ├── NamespacePanel.tsx  # 네임스페이스 사이드바
│       │   ├── NodeDetailPanel.tsx # 리소스 상세 패널
│       │   ├── ImpactPanel.tsx     # 장애 영향 범위 패널
│       │   ├── IncidentTimeline.tsx    # 인시던트 근거 타임라인
│       │   ├── InvestigationPanel.tsx  # 조사 리포트 패널 (facts/causes/nextChecks)
│       │   └── SeverityBadge.tsx   # critical/high/medium/low 뱃지
│       ├── routes/
│       │   ├── Overview.tsx
│       │   ├── Topology.tsx
│       │   ├── Incidents.tsx       # 리스트 + 타임라인 + 자동 impact + 조사 패널
│       │   └── Investigation.tsx   # 단일 인시던트 조사 리포트
│       ├── hooks/
│       │   ├── useTwin.ts          # twin 스냅샷 fetching (30s 폴링)
│       │   ├── useImpact.ts        # 장애 시뮬레이션 상태 (Zustand)
│       │   ├── useSelection.ts     # 노드 선택 상태
│       │   ├── useIncident.ts      # 선택된 인시던트 ID
│       │   ├── useInvestigate.ts   # 조사 리포트 mutation
│       │   └── useNamespaces.ts
│       ├── api/
│       │   └── client.ts           # REST API 클라이언트 (fetch 래퍼)
│       └── types/
│           └── index.ts            # 공유 TypeScript 타입 (Go DTO 미러링)
├── tests/
│   ├── fixtures/                   # 테스트용 K8s 리소스 YAML 픽스처
│   ├── probe_test.go
│   ├── graph_builder_test.go
│   ├── impact_test.go
│   └── graph_test.go
└── demo/
    ├── mock_cluster.yaml           # 시연용 가상 클러스터 리소스
    └── run_demo.sh
```

---

## 기능 명세

### 1. 리소스 수집 (`internal/probe/collector.go`)

**수집 대상 리소스:**
- `Namespace`
- `Deployment` / `StatefulSet` / `DaemonSet`
- `Pod`
- `Service`
- `Ingress` / `IngressClass`
- `ConfigMap` / `Secret` (이름 참조만, 값 제외)
- `HorizontalPodAutoscaler`
- `NetworkPolicy`

**수집 방식:**
```go
// client-go 사용 (kubeconfig 또는 in-cluster)
// 전체 네임스페이스 또는 특정 네임스페이스 지정 가능
collector := probe.NewClusterCollector(clientset, "production")
snapshot, err := collector.Collect(ctx)
```

**출력:** `ClusterSnapshot` — 모든 리소스를 Kind별 슬라이스로 보유

---

### 2. 의존성 그래프 구성 (`internal/graph/builder.go`)

**Edge 추론 규칙 (우선순위 순):**

| 규칙 | 설명 |
|------|------|
| **Label Selector 매칭** | Service의 `spec.selector`와 Pod/Deployment `metadata.labels` 매칭 |
| **Ingress → Service** | `spec.rules[].http.paths[].backend.service.name` 참조 |
| **HPA → Deployment** | `spec.scaleTargetRef` 참조 |
| **NetworkPolicy 범위** | `podSelector` + `namespaceSelector`로 허용/차단 edge |
| **ConfigMap/Secret 마운트** | `volumes[]` 또는 `envFrom[]` 참조 |
| **Env URL 참조** | 컨테이너 env의 URL이 Service 이름을 가리키면 연결 (예: `http://payment-svc:8080`) — confidence: 0.7 |
| **이름 기반 추론** | 동일 prefix를 공유하는 리소스 간 약한 연결 (confidence: low) |

**그래프 모델:**
```go
type ResourceNode struct {
    ID        string            // "{Kind}/{namespace}/{name}"
    Kind      string            // "Service", "Deployment", ...
    Name      string
    Namespace string
    Labels    map[string]string
    Meta      map[string]any    // Kind별 추가 속성
}

type DependencyEdge struct {
    Source     string  // ResourceNode.ID
    Target     string  // ResourceNode.ID
    EdgeType   string  // "selector_match", "ingress_backend", "hpa_target", ...
    Confidence float64 // 0.0 ~ 1.0
    Direction  string  // "upstream" | "downstream" | "bidirectional"
}
```

---

### 3. AI 분석 엔진 (`internal/ai/engine.go`, `internal/ai/impact.go`)

#### 3-1. 구조 설명 생성

입력: `ClusterSnapshot` + `DependencyGraph`
출력: 자연어 클러스터 구조 요약 + 주요 서비스 흐름 설명

프롬프트 전략:
- 리소스 목록을 JSON으로 직렬화해 컨텍스트 제공
- "이 클러스터에서 외부 트래픽이 어떻게 흐르는지 설명하라" 형태
- 최대 토큰 제한을 위해 스냅샷을 요약(Summary) 형태로 압축

#### 3-2. 장애 영향 범위 분석

입력: `target_resource` (장애 대상), `DependencyGraph`
출력: 영향받는 리소스 목록 + 영향 경로 + 심각도

```go
// 사용 예시
impact, err := analyzer.AnalyzeFailure(ctx, AnalyzeRequest{
    Target: "Service/production/payment-svc",
    Graph:  depGraph,
})
// impact.AffectedResources []ResourceNode
// impact.PropagationPath   [][]string  // 경로별 체인
// impact.Severity          string      // "critical" | "high" | "medium" | "low"
// impact.Explanation       string      // AI 생성 자연어 설명
```

**분석 2단계:**
1. **그래프 탐색** — BFS로 downstream 리소스 탐색 (depth ≤ 5)
2. **AI 추론** — 탐색 결과를 LLM에게 전달, 비명시적 영향(SLA, 외부 API 의존 등) 추론

---

### 4. React Flow 그래프 렌더링 (`internal/render/graph.go` + `web/src/components/TopologyGraph.tsx`)

**백엔드 — nodes/edges JSON 변환 (`internal/render/graph.go`):**

#### React Flow 노드 구조 (Go → JSON → TypeScript)
```go
// Go 백엔드가 React Flow 호환 형식으로 직렬화
type FlowNode struct {
    ID       string         `json:"id"`
    Type     string         `json:"type"`    // "service" | "deployment" | "ingress" | ...
    Position Position       `json:"position"` // render.GridPositions가 백엔드에서 계산 (layered grid)
    Data     FlowNodeData   `json:"data"`
}

type FlowNodeData struct {
    Label       string  `json:"label"`
    Kind        string  `json:"kind"`
    Namespace   string  `json:"namespace"`
    ImpactLevel string  `json:"impactLevel"` // "critical"|"direct"|"indirect"|"none"
    Replicas    *int32  `json:"replicas,omitempty"`
}

type FlowEdge struct {
    ID           string  `json:"id"`
    Source       string  `json:"source"`
    Target       string  `json:"target"`
    EdgeType     string  `json:"type"`      // "smoothstep"
    Animated     bool    `json:"animated"`  // 장애 전파 엣지 시 true
    Label        string  `json:"label,omitempty"`
}
```

**프론트엔드 — React Flow 캔버스 (`web/src/components/TopologyGraph.tsx`):**

```tsx
// 단일 커스텀 노드 컴포넌트 — TwinNode 기반, impact 오버레이 포함
function ResourceNodeView({ data }: NodeProps<Node<ResourceNodeData>>) {
  const { resource: r, impactLevel } = data   // impactLevel: critical|direct|indirect|none
  const Icon = kindIconFor(r.kind)            // lucide-react K8s 아이콘
  return (
    <div className={cn('w-[196px] rounded-md border px-2.5 py-2 transition-all', impactBorder[impactLevel])}>
      <Handle type="target" position={Position.Top} />
      <span className={cn('size-1.5 rounded-full', statusDot[r.status])} />
      <span className="font-mono text-xs">{r.name}</span>
      <Handle type="source" position={Position.Bottom} />
    </div>
  )
}

// 노드/엣지 데이터는 useTwin() (30s 폴링) + useImpact()에서 파생
export function TopologyGraph() {
  const { data: twin } = useTwin()     // GET /api/v1/twin/snapshot
  const { result: impactResult } = useImpact()
  // twin.nodes/edges → ReactFlow nodes (position은 백엔드 grid 값 사용)
  return <ReactFlow nodes={nodes} edges={edges} nodeTypes={{ resource: ResourceNodeView }} fitView />
}
```

#### 장애 영향 범위 색상 규칙 (`--color-*` oklch 토큰 기반)
- **장애 원점 (critical)**: `border-critical bg-critical/10 animate-pulse`
- **직접 영향 (depth 1, direct)**: `border-warning bg-warning/10`
- **간접 영향 (depth 2+, indirect)**: `border-dashed border-warning/70 bg-warning/5`
- **미영향 (none)**: `border-border-strong bg-surface-2`
- **health 상태 점**: `healthy`→`bg-healthy`, `degraded`→`bg-warning`, `unhealthy`→`bg-critical`

**출력 형식:**
```go
result := renderer.RenderGraph(graph, impactResult)
// result.Nodes  []FlowNode  // React Flow nodes 배열 (position: GridPositions)
// result.Edges  []FlowEdge  // React Flow edges 배열
// twinSnapshot: TwinNode{status, metrics, labels, position, replicas} + TwinEdge{protocol}
```

---

### 5. 인시던트 파생 + 조사 리포트 (`internal/api/incidents.go`, `internal/ai/investigation.go`)

**인시던트 파생 (Collect → BuildTwinIncidents):**
1. 스냅샷의 Warning 이벤트를 `resolveEventResource`로 리소스에 매칭 (ReplicaSet·Pod 이름의 pod-template-hash 접미스를 제거해 Deployment까지 2단계 매칭)
2. health가 `unhealthy`/`degraded`인 리소스는 합성 alert 근거 1건 보장 (이벤트 TTL 1h로 타임라인이 비는 경우 대비)
3. severity: unhealthy→`sev1`, degraded→`sev2`, **healthy지만 Warning 이벤트 보유→`sev3`** (최신 Warning이 10분 이내일 때만 — 복구 후 잔류 방지), 같은 리소스의 근거는 최신순 병합
4. `ScalingReplicaSet` 같은 Normal 이벤트는 `changes`(최신순 20건)로 분리 — 인시던트가 아님

**조사 리포트 (`Analyzer.Investigate`):**
1. `traverse(g, target)`으로 blast-radius 재사용 + incident timeline 전달
2. Gemini에 구조화 JSON 요청 (`InvestigationPrompt`): summary / observedFacts / possibleCauses(confidence) / supportingEvidence(evidenceId 검증) / nextChecks
3. LLM 미사용·파싱 실패·요약 공백 시 결정적 fallback: 타임라인 사실 나열 + 실패 시그니처 추론(CrashLoopBackOff/OOMKilled/…) + kubectl 명령 nextChecks
4. `generatedBy`: "gemini" | "fallback" — 프론트 배지로 노출

**프론트:** Incidents 뷰(리스트+타임라인+자동 impact 오버레이+InvestigationPanel) → `useIncident` 선택 공유 → Investigation 뷰(선택 시 자동 실행)

---

## API 인터페이스

### REST API (Go/Gin)

```
GET  /api/v1/cluster/snapshot          # 리소스 카운트 요약
GET  /api/v1/cluster/graph             # React Flow nodes+edges JSON 반환
GET  /api/v1/twin/snapshot             # 디지털 트윈 스냅샷 (노드 status/metrics/labels + edges + changes + incidents)
GET  /api/v1/namespaces                # 네임스페이스별 리소스 카운트/상태 집계
GET  /api/v1/namespaces/:ns/resources  # 특정 네임스페이스 리소스 + 정규화 health 상태
POST /api/v1/impact                    # 장애 영향 분석 요청
     body: { "target": "Service/production/payment-svc" }
POST /api/v1/investigate               # 인시던트 조사 리포트 (Gemini + 결정적 fallback)
     body: { "incidentId": "inc-deployment-demo-web" }
     → InvestigationResult { summary, observedFacts, possibleCauses, supportingEvidence, nextChecks }
GET  /api/v1/watch                     # SSE 스트림 — 클러스터 변경 이벤트
```

### CLI 인터페이스

```bash
# 백엔드 API 서버 기동
clustermind serve --port 8080 --namespace production

# 전체 클러스터 구조 분석 (터미널 출력)
clustermind analyze --namespace production

# 특정 서비스 장애 시뮬레이션
clustermind impact --target Service/production/payment-svc

# 자연어 질문
clustermind ask "payment 서비스가 내려가면 어떤 서비스가 영향받아?"

# 클러스터 그래프 JSON 출력 (React Flow 호환)
clustermind diagram --format reactflow --output graph.json
```

---

## 시연 시나리오

### 핵심 시나리오: `payment-svc` 장애 전파 시각화

**시연용 mock 클러스터 구성 (`demo/mock_cluster.yaml`):**
```
[ingress-nginx]
    └──▶ [frontend-svc]
             └──▶ [frontend-deployment]
                      └──▶ [api-svc]
                               └──▶ [api-deployment]
                                        ├──▶ [payment-svc]  ⚠️ 장애 대상
                                        │        └──▶ [payment-deployment]
                                        │        └──▶ 외부 PG API
                                        └──▶ [auth-svc]
                                                 └──▶ [auth-deployment]
                                                          └──▶ [postgres-svc]
```

---

### 실제 실행 시나리오

#### Step 1 — 로컬 환경 준비

```bash
# 레포 클론 및 의존성 설치
git clone https://github.com/your-org/clustermind && cd clustermind

# Go 백엔드 빌드
go mod tidy
go build -o bin/clustermind ./cmd/clustermind

# 프론트엔드 의존성 설치
cd web && npm install && cd ..

# 환경 변수 설정
export GEMINI_API_KEY=AIza...
export KUBECONFIG=~/.kube/config   # 또는 demo mock 사용
```

#### Step 2 — Mock 클러스터로 데모 실행

```bash
# mock 클러스터 YAML을 로컬 kind 클러스터에 적용
kind create cluster --name clustermind-demo
kubectl apply -f demo/mock_cluster.yaml

# 백엔드 API 서버 기동 (8080 포트)
bin/clustermind serve --port 8080 --namespace demo

# 별도 터미널 — 프론트엔드 dev 서버 기동
cd web && npm run dev   # → http://localhost:5173
```

#### Step 3 — 클러스터 구조 확인 (Web UI)

```
브라우저에서 http://localhost:5173 접속

① 좌측 사이드바: Namespace/Kind 별 리소스 트리 확인
② 중앙 패널: React Flow 캔버스에 인터랙티브 클러스터 그래프 렌더링

   ELK Layered 알고리즘으로 계층 자동 정렬:
   [🌐 ingress-nginx]
        │
   [⚙️ frontend-svc] ──── [⚙️ api-svc]
        │                      │
   [📦 frontend]          [📦 api-deploy]

   → 드래그, 줌, 노드 클릭, 미니맵 기본 제공

③ 우측 패널: AI 생성 클러스터 구조 요약 텍스트
   예) "이 클러스터는 ingress-nginx를 통해 외부 트래픽을 수신하며
       frontend → api → payment/auth 계층 구조로 서비스가 연결됩니다."
```

#### Step 4 — 장애 시뮬레이션 (핵심 데모)

```
① Web UI 상단 「Impact Simulator」 패널 클릭
② 드롭다운에서 「Service / demo / payment-svc」 선택 후 [Analyze] 클릭

③ 백엔드 처리 흐름 (약 2~3초):
   POST /api/v1/impact  { "target": "Service/demo/payment-svc" }
    └─▶ BFS 그래프 탐색 (depth ≤ 5)
    └─▶ LLM 추론 (비명시적 외부 의존 포함)
    └─▶ ImpactResult JSON 반환

④ 결과 — React Flow 장애 영향 범위 그래프 (색상 강조 + 애니메이션)
   ┌──────────────────────────────────────────────────────┐
   │  🔴 payment-svc     ← 장애 원점  (빨강 + pulse 효과) │
   │  🟠 checkout-deploy ← 직접 영향  (주황, 점선 엣지)   │
   │  🟡 order-svc       ← 간접 영향  (노랑)              │
   │  🟡 notification-svc← 간접 영향  (노랑)              │
   │  ⚪ frontend-svc    ← 미영향     (기본색)             │
   └──────────────────────────────────────────────────────┘
   → 노드 클릭 시 상세 정보 팝업 / 엣지에 전파 방향 애니메이션

⑤ AI 생성 자연어 설명 (우측 패널):
   "payment-svc 장애 발생 시 checkout-deployment가 결제 API
    호출에 실패하여 주문 완료 불가 상태가 됩니다. 이로 인해
    order-svc의 주문 상태 업데이트와 notification-svc의 결제
    완료 알림 발송이 연쇄 실패합니다. 외부 PG 게이트웨이의
    콜백 수신도 불가하여 미결제 건이 누적될 수 있습니다."
   severity: CRITICAL
```

#### Step 5 — CLI 단독 실행

```bash
# 터미널에서 바로 영향 분석 결과 확인
bin/clustermind impact --target Service/demo/payment-svc

# 출력 예시:
# [CRITICAL] payment-svc 장애 영향 범위
# ├── 직접 영향 (depth 1)
# │   └── Deployment/demo/checkout-deployment
# └── 간접 영향 (depth 2)
#     ├── Service/demo/order-svc
#     └── Service/demo/notification-svc
#
# React Flow JSON → cluster_impact.json 저장됨

# 실시간 감시 모드 (30초 간격, 변경 감지 시 재분석)
bin/clustermind watch --interval 30 --namespace demo
```

---

## 구현 지침 (AI 에이전트용)

### 코드 품질 기준

**백엔드 (Go):**
- **언어**: Go 1.22+
- **타입**: 모든 exported 타입에 godoc 주석 필수
- **데이터 모델**: Go struct + JSON 태그, 인터페이스로 레이어 추상화
- **비동기**: goroutine + channel 또는 `errgroup` 사용
- **오류 처리**: `fmt.Errorf("context: %w", err)` 래핑, 빈 `if err != nil {}` 금지
- **로깅**: `log/slog` (표준 라이브러리), `log.Printf` 직접 사용 금지
- **테스트**: `testing` 패키지 + testify, 커버리지 ≥ 80%

**프론트엔드 (TypeScript):**
- **언어**: TypeScript 5.x (strict mode 필수)
- **프레임워크**: React 18 + Vite + React Router (Overview/Topology 4뷰 셸)
- **스타일**: Tailwind CSS v4 (`@tailwindcss/vite`, oklch 다크 토큰 — `src/index.css`의 `@theme inline` 정의)
- **디자인 시스템**: `components/cm/` (Panel, Button, StatusBadge, StateMessage, PageHeader + kindIconFor)
- **상태 관리**: React Query (서버 상태) + Zustand (클라이언트 상태)
- **그래프 시각화**: `@xyflow/react` (React Flow v12) — 노드 위치는 백엔드 grid 배치
- **아이콘**: `lucide-react` (K8s 리소스별 아이콘)
- **테스트**: Vitest + Testing Library

### 금지 사항

- Go: `interface{}` / `any` 남용, `panic()` 사용 (복구 불가 상황 제외)
- TypeScript: `any` 타입 사용, `as unknown as T` 패턴
- Secret 값 로깅 또는 응답 본문 포함
- 클러스터에 쓰기 작업 (read-only 원칙 엄수)
- 하드코딩된 네임스페이스 또는 리소스 이름

### LLM 호출 원칙

- 시스템 프롬프트에 항상 "Kubernetes 전문가" 페르소나 명시
- 사용자 입력을 그대로 LLM에 전달하지 말 것 (구조화 후 전달)
- 토큰 비용 절감을 위해 리소스 스냅샷은 요약 형태로 압축
- LLM 응답은 항상 JSON 파싱 후 `ImpactResult` / `DiagramResult` 구조체로 변환

### 그래프 탐색 원칙

- 최대 탐색 깊이: 5 hop
- Confidence < 0.3 인 edge는 기본적으로 탐색 제외 (옵션으로 포함 가능)
- 순환 의존성 감지 시 경고 로그 출력 후 탐색 중단
- **장애 전파 방향 (`traverse`)**:
  - `env_reference` / `ingress_backend` / `hpa_target`: **역방향** (callee 장애 → caller 영향, 예: payment-svc → api-deployment)
  - `selector_match` / `name_prefix`: **양방향** (Service↔Workload는 한 단위)

---

## 개발 우선순위

| 단계 | 기능 | 담당 | 상태 |
|------|------|------|------|
| P0 | K8s 리소스 수집 (`probe/collector.go`) | Go | 완료 |
| P0 | Label Selector 기반 edge 추론 (`graph/builder.go`) | Go | 완료 |
| P0 | React Flow nodes/edges JSON 변환 (`render/graph.go`) | Go | 완료 |
| P0 | REST API 서버 (`api/server.go`) | Go | 완료 |
| P1 | LLM 기반 장애 영향 분석 (`ai/impact.go`) | Go | 완료 |
| P1 | CLI (`cmd/clustermind/main.go`) | Go | 완료 |
| P1 | Web UI 기본 구조 (Overview/Topology + ImpactPanel) | TypeScript | 완료 |
| P1 | 인시던트 파생 (K8s Events → incidents/changes, `api/incidents.go`) | Go | 완료 |
| P1 | 조사 리포트 (`POST /api/v1/investigate`, `ai/investigation.go`) | Go | 완료 |
| P1 | Incidents/Investigation 뷰 (타임라인 + 리포트) | TypeScript | 완료 |
| P1 | 시연용 mock 클러스터 + demo 스크립트 | YAML/sh | 완료 |
| P2 | SSE 실시간 감시 스트림 | Go + TypeScript | 미구현 |
| P2 | NetworkPolicy edge 분석 | Go | 미구현 |
| P2 | 대화형 질문 모드 (Web UI 채팅 패널) | TypeScript | 미구현 |

---

## 환경 설정

```go
// go.mod 주요 의존성
require (
    k8s.io/client-go                  v0.29.0
    k8s.io/api                        v0.29.0
    github.com/gin-gonic/gin          v1.10.0
    google.golang.org/genai              v1.75.0
    gonum.org/v1/gonum                v0.15.0
    golang.org/x/sync                 v0.7.0   // errgroup
)
```

```json
// web/package.json 주요 의존성
{
  "dependencies": {
    "react": "^18.3.0",
    "react-dom": "^18.3.0",
    "react-router-dom": "^7.1.0",
    "@xyflow/react": "^12.0.0",
    "lucide-react": "^0.400.0",
    "@tanstack/react-query": "^5.0.0",
    "zustand": "^4.5.0"
  },
  "devDependencies": {
    "@tailwindcss/vite": "^4.0.0",
    "tailwindcss": "^4.0.0",
    "typescript": "^5.4.0",
    "vite": "^5.0.0"
  }
}
```

### 환경 변수

```bash
# 공통
KUBECONFIG=~/.kube/config        # kubeconfig 경로 (기본값)
GEMINI_API_KEY=AIza...           # LLM API 키 (https://aistudio.google.com/apikey)

# 백엔드 (Go)
CLUSTERMIND_PORT=8080            # API 서버 포트
CLUSTERMIND_LOG_LEVEL=INFO       # 로그 레벨 (DEBUG/INFO/WARN/ERROR)
CLUSTERMIND_MAX_DEPTH=5          # 그래프 탐색 최대 깊이
CLUSTERMIND_MIN_CONFIDENCE=0.3   # 최소 edge confidence 임계값
CLUSTERMIND_NAMESPACE=           # 비어 있으면 전체 네임스페이스

# 프론트엔드 (Vite)
VITE_API_BASE_URL=http://localhost:8080   # 백엔드 API 주소
```

---

## 배포 (Google Cloud Run)

**아키텍처 — 단일 컨테이너**: `Dockerfile`이 워드프론트(`web/dist`)를 바이너리에 embed(`webdist/` 패키지)해 Go 서버가 REST API + SPA를 동시 서빙한다. 서브라우더 경로(`/incidents` 등)는 index.html로 fallback, `/api/*` 미일치는 404 JSON.

**`cloudrundeploy.sh` (원커맨드 배포):**
```bash
PROJECT_ID=my-project GEMINI_API_KEY=AIza... ./cloudrundeploy.sh
```
1. API 활성화 (run/cloudbuild/artifactregistry/secretmanager)
2. Artifact Registry 저장소 생성
3. 타깃 클러스터에 read-only ServiceAccount(`view` ClusterRole) + 장기 토큰 Secret 생성 → 컨테이너 호환 kubeconfig 조립 (gcloud exec 인증 의존 없음)
4. kubeconfig → Secret Manager, (선택) GEMINI_API_KEY → Secret Manager
5. `gcloud run deploy --source .` (Cloud Build로 Docker 빌드) + `--set-secrets`로 kubeconfig 볼륨 마운트

**환경변수 (env):** `REGION`(기본 asia-northeast3), `SERVICE`(clustermind), `NAMESPACE`(수집 제한), `KUBE_CONTEXT`, `RUNTIME_SA`, `USE_RAW_KUBECONFIG=1`, `ALLOW_UNAUTH=0`

**제약:** Cloud Run에서 타깃 클러스터 API 서버에 도달 가능해야 함 (GKE public endpoint OK, private IP는 Serverless VPC Access 필요, localhost 불가 — 스크립트가 사전 검사). 서버는 `PORT` env도 인식한다.
