# ClusterMind — AI 인프라 디지털 트윈

> Kubernetes 클러스터를 실시간으로 분석해 **구조도**, **의존성 그래프**, **장애 영향 범위**를 자동 생성하는 AI 인프라 디지털 트윈 시스템

**[English README →](./README.md)**

```
┌──────────────────────────────────────────────────────────────┐
│                        ClusterMind                          │
│                                                              │
│  K8s Probe ──▶ Graph Builder ──▶ AI Engine                  │
│      │               │               │                      │
│  ClusterSnap     DepGraph       ReactFlow JSON               │
│                                       │                     │
│                     REST API (Go/Gin) │                     │
│                  ┌────────────────────┘                     │
│                  ▼                                          │
│      Web UI (TypeScript / React Flow + ELK)                 │
└──────────────────────────────────────────────────────────────┘
```

---

## 기능

### 핵심 기능

| 기능 | 설명 |
|------|------|
| **클러스터 리소스 수집** | Pod, Service, Deployment, Ingress, HPA 등 10종 수집 (read-only) |
| **의존성 그래프 자동 구성** | Label Selector·Ingress 백엔드·HPA·이름 prefix 기반 5가지 규칙으로 Edge 자동 추론 |
| **AI 구조 설명 생성** | LLM이 클러스터 아키텍처를 자연어로 요약 |
| **장애 영향 범위 분석** | BFS 탐색(depth ≤ 5) + LLM 추론으로 전파 경로·심각도 계산 |
| **React Flow 인터랙티브 그래프** | 드래그·줌·미니맵 가능한 실시간 노드 그래프 |
| **장애 색상 하이라이트** | critical(빨강+pulse) / direct(주황) / indirect(노랑) / none(흰색) |
| **SSE 실시간 감시** | 30초 간격으로 클러스터 변경 스트리밍 |
| **CLI** | `serve`, `analyze`, `impact` 커맨드 제공 |

### API 엔드포인트

| Method | Path | 설명 |
|--------|------|------|
| `GET` | `/api/v1/cluster/snapshot` | 리소스 종류별 개수 요약 |
| `GET` | `/api/v1/cluster/graph` | React Flow nodes+edges JSON |
| `POST` | `/api/v1/impact` | 장애 영향 범위 분석 |
| `GET` | `/api/v1/watch` | SSE 스트림 (클러스터 변경 감지) |
| `GET` | `/health` | 헬스체크 |

### 수집 리소스 종류

`Namespace` · `Deployment` · `StatefulSet` · `DaemonSet` · `Pod` · `Service` · `Ingress` · `ConfigMap` · `Secret`(이름만, 값 제외) · `HorizontalPodAutoscaler` · `NetworkPolicy`

---

## 기술 스택

| 레이어 | 기술 |
|--------|------|
| 백엔드 | Go 1.22 + Gin + client-go + Gemini API (`google.golang.org/genai`) |
| 프론트엔드 | TypeScript 5 + React 18 + React Flow v12 + ELK.js |
| 스타일링 | Tailwind CSS |
| 상태 관리 | React Query (서버) + Zustand (클라이언트) |
| 아이콘 | lucide-react |

---

## 디렉터리 구조

```
clustermind/
├── cmd/clustermind/main.go        # CLI 진입점 (serve / analyze / impact)
├── internal/
│   ├── probe/
│   │   ├── collector.go           # K8s 리소스 수집
│   │   └── snapshot.go            # ClusterSnapshot 구조체
│   ├── graph/
│   │   ├── builder.go             # 의존성 Edge 추론
│   │   └── models.go              # ResourceNode / DependencyEdge 타입
│   ├── ai/
│   │   ├── engine.go              # LLM 호출 추상화
│   │   ├── prompts.go             # 프롬프트 템플릿
│   │   └── impact.go              # BFS 탐색 + LLM 장애 분석
│   ├── render/
│   │   └── graph.go               # React Flow JSON 변환
│   └── api/
│       ├── server.go              # Gin HTTP 서버
│       ├── handlers.go            # REST 핸들러
│       └── dto.go                 # 요청/응답 DTO
├── web/src/
│   ├── components/
│   │   ├── ClusterGraph.tsx       # React Flow 캔버스 + 커스텀 노드
│   │   ├── ImpactPanel.tsx        # 장애 시뮬레이션 패널
│   │   ├── ResourceTree.tsx       # 리소스 사이드바
│   │   └── SeverityBadge.tsx      # 심각도 뱃지
│   ├── hooks/
│   │   ├── useCluster.ts          # 그래프 데이터 fetching (30초 폴링)
│   │   └── useImpact.ts           # 장애 시뮬레이션 상태
│   ├── api/client.ts              # fetch 래퍼
│   └── types/index.ts             # Go DTO 미러링 TypeScript 타입
├── tests/
│   ├── fixtures/mock_snapshot.go  # 테스트용 mock 클러스터 데이터
│   └── graph_builder_test.go      # 그래프 빌더 유닛 테스트
└── demo/
    ├── mock_cluster.yaml          # 시연용 K8s 리소스 (Namespace + 5 Services + 4 Deployments + Ingress)
    └── run_demo.sh                # 데모 환경 자동 구성 스크립트
```

---

## 사전 준비

| 도구 | 버전 | 설치 |
|------|------|------|
| Go | 1.22+ | https://go.dev/dl |
| Node.js | 18+ | https://nodejs.org |
| kubectl | 최신 | https://kubernetes.io/docs/tasks/tools |
| kind | 최신 (로컬 데모용) | https://kind.sigs.k8s.io |
| Gemini API Key | — | https://aistudio.google.com/apikey |

---

## 실행 방법

### 1. 레포 클론 및 빌드

```bash
git clone https://github.com/your-org/clustermind
cd clustermind

# Go 의존성 설치 및 빌드
go mod tidy
make build
# → bin/clustermind 생성

# 프론트엔드 의존성 설치
cd web && npm install && cd ..
```

### 2. 환경 변수 설정

```bash
export GEMINI_API_KEY=AIza...                       # LLM API 키 (AI 기능 필수)
export KUBECONFIG=~/.kube/config                    # kubeconfig 경로 (기본값)
export CLUSTERMIND_NAMESPACE=demo                   # 수집 네임스페이스 (빈값=전체)
export CLUSTERMIND_PORT=8080                        # API 서버 포트
export CLUSTERMIND_LOG_LEVEL=INFO                   # 로그 레벨: DEBUG|INFO|WARN|ERROR
```

---

### 시나리오 A — 로컬 데모 (kind + mock 클러스터)

실제 클러스터 없이 mock 리소스로 전체 기능을 체험할 수 있습니다.

```bash
# 1. kind 클러스터 생성 + mock 리소스 적용
make demo
# → kind create cluster --name clustermind-demo
# → kubectl apply -f demo/mock_cluster.yaml (Namespace, 5 Services, 4 Deployments, Ingress)

# 2. 백엔드 API 서버 기동 (터미널 1)
make run-backend
# → http://localhost:8080

# 3. 프론트엔드 dev 서버 기동 (터미널 2)
make run-frontend
# → http://localhost:5173
```

브라우저에서 `http://localhost:5173` 접속하면 아래 화면이 표시됩니다:

```
┌─────────────────────────────────────────────────────────────────┐
│  🧠 ClusterMind   AI Infrastructure Digital Twin       [demo]   │
├──────────────┬──────────────────────────────────┬───────────────┤
│ Resources    │                                  │ ⚠️ Impact     │
│  Ingress (1) │   [🌐 ingress-nginx]             │  Simulator    │
│  Service (5) │          │                       │               │
│  Deployment  │   [⚙️ frontend-svc]              │ Target:       │
│    (4)       │          │                       │ [Select ▼]   │
│              │   [📦 frontend-deploy]            │               │
│              │          │                       │ [▶ Analyze]   │
│              │   [⚙️ api-svc] ──▶ [⚙️ payment-svc]│            │
│              │          │           │            │               │
│              │   [📦 api-deploy]  [📦 payment]  │               │
└──────────────┴──────────────────────────────────┴───────────────┘
```

---

### 시나리오 B — 실제 클러스터 연결

```bash
# kubeconfig가 실제 클러스터를 가리키는 상태에서 실행
export KUBECONFIG=~/.kube/config
export CLUSTERMIND_NAMESPACE=""   # 전체 네임스페이스 수집

make run-backend
make run-frontend
```

---

### 시나리오 C — CLI 단독 사용

```bash
# 클러스터 구조 전체 분석 (터미널 출력 + AI 요약)
./bin/clustermind analyze --namespace demo

# 장애 영향 범위 시뮬레이션
./bin/clustermind impact --target Service/demo/payment-svc
```

**`impact` 명령 출력 예시:**
```
[CRITICAL] Service/demo/payment-svc 장애 영향 범위

├── 직접 영향 (depth 1): 1개
│   └── Deployment/demo/payment-deployment
└── 간접 영향 (depth 2+): 2개
    └── Service/demo/auth-svc
    └── Service/demo/postgres-svc

=== AI 분석 ===
payment-svc 장애 발생 시 payment-deployment가 결제 API 호출에 실패하여
주문 완료 불가 상태가 됩니다. 이로 인해 order-svc의 주문 상태 업데이트와
notification-svc의 결제 완료 알림 발송이 연쇄 실패합니다.

React Flow JSON → cluster_impact.json 저장됨
```

---

## 장애 시뮬레이션 데모 (Web UI)

```
① 우측 「Impact Simulator」 패널에서 장애 대상 선택
   예) [Service] payment-svc

② [▶ Analyze] 클릭
   → POST /api/v1/impact { "target": "Service/demo/payment-svc" }
   → BFS 그래프 탐색 (depth ≤ 5)
   → LLM 심각도·영향 추론

③ 결과 — React Flow 그래프 색상 강조
   🔴 payment-svc          ← 장애 원점  (빨강 + animate-pulse)
   🟠 checkout-deployment  ← 직접 영향  (주황, 점선 애니메이션 엣지)
   🟡 order-svc            ← 간접 영향  (노랑)
   🟡 notification-svc     ← 간접 영향  (노랑)
   ⚪ frontend-svc         ← 미영향    (흰색)

④ AI 자연어 설명이 우측 패널에 표시됨
   severity: CRITICAL / HIGH / MEDIUM / LOW
```

---

## Makefile 명령 목록

```bash
make build          # Go 백엔드 바이너리 빌드 → bin/clustermind
make run-backend    # 백엔드 서버 기동 (포트 8080)
make run-frontend   # 프론트엔드 dev 서버 기동 (포트 5173)
make demo           # kind 클러스터 생성 + mock 리소스 적용
make test           # Go 유닛 테스트 실행
make tidy           # go mod tidy
make clean          # bin/, web/dist/ 삭제
```

---

## 설계 원칙

- **Read-only**: 클러스터에 쓰기 작업 일절 없음
- **Secret 보호**: Secret Data 필드는 수집 즉시 nil 처리 — 값이 API 응답에 포함되지 않음
- **신뢰도 필터링**: Confidence < 0.3인 Edge는 그래프에서 제외
- **깊이 제한**: BFS 탐색은 최대 5 hop으로 제한하여 순환 참조 방지
- **LLM 구조화**: 사용자 입력은 절대 LLM에 직접 전달하지 않고 JSON 직렬화 후 전달
