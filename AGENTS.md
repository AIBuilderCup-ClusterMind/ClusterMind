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
│   │   ├── prompts.go              # 구조 설명 / 장애 추론 프롬프트 템플릿
│   │   └── impact.go               # 장애 영향 범위 분석 로직
│   ├── render/
│   │   └── graph.go               # React Flow용 nodes/edges JSON 변환
│   └── api/
│       ├── server.go               # Gin HTTP 서버
│       ├── handlers.go             # REST 엔드포인트 핸들러
│       └── dto.go                  # 요청/응답 DTO 구조체
├── web/                            # TypeScript + React 프론트엔드
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   └── src/
│       ├── main.tsx
│       ├── App.tsx
│       ├── components/
│       │   ├── ClusterGraph.tsx    # React Flow 그래프 캔버스 컴포넌트
│       │   ├── ImpactPanel.tsx     # 장애 영향 범위 패널
│       │   ├── ResourceTree.tsx    # 리소스 트리 사이드바
│       │   └── SeverityBadge.tsx   # critical/high/medium/low 뱃지
│       ├── hooks/
│       │   ├── useCluster.ts       # 클러스터 데이터 fetching
│       │   └── useImpact.ts        # 장애 시뮬레이션 상태
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

### 4. React Flow 그래프 렌더링 (`internal/render/graph.go` + `web/src/components/ClusterGraph.tsx`)

**백엔드 — nodes/edges JSON 변환 (`internal/render/graph.go`):**

#### React Flow 노드 구조 (Go → JSON → TypeScript)
```go
// Go 백엔드가 React Flow 호환 형식으로 직렬화
type FlowNode struct {
    ID       string         `json:"id"`
    Type     string         `json:"type"`    // "service" | "deployment" | "ingress" | ...
    Position Position       `json:"position"` // ELK가 자동 계산
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

**프론트엔드 — React Flow 커스텀 노드 (`web/src/components/ClusterGraph.tsx`):**

```tsx
// K8s 리소스 종류별 커스텀 노드 컴포넌트
const ServiceNode = ({ data }: NodeProps<FlowNodeData>) => (
  <div className={cn(
    "px-4 py-2 rounded-lg border-2 shadow-md min-w-[140px]",
    data.impactLevel === "critical" && "border-red-500 bg-red-50 animate-pulse",
    data.impactLevel === "direct"   && "border-orange-400 bg-orange-50",
    data.impactLevel === "indirect" && "border-yellow-400 bg-yellow-50",
    data.impactLevel === "none"     && "border-slate-200 bg-white",
  )}>
    <div className="flex items-center gap-2">
      <NetworkIcon className="w-4 h-4 text-blue-600" />
      <span className="font-mono text-sm font-semibold">{data.label}</span>
    </div>
    <span className="text-xs text-slate-500">{data.namespace}</span>
    <Handle type="target" position={Position.Top} />
    <Handle type="source" position={Position.Bottom} />
  </div>
)

// React Flow 캔버스 + ELK 자동 레이아웃
export const ClusterGraph = () => {
  const { nodes, edges } = useCluster()   // React Query로 /api/v1/cluster/graph fetch
  const { impactTarget } = useImpact()    // 장애 시뮬레이션 선택 상태

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      nodeTypes={{ service: ServiceNode, deployment: DeploymentNode, ingress: IngressNode }}
      fitView
    >
      <Background />
      <Controls />
      <MiniMap />
    </ReactFlow>
  )
}
```

#### 장애 영향 범위 색상 규칙
- **장애 원점**: 빨간색 테두리 + `animate-pulse` (`border-red-500 bg-red-50`)
- **직접 영향 (depth 1)**: 주황색 (`border-orange-400 bg-orange-50`)
- **간접 영향 (depth 2+)**: 노란색 (`border-yellow-400 bg-yellow-50`)
- **미영향**: 기본 흰색 (`border-slate-200 bg-white`)
- **전파 엣지**: `animated: true` → 점선 흐름 애니메이션

**출력 형식:**
```go
result := renderer.RenderGraph(graph, impactResult)
// result.Nodes  []FlowNode  // React Flow nodes 배열
// result.Edges  []FlowEdge  // React Flow edges 배열
// result.Layout string      // "elk-layered" | "elk-force"
```

---

## API 인터페이스

### REST API (Go/Gin)

```
GET  /api/v1/cluster/snapshot          # 전체 리소스 스냅샷 반환
GET  /api/v1/cluster/graph             # 의존성 그래프 (nodes + edges) 반환
GET  /api/v1/cluster/graph             # React Flow nodes+edges JSON 반환
POST /api/v1/impact                    # 장애 영향 분석 요청
     body: { "target": "Service/production/payment-svc" }
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
export OPENAI_API_KEY=sk-...
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
- **프레임워크**: React 18 + Vite
- **스타일**: Tailwind CSS
- **상태 관리**: React Query (서버 상태) + Zustand (클라이언트 상태)
- **그래프 시각화**: `@xyflow/react` (React Flow v12) + `@xyflow/elk` (ELK 자동 레이아웃)
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

---

## 개발 우선순위

| 단계 | 기능 | 담당 | 상태 |
|------|------|------|------|
| P0 | K8s 리소스 수집 (`probe/collector.go`) | Go | 미구현 |
| P0 | Label Selector 기반 edge 추론 (`graph/builder.go`) | Go | 미구현 |
| P0 | React Flow nodes/edges JSON 변환 (`render/graph.go`) | Go | 미구현 |
| P0 | REST API 서버 (`api/server.go`) | Go | 미구현 |
| P1 | LLM 기반 장애 영향 분석 (`ai/impact.go`) | Go | 미구현 |
| P1 | CLI (`cmd/clustermind/main.go`) | Go | 미구현 |
| P1 | Web UI 기본 구조 (ClusterGraph, ImpactPanel) | TypeScript | 미구현 |
| P1 | 시연용 mock 클러스터 + demo 스크립트 | YAML/sh | 미구현 |
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
    github.com/sashabaranov/go-openai v1.24.0
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
    "@xyflow/react": "^12.0.0",
    "@xyflow/elk": "^0.0.8",
    "lucide-react": "^0.400.0",
    "@tanstack/react-query": "^5.0.0",
    "zustand": "^4.5.0",
    "tailwindcss": "^3.4.0"
  },
  "devDependencies": {
    "typescript": "^5.4.0",
    "vite": "^5.0.0",
    "vitest": "^1.6.0"
  }
}
```

### 환경 변수

```bash
# 공통
KUBECONFIG=~/.kube/config        # kubeconfig 경로 (기본값)
OPENAI_API_KEY=sk-...            # LLM API 키

# 백엔드 (Go)
CLUSTERMIND_PORT=8080            # API 서버 포트
CLUSTERMIND_LOG_LEVEL=INFO       # 로그 레벨 (DEBUG/INFO/WARN/ERROR)
CLUSTERMIND_MAX_DEPTH=5          # 그래프 탐색 최대 깊이
CLUSTERMIND_MIN_CONFIDENCE=0.3   # 최소 edge confidence 임계값
CLUSTERMIND_NAMESPACE=           # 비어 있으면 전체 네임스페이스

# 프론트엔드 (Vite)
VITE_API_BASE_URL=http://localhost:8080   # 백엔드 API 주소
```
