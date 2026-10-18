#!/usr/bin/env bash
# deploycmd.sh — 실제 Cloud Run 배포 명령어 (docs 소개 페이지)
#
# 사용법 (어디서 실행해도 됩니다):
#   ./docs/deploycmd.sh
#   PROJECT_ID=xxx ./docs/deploycmd.sh
set -euo pipefail

# 스크립트가 속한 docs/ 디렉터리를 소스 컨텍스트로 사용
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── 1. 배포 (실제 사용하는 gcloud run deploy 명령어) ────────────────────────
gcloud run deploy clustermind-intro \
  --source="${SCRIPT_DIR}" \
  --project="${PROJECT_ID:-clusermind}" \
  --region="${REGION:-asia-northeast3}" \
  --platform=managed \
  --port=8080 \
  --cpu=1 \
  --memory="${MEMORY:-256Mi}" \
  --timeout=60 \
  --min-instances=0 \
  --max-instances=3 \
  --allow-unauthenticated

# ── 2. 배포 URL 출력 ────────────────────────────────────────────────────────
gcloud run services describe clustermind-intro \
  --project="${PROJECT_ID:-clusermind}" \
  --region="${REGION:-asia-northeast3}" \
  --format='value(status.url)'

# ── 3. 접속 검증 ────────────────────────────────────────────────────────────
URL="$(gcloud run services describe clustermind-intro \
  --project="${PROJECT_ID:-clusermind}" \
  --region="${REGION:-asia-northeast3}" \
  --format='value(status.url)')"
echo "GET ${URL}/ → $(curl -s -o /dev/null -w '%{http_code}' "${URL}/")"
