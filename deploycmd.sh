#!/usr/bin/env bash
# deploycmd.sh — ClusterMind (Go API + Web UI) Cloud Run 배포
#
# 실제 사용하는 gcloud 명령어 그대로입니다. 이번 세션에서 검증된 배포 흐름:
#   1. (선택) 필요한 GCP API 활성화
#   2. (선택) .env의 GEMINI_API_KEY를 Secret Manager에 저장
#   3. Cloud Build로 이미지 빌드 후 Cloud Run 배포 (데모 모드)
#   4. 배포 URL 출력
#
# 사용법:
#   ./deploycmd.sh
#   PROJECT_ID=my-project ./deploycmd.sh
#   CLUSTERMIND_DEMO=healthy ./deploycmd.sh          # 장애 없는 healthy 시나리오
#   SKIP_SECRET=1 ./deploycmd.sh                     # 시크릿 설정 건너뛰기
#
# 환경 변수 (기본값은 대회 데모 기준):
#   PROJECT_ID            GCP 프로젝트 ID        (default: clusermind)
#   REGION                Cloud Run 리전        (default: asia-northeast3)
#   SERVICE               Cloud Run 서비스명    (default: clustermind)
#   CLUSTERMIND_DEMO      데모 시나리오         (default: failure; healthy도 가능)
#   CLUSTERMIND_NAMESPACE 수집 네임스페이스     (default: demo)
#   MEMORY                컨테이너 메모리       (default: 512Mi)
#   SKIP_SECRET=1         GEMINI_API_KEY 시크릿 설정 생략
set -euo pipefail

PROJECT_ID="${PROJECT_ID:-clusermind}"
REGION="${REGION:-asia-northeast3}"
SERVICE="${SERVICE:-clustermind}"
DEMO="${CLUSTERMIND_DEMO:-failure}"
NAMESPACE="${CLUSTERMIND_NAMESPACE:-demo}"
MEMORY="${MEMORY:-512Mi}"
SKIP_SECRET="${SKIP_SECRET:-0}"
SECRET_NAME="clustermind-gemini"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── 1. GCP API 활성화 ───────────────────────────────────────────────────────
echo "==> Enable APIs (run, cloudbuild, artifactregistry, secretmanager)"
gcloud services enable \
  run.googleapis.com \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com \
  secretmanager.googleapis.com \
  --project="${PROJECT_ID}"

# ── 2. GEMINI_API_KEY → Secret Manager ─────────────────────────────────────
# 키는 .env 파일에만 두고(파일은 gitignore), 시크릿으로 배포 환경에 주입합니다.
if [[ "${SKIP_SECRET}" != "1" ]]; then
  ENV_FILE="${SCRIPT_DIR}/.env"
  if [[ -f "${ENV_FILE}" ]] && grep -q '^GEMINI_API_KEY=..*' "${ENV_FILE}"; then
    echo "==> Sync GEMINI_API_KEY from .env → Secret Manager (${SECRET_NAME})"
    if ! gcloud secrets describe "${SECRET_NAME}" --project="${PROJECT_ID}" >/dev/null 2>&1; then
      grep '^GEMINI_API_KEY=' "${ENV_FILE}" | cut -d= -f2- \
        | tr -d '\n' \
        | gcloud secrets create "${SECRET_NAME}" --data-file=- --project="${PROJECT_ID}"
    else
      grep '^GEMINI_API_KEY=' "${ENV_FILE}" | cut -d= -f2- \
        | tr -d '\n' \
        | gcloud secrets versions add "${SECRET_NAME}" --data-file=- --project="${PROJECT_ID}"
    fi
    # Cloud Run revision가 시크릿을 읽도록 실행 계정에 접근 권한 부여
    PROJECT_NUMBER="$(gcloud projects describe "${PROJECT_ID}" --format='value(projectNumber)')"
    gcloud secrets add-iam-policy-binding "${SECRET_NAME}" \
      --project="${PROJECT_ID}" \
      --member="serviceAccount:${PROJECT_NUMBER}-compute@developer.gserviceaccount.com" \
      --role="roles/secretmanager.secretAccessor" >/dev/null
    SET_SECRETS="--set-secrets=GEMINI_API_KEY=${SECRET_NAME}:latest"
  else
    echo "==> .env에 GEMINI_API_KEY가 없어 시크릿을 건너뜁니다 (fallback 분석만 동작)"
    SET_SECRETS=""
  fi
else
  SET_SECRETS=""
fi

# ── 3. 배포 (Cloud Build로 소스 빌드) ──────────────────────────────────────
echo "==> Deploy ${SERVICE} (demo=${DEMO}, namespace=${NAMESPACE})"
# shellcheck disable=SC2086
gcloud run deploy "${SERVICE}" \
  --source="${SCRIPT_DIR}" \
  --project="${PROJECT_ID}" \
  --region="${REGION}" \
  --platform=managed \
  --allow-unauthenticated \
  --port=8080 \
  --memory="${MEMORY}" \
  --min-instances=0 \
  --set-env-vars="CLUSTERMIND_DEMO=${DEMO},CLUSTERMIND_NAMESPACE=${NAMESPACE}" \
  ${SET_SECRETS}

# ── 4. 배포 URL 출력 ────────────────────────────────────────────────────────
URL="$(gcloud run services describe "${SERVICE}" \
  --project="${PROJECT_ID}" \
  --region="${REGION}" \
  --format='value(status.url)')"

echo ""
echo "Deployed: ${URL}"
echo ""
echo "검증 예시:"
echo "  curl ${URL}/health"
echo "  curl ${URL}/api/v1/twin/snapshot | python3 -m json.tool | head -20"
echo "  curl -X POST ${URL}/api/v1/investigate -H 'Content-Type: application/json' \\"
echo "    -d '{\"incidentId\":\"inc-deployment-demo-payment-deployment\"}'"
