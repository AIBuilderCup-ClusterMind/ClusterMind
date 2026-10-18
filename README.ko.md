# ClusterMind — AI 인프라 디지털 트윈

> Kubernetes 클러스터를 실시간으로 분석해 **구조도**, **의존성 그래프**, **장애 영향 범위**를 자동 생성하는 AI 인프라 디지털 트윈 시스템

**[English README →](./README.md)**

## 라이브 데모

| | 접속 주소 |
|---|---|
| **소개 페이지** | https://clustermind-intro-607541401069.asia-northeast3.run.app |
| **데모 앱** | https://clustermind-607541401069.asia-northeast3.run.app |

데모 앱은 Cloud Run에서 **데모 모드**(`CLUSTERMIND_DEMO=failure`)로 동작합니다: `payment-deployment` 장애(ImagePullBackOff)가 발생한 가상 클러스터에서 sev1 인시던트 타임라인 파생과 Gemini 조사 리포트를 확인할 수 있습니다 — Kubernetes 클러스터가 없어도 동작합니다.

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
│      Web UI (TypeScript / React Flow)                        │
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
| **네임스페이스별 리소스 상태** | 모든 네임스페이스를 Healthy/Degraded/Warning/Pending 상태 rollup과 함께 표시, 네임스페이스를 펼치면 각 리소스의 실시간 상태·ready 비율·사유 확인 가능 |
| **노드 상세 정보 패널** | 그래프에서 노드를 클릭하면 우측에 실시간 상태·메타데이터·상하류 의존관계·"Analyze Impact" 버튼이 표시되는 상세 패널 |
| **인시던트 타임라인** | K8s Warning 이벤트 + 리소스 health를 결합해 sev1~sev3 인시던트를 파생하고, 인시던트별 근거 타임라인 제공 |
| **AI 조사 리포트** | `POST /api/v1/investigate`로 Gemini 기반 근본 원인 조사 실행 — 요약·관찰 사실·신뢰도순 원인·근거·kubectl 점검 명령 제공 (LLM 키 미설정 시 결정적 fallback) |
| **클러스터리스 데모 모드** | `CLUSTERMIND_DEMO=failure|healthy` 설정만으로 kubeconfig 없이 내장 가상 클러스터로 전체 기능 동작 |
| **원샷 실행 스크립트** | `./run.sh` 한 번으로 Go 백엔드 빌드 + 프론트엔드 의존성 설치 + 두 서버 동시 기동 |
| **SSE 실시간 감시** | 30초 간격으로 클러스터 변경 스트리밍 |
| **CLI** | `serve`, `analyze`, `impact` 커맨드 제공 |

### API 엔드포인트

| Method | Path | 설명 |
|--------|------|------|
| `GET` | `/api/v1/cluster/snapshot` | 리소스 종류별 개수 요약 |
| `GET` | `/api/v1/cluster/graph` | React Flow nodes+edges JSON |
| `GET` | `/api/v1/twin/snapshot` | 디지털 트윈 스냅샷 (노드 상태/메트릭/엣지 + changes + incidents) |
| `GET` | `/api/v1/namespaces` | 전체 네임스페이스 + 리소스 수 + 상태 rollup |
| `GET` | `/api/v1/namespaces/:namespace/resources?kind=Pod` | 네임스페이스 내 리소스 목록 + 실시간 상태 |
| `POST` | `/api/v1/impact` | 장애 영향 범위 분석 |
| `POST` | `/api/v1/investigate` | 인시던트 근본 원인 조사 리포트 (Gemini + fallback) |
| `GET` | `/api/v1/watch` | SSE 스트림 (클러스터 변경 감지, 30초 폴링) |
| `GET` | `/health` | 헬스체크 |

### 수집 리소스 종류

`Namespace` · `Deployment` · `StatefulSet` · `DaemonSet` · `Pod` · `Service` · `Ingress` · `ConfigMap` · `HorizontalPodAutoscaler` · `NetworkPolicy`

> Secret은 보안·RBAC 이유로 **수집하지 않습니다** (이름도 읽지 않음).

---

## 기술 스택

| 레이어 | 기술 |
|--------|------|
| 백엔드 | Go 1.24 + Gin + client-go + Gemini API (`google.golang.org/genai`) |
| 프론트엔드 | TypeScript 5 + React 18 + React Router + React Flow v12 + Vite |
| 스타일링 | Tailwind CSS v4 (oklch 다크 토큰) |
| 상태 관리 | React Query (서버) + Zustand (클라이언트) |
| 아이콘 | lucide-react |
| 배포 | Google Cloud Run + Artifact Registry + Secret Manager |

---

## 디렉터리 구조

```
clustermind/
├── run.sh                          # 원샷 빌드 & 실행 (백엔드 + 프론트엔드)
├── deploycmd.sh                    # Cloud Run 배포 — 클러스터리스 데모 모드
├── cloudrundeploy.sh               # Cloud Run 배포 — 실제 원격 GKE 클러스터
├── cmd/clustermind/main.go         # CLI 진입점 (serve / analyze / impact)
├── internal/
│   ├── probe/
│   │   ├── collector.go            # K8s 리소스 수집
│   │   ├── demo.go                 # 클러스터리스 데모 모드 가상 컬렉터
│   │   ├── snapshot.go             # ClusterSnapshot 구조체
│   │   └── status.go               # 리소스별 상태 정규화
│   ├── graph/
│   │   ├── builder.go              # 의존성 Edge 추론
│   │   └── models.go               # ResourceNode / DependencyEdge 타입
│   ├── ai/
│   │   ├── engine.go               # LLM 호출 추상화
│   │   ├── prompts.go              # 프롬프트 템플릿
│   │   ├── impact.go               # BFS 탐색 + LLM 장애 분석
│   │   └── investigation.go        # 인시던트 근본 원인 조사 리포트
│   ├── render/
│   │   └── graph.go                # React Flow JSON 변환
│   └── api/
│       ├── server.go               # Gin HTTP 서버
│       ├── handlers.go             # REST 핸들러
│       ├── incidents.go            # K8s Events → incidents/changes 파생
│       └── dto.go                  # 요청/응답 DTO
├── web/src/
│   ├── App.tsx                     # 4뷰 라우팅 (Overview/Topology/Incidents/Investigation)
│   ├── components/
│   │   ├── cm/                     # 디자인 시스템 (AppShell, 프로시티브, 아이콘)
│   │   ├── TopologyGraph.tsx       # React Flow 캔버스 + impact 오버레이
│   │   ├── NamespacePanel.tsx      # 네임스페이스 트리 + 상태 점 표시
│   │   ├── NodeDetailPanel.tsx     # 우측 노드 상세 정보 패널
│   │   ├── ImpactPanel.tsx         # 장애 시뮬레이션 패널
│   │   ├── IncidentTimeline.tsx    # 인시던트 근거 타임라인
│   │   ├── InvestigationPanel.tsx  # 조사 리포트 (사실/원인/점검 명령)
│   │   └── SeverityBadge.tsx       # 심각도 뱃지
│   ├── routes/                     # Overview / Topology / Incidents / Investigation 뷰
│   ├── hooks/                      # useTwin, useImpact, useSelection, useNamespaces, useIncident, useInvestigate
│   ├── api/client.ts               # fetch 래퍼
│   └── types/index.ts              # Go DTO 미러링 TypeScript 타입
├── tests/
│   ├── fixtures/                   # 테스트용 mock 클러스터 데이터
│   └── *_test.go                   # 그래프 빌더 / 영향 분석 / 데모 컬렉터 테스트
├── docs/index.html                 # 한/영 인트로 페이지 (docs/cloudeRunDeploy.sh로 Cloud Run 배포)
└── demo/
    ├── mock_cluster.yaml           # 시연용 K8s 리소스 (Namespace + 5 Services + 4 Deployments + Ingress)
    └── run_demo.sh                 # 데모 환경 자동 구성 스크립트
```

---

## 사전 준비

| 도구 | 버전 | 설치 |
|------|------|------|
| Go | 1.24+ | https://go.dev/dl |
| Node.js | 18+ | https://nodejs.org |
| kubectl | 최신 | https://kubernetes.io/docs/tasks/tools |
| kind | 최신 (로컬 데모용) | https://kind.sigs.k8s.io |
| Gemini API Key | — | https://aistudio.google.com/apikey |

---

## 실행 방법

### 방법 A — 원샷 실행 스크립트 (권장)

```bash
# 백엔드 + 프론트엔드 의존성 빌드 후 두 서버 동시 기동
./run.sh

# 특정 네임스페이스만 수집
./run.sh --namespace demo

# 커스텀 포트 / npm install 생략
./run.sh --port 9090 --skip-install
```

`./run.sh`는 기본값으로 **전체 네임스페이스**(`default` 포함)를 수집합니다. 백엔드는 `:8080`, 프론트엔드는 `:5173`에서 기동됩니다.

### 방법 B — 수동 단계

#### 1. 레포 클론 및 빌드

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

#### 2. 환경 변수 설정

```bash
export GEMINI_API_KEY=AIza...                       # LLM API 키 (AI 기능 필수)
export GEMINI_MODEL=gemini-3.5-flash-lite           # 모델 오버라이드 (기본값)
export KUBECONFIG=~/.kube/config                    # kubeconfig 경로 (기본값)
export CLUSTERMIND_NAMESPACE=                       # 수집 네임스페이스 (빈값=전체)
export CLUSTERMIND_PORT=8080                        # API 서버 포트
export CLUSTERMIND_DEMO=failure                     # 클러스터리스 데모: 1|healthy | failure (기본값: off)
export CLUSTERMIND_LOG_LEVEL=INFO                   # 로그 레벨: DEBUG|INFO|WARN|ERROR
```

---

### 시나리오 A — 로컬 데모 (kind + mock 클러스터)

실제 클러스터 없이 mock 리소스로 전체 기능을 체험할 수 있습니다. kind나 kubeconfig가 없으면 **데모 모드**로 실행하세요:

```bash
# 클러스터 없이 내장 가상 클러스터로 실행 (가장 간편)
CLUSTERMIND_DEMO=failure ./run.sh
```

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

브라우저에서 `http://localhost:5173`에 접속하면 아래 화면이 표시됩니다. 좌측 내비게이션은 **Overview**(상태 요약) / **Topology**(인터랙티브 그래프) / **Incidents**(타임라인) / **Investigation**(AI 조사 리포트) 4개 뷰로 구성되며, 아래는 Topology 뷰입니다:

```
┌──────────────────────────────────────────────────────────────────────────┐
│  🧠 ClusterMind   AI Infrastructure Digital Twin           [ns badges]  │
├────────────────┬───────────────────────────────────┬─────────────────────┤
│ 네임스페이스    │                                   │ 상세 | 영향분석      │
│ ▸ default (12) │   [🌐 ingress-nginx]              │ ── 상세 탭 ──        │
│   🟢 Healthy   │          │                        │ 상태: 🟢 Healthy     │
│ ▸ demo (9)     │   [⚙️ frontend-svc]               │ Ready:  3/3          │
│   🔴 Degraded  │          │                        │ 사유: Available      │
│                │   [📦 frontend-deploy]            │                      │
│ (네임스페이스   │          │                        │ 메타데이터           │
│  펼치면 각     │   [⚙️ api-svc] ──▶ [⚙️ payment-svc]│  Kind:  Service     │
│  리소스 +      │          │           │            │  NS:    demo         │
│  상태 점 표시) │   [📦 api-deploy]  [📦 payment]   │                      │
│                │                                   │ 의존관계             │
│                │   (노드 클릭 → 상세 패널)          │  ↑ api-svc           │
│                │                                   │  ↓ payment-deploy    │
│                │                                   │ [▶ 영향 분석]        │
└────────────────┴───────────────────────────────────┴─────────────────────┘
```

---

### 시나리오 B — 실제 클러스터 연결

```bash
# kubeconfig가 실제 클러스터를 가리키는 상태에서 실행
export KUBECONFIG=~/.kube/config
export CLUSTERMIND_NAMESPACE=""   # 전체 네임스페이스 수집 (기본값)

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

> 모든 CLI 커맨드는 `--demo [healthy|failure]` 플래그를 지원합니다 — kubeconfig 없이 내장 가상 클러스터로 동작합니다.

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

### 시나리오 D — 인시던트 & AI 조사 (Web UI)

```
Incidents 뷰 (좌측 내비게이션 → Incidents):
  ① K8s Warning 이벤트 + 리소스 health로 sev1/sev2/sev3 인시던트 파생
  ② 인시던트 클릭 → 근거 타임라인 + 그래프에 자동 impact 오버레이
  ③ 조사 패널 → POST /api/v1/investigate
     → 요약 / 관찰 사실 / 신뢰도순 가능 원인 / 근거 / kubectl 점검 명령
     → GEMINI_API_KEY 미설정 시 결정적 fallback 리포트

Investigation 뷰 (좌측 내비게이션 → Investigation):
  ④ 선택된 인시던트의 전체 조사 리포트 페이지
```

---

## 배포 (Google Cloud Run)

| 스크립트 | 용도 |
|----------|------|
| `./deploycmd.sh` | 클러스터리스 데모 배포 (`CLUSTERMIND_DEMO`, GEMINI_API_KEY → Secret Manager) |
| `PROJECT_ID=… KUBE_CONTEXT=… ./cloudrundeploy.sh` | 실제 원격 GKE 클러스터 대상 배포 (kubeconfig → Secret Manager) |
| `./docs/cloudeRunDeploy.sh` | 한/영 인트로 페이지(`docs/`) 배포 |

```bash
# 라이브 데모 (클러스터리스 failure 시나리오) — 위 "라이브 데모" 섹션 참조
PROJECT_ID=my-project ./deploycmd.sh
```

---

## 장애 시뮬레이션 데모 (Web UI)

```
네임스페이스 패널 (좌측):
  ① 네임스페이스를 펼치면 리소스별 실시간 상태 점 확인
     🟢 Healthy  🔴 Degraded  🟡 Warning  🔵 Pending

노드 상세 패널 (우측 — 상세 탭):
  ② 그래프에서 노드 클릭
     → 상태: 실시간 phase, ready 비율, 사유, age
     → 메타데이터: kind, name, namespace, resource ID, replicas
     → 의존관계: 상하류 연결 목록 (클릭하여 해당 노드로 이동)
     → [영향 분석] 버튼으로 바로 장애 분석 실행

장애 시뮬레이터 (우측 — 영향분석 탭):
  ③ [영향 분석] 클릭 (또는 대상을 선택 후 Analyze)
     → POST /api/v1/impact { "target": "Service/demo/payment-svc" }
     → BFS 그래프 탐색 (depth ≤ 5)
     → LLM 심각도·영향 추론

  ④ 결과 — React Flow 그래프 색상 강조
     🔴 payment-svc          ← 장애 원점  (빨강 + animate-pulse)
     🟠 checkout-deployment  ← 직접 영향  (주황, 점선 애니메이션 엣지)
     🟡 order-svc            ← 간접 영향  (노랑)
     🟡 notification-svc     ← 간접 영향  (노랑)
     ⚪ frontend-svc         ← 미영향    (흰색)

  ⑤ AI 자연어 설명이 영향분석 탭에 표시됨
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
make test-coverage  # 커버리지 포함 테스트 실행
make tidy           # go mod tidy
make clean          # bin/, web/dist/ 삭제
```

---

## 설계 원칙

- **Read-only**: 클러스터에 쓰기 작업 일절 없음
- **Secret 미수집**: Kubernetes Secret은 보안·RBAC 이유로 애초에 수집하지 않음 — 이름도 값도 읽지 않음
- **신뢰도 필터링**: Confidence < 0.3인 Edge는 그래프에서 제외
- **깊이 제한**: BFS 탐색은 최대 5 hop으로 제한하여 순환 참조 방지
- **LLM 구조화**: 사용자 입력은 절대 LLM에 직접 전달하지 않고 JSON 직렬화 후 전달

---

## 테스트

```bash
go test ./...        # Go 백엔드 테스트 (그래프 빌더/영향 분석/인시던트/조사/데모 컬렉터 — 전부 통과)
cd web && npm test   # 프론트엔드 테스트 (node --test)
```

---

## 알려진 제한 사항 및 로드맵

1. **프론트엔드 ELK 자동 레이아웃**: 현재 백엔드는 노드별 그리드 기반 `position`을 반환합니다. `elkjs` 기반 계층 자동 레이아웃은 클라이언트 사이드 통합이 다음 릴리스 계획입니다.
2. **NetworkPolicy Edge 분석**: NetworkPolicy는 수집하지만 아직 의존성 그래프 edge로 반영되지 않습니다.
3. **실시간 SSE Watch**: `/api/v1/watch`는 30초 간격 폴링입니다. `client-go` informer 기반 이벤트 스트리밍은 계획 단계입니다.
