#!/usr/bin/env bash
# cloudeRunDeploy.sh — Deploy the ClusterMind intro page (docs/) to Cloud Run.
#
# What this script does:
#   1. Preflight checks (gcloud installed & authenticated)
#   2. Enables required GCP APIs (run, cloudbuild, artifactregistry)
#   3. Grants the default compute SA the permissions Cloud Build needs
#      (storage.objectAdmin + artifactregistry.writer — idempotent)
#   4. Builds via Cloud Build (Dockerfile in this directory) and deploys
#   5. Verifies the deployed URL (/ returns 200)
#
# Usage:
#   ./cloudeRunDeploy.sh
#
# Configuration (env vars):
#   PROJECT_ID   GCP project ID        (default: gcloud config project)
#   REGION       Cloud Run region      (default: asia-northeast3)
#   SERVICE      Cloud Run service name (default: clustermind-intro)
#   MEMORY       Memory limit          (default: 256Mi)
#   MIN_INSTANCES  Min instances        (default: 0, idle → no cost)
#   ALLOW_UNAUTH   Public access        (default: 1)
set -euo pipefail

REGION="${REGION:-asia-northeast3}"
SERVICE="${SERVICE:-clustermind-intro}"
MEMORY="${MEMORY:-256Mi}"
MIN_INSTANCES="${MIN_INSTANCES:-0}"
ALLOW_UNAUTH="${ALLOW_UNAUTH:-1}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

info() { printf '\n\033[1;34m▸\033[0m %s\n' "$*"; }
warn() { printf '\n\033[1;33m!\033[0m %s\n' "$*" >&2; }
fail() { printf '\n\033[1;31m✗\033[0m %s\n' "$*" >&2; exit 1; }

# ── Preflight ───────────────────────────────────────────────────────────────
command -v gcloud >/dev/null || fail "gcloud CLI not found: https://cloud.google.com/sdk/docs/install"
gcloud auth list --filter=status:ACTIVE --format='value(account)' 2>/dev/null | grep -q . \
  || fail "No active gcloud account. Run: gcloud auth login"

PROJECT_ID="${PROJECT_ID:-$(gcloud config get-value project 2>/dev/null || true)}"
[[ -n "${PROJECT_ID}" ]] || fail "PROJECT_ID not set (and no gcloud default project). Run: gcloud config set project <PROJECT_ID>"
gcloud config set project "${PROJECT_ID}" >/dev/null

[[ -f "${SCRIPT_DIR}/Dockerfile" ]] || fail "Dockerfile not found in ${SCRIPT_DIR}"
[[ -f "${SCRIPT_DIR}/index.html" ]] || fail "index.html not found in ${SCRIPT_DIR}"

info "Project: ${PROJECT_ID}  Region: ${REGION}  Service: ${SERVICE}"

# ── 1. Enable APIs ──────────────────────────────────────────────────────────
info "Enabling GCP APIs..."
gcloud services enable \
  run.googleapis.com \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com \
  --project="${PROJECT_ID}"

# ── 2. Cloud Build permissions for the default compute SA ───────────────────
PROJECT_NUMBER="$(gcloud projects describe "${PROJECT_ID}" --format='value(projectNumber)')"
BUILD_SA="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"

has_role() {
  local role="$1"
  gcloud projects get-iam-policy "${PROJECT_ID}" \
    --flatten="bindings[].members" \
    --filter="bindings.role:${role} AND bindings.members:serviceAccount:${BUILD_SA}" \
    --format='value(bindings.role)' 2>/dev/null | grep -q .
}

for role in roles/storage.objectAdmin roles/artifactregistry.writer; do
  if has_role "${role}"; then
    info "${role} already granted to ${BUILD_SA}"
  else
    info "Granting ${role} to ${BUILD_SA}..."
    gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
      --member="serviceAccount:${BUILD_SA}" \
      --role="${role}" \
      --condition=None >/dev/null
  fi
done

# ── 3. Deploy ───────────────────────────────────────────────────────────────
info "Deploying to Cloud Run (source build via Cloud Build)..."
DEPLOY_ARGS=(
  deploy "${SERVICE}"
  --source="${SCRIPT_DIR}"
  --region="${REGION}"
  --project="${PROJECT_ID}"
  --platform=managed
  --port=8080
  --memory="${MEMORY}"
  --cpu=1
  --timeout=60
  --min-instances="${MIN_INSTANCES}"
  --max-instances=3
)
if [[ "${ALLOW_UNAUTH}" == "1" ]]; then
  DEPLOY_ARGS+=(--allow-unauthenticated)
fi

gcloud run "${DEPLOY_ARGS[@]}"

# ── 4. Verify ───────────────────────────────────────────────────────────────
URL="$(gcloud run services describe "${SERVICE}" --region="${REGION}" --project="${PROJECT_ID}" --format='value(status.url)')"
info "Deployed ✓  ${URL}"

HTTP_CODE="$(curl -s -o /dev/null -w '%{http_code}' "${URL}/" || true)"
if [[ "${HTTP_CODE}" == "200" ]]; then
  info "Verification ✓  GET / → 200"
else
  warn "GET / returned ${HTTP_CODE:-none} — check: gcloud run services logs read ${SERVICE} --region=${REGION}"
fi
