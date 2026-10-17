#!/usr/bin/env bash
# cloudrundeploy.sh — Deploy ClusterMind (Go API + embedded Web UI) to Cloud Run.
#
# What this script does:
#   1. Enables required GCP APIs
#   2. Creates an Artifact Registry repo (if missing)
#   3. Builds a container-safe kubeconfig (SA token, no gcloud dependency)
#      and stores it in Secret Manager
#   4. (Optional) stores GEMINI_API_KEY in Secret Manager
#   5. Builds the image via Cloud Build (Dockerfile at repo root) and
#      deploys it to Cloud Run with the kubeconfig mounted as a volume
#
# Usage:
#   PROJECT_ID=my-project ./cloudrundeploy.sh
#
# Configuration (env vars):
#   PROJECT_ID          GCP project ID (required; falls back to gcloud config)
#   REGION              Cloud Run region            (default: asia-northeast3)
#   SERVICE             Cloud Run service name      (default: clustermind)
#   NAMESPACE           Restrict collection to one K8s namespace (optional)
#   GEMINI_API_KEY      Enables AI features; stored in Secret Manager
#   KUBE_CONTEXT        kubectl context to target   (default: current context)
#   RUNTIME_SA          Runtime service account email (default: compute SA)
#   USE_RAW_KUBECONFIG=1  Upload ~/.kube/config as-is (only if it has no
#                         exec/plugin auth — e.g. plain token/cert auth)
#   ALLOW_UNAUTH=0      Deploy as private (default: public for demo)
#
# Notes:
#   * The target cluster's API server MUST be reachable from Cloud Run.
#     GKE public endpoint works; private IP / localhost does not (needs a
#     Serverless VPC Access connector — out of scope for this script).
#   * Step 3 creates a read-only ServiceAccount (ClusterRole "view") in the
#     target cluster and a long-lived token Secret for it.
set -euo pipefail

REGION="${REGION:-asia-northeast3}"
SERVICE="${SERVICE:-clustermind}"
ALLOW_UNAUTH="${ALLOW_UNAUTH:-1}"
USE_RAW_KUBECONFIG="${USE_RAW_KUBECONFIG:-0}"

info() { printf '\n\033[1;34m▸\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m! %s\033[0m\n' "$*" >&2; }
fail() { printf '\033[1;31m✗ %s\033[0m\n' "$*" >&2; exit 1; }

# ── Preflight ───────────────────────────────────────────────────────────────
command -v gcloud >/dev/null || fail "gcloud CLI not found: https://cloud.google.com/sdk/docs/install"
command -v kubectl >/dev/null || fail "kubectl not found"

PROJECT_ID="${PROJECT_ID:-$(gcloud config get-value project 2>/dev/null || true)}"
[[ -n "${PROJECT_ID}" ]] || fail "PROJECT_ID not set (and no gcloud default project)"

info "Project: ${PROJECT_ID}  Region: ${REGION}  Service: ${SERVICE}"
gcloud config set project "${PROJECT_ID}" >/dev/null

# ── 1. Enable APIs ──────────────────────────────────────────────────────────
info "Enabling GCP APIs..."
gcloud services enable \
  run.googleapis.com \
  cloudbuild.googleapis.com \
  artifactregistry.googleapis.com \
  secretmanager.googleapis.com \
  --project="${PROJECT_ID}"

# ── 2. Artifact Registry repo ──────────────────────────────────────────────
info "Ensuring Artifact Registry repo 'clustermind'..."
gcloud artifacts repositories describe clustermind \
  --repository-format=docker --location="${REGION}" --project="${PROJECT_ID}" >/dev/null 2>&1 \
  || gcloud artifacts repositories create clustermind \
       --repository-format=docker --location="${REGION}" --project="${PROJECT_ID}"

# ── 3. Kubeconfig for Cloud Run ─────────────────────────────────────────────
KUBECONFIG_OUT="$(mktemp -d)/kubeconfig.yaml"
cleanup() { rm -rf "$(dirname "${KUBECONFIG_OUT}")"; }
trap cleanup EXIT

if [[ "${USE_RAW_KUBECONFIG}" == "1" ]]; then
  warn "USE_RAW_KUBECONFIG=1 — uploading kubeconfig as-is."
  warn "It must NOT use exec/plugin auth (gcloud exec will fail inside the container)."
  KUBECONFIG_SRC="${KUBECONFIG:-${HOME}/.kube/config}"
  [[ -f "${KUBECONFIG_SRC}" ]] || fail "kubeconfig not found: ${KUBECONFIG_SRC}"
  cp "${KUBECONFIG_SRC}" "${KUBECONFIG_OUT}"
else
  info "Building a container-safe kubeconfig (ServiceAccount token)..."
  KUBE_CONTEXT="${KUBE_CONTEXT:-$(kubectl config current-context)}"

  SERVER="$(kubectl config view --minify --context="${KUBE_CONTEXT}" -o jsonpath='{.clusters[0].cluster.server}')"
  CA_DATA="$(kubectl config view --minify --context="${KUBE_CONTEXT}" --raw -o jsonpath='{.clusters[0].cluster.certificate-authority-data}')"

  case "${SERVER}" in
    https://127.*|https://localhost*)
      fail "Cluster API server is local (${SERVER}) — unreachable from Cloud Run. Target a remote cluster (e.g. GKE)."
      ;;
    https://10.*|https://192.168.*|https://172.1[6-9].*|https://172.2[0-9].*|https://172.3[01].*)
      warn "Cluster API server uses a private IP (${SERVER})."
      warn "Cloud Run needs a Serverless VPC Access connector to reach it."
      ;;
  esac

  SA_NAME="clustermind-reader"
  kubectl create serviceaccount "${SA_NAME}" -n default \
    --dry-run=client -o yaml | kubectl apply -f - >/dev/null
  kubectl create clusterrolebinding "${SA_NAME}" \
    --clusterrole=view --serviceaccount="default:${SA_NAME}" \
    --dry-run=client -o yaml | kubectl apply -f - >/dev/null
  kubectl apply -f - >/dev/null <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: ${SA_NAME}-token
  namespace: default
  annotations:
    kubernetes.io/service-account.name: ${SA_NAME}
type: kubernetes.io/service-account-token
EOF

  TOKEN=""
  for _ in $(seq 1 15); do
    TOKEN="$(kubectl get secret "${SA_NAME}-token" -n default -o jsonpath='{.data.token}' 2>/dev/null | base64 -d || true)"
    [[ -n "${TOKEN}" ]] && break
    sleep 1
  done
  [[ -n "${TOKEN}" ]] || fail "timed out waiting for SA token secret"

  cat > "${KUBECONFIG_OUT}" <<EOF
apiVersion: v1
kind: Config
clusters:
- name: clustermind-target
  cluster:
    server: ${SERVER}
    certificate-authority-data: ${CA_DATA}
users:
- name: ${SA_NAME}
  user:
    token: ${TOKEN}
contexts:
- name: clustermind-target
  context:
    cluster: clustermind-target
    user: ${SA_NAME}
current-context: clustermind-target
EOF
  info "kubeconfig ready (server: ${SERVER}, user: default/${SA_NAME}, role: view)"
fi

# ── 4. Secrets ──────────────────────────────────────────────────────────────
KUBE_SECRET="${SERVICE}-kubeconfig"
info "Storing kubeconfig in Secret Manager (${KUBE_SECRET})..."
if gcloud secrets describe "${KUBE_SECRET}" --project="${PROJECT_ID}" >/dev/null 2>&1; then
  gcloud secrets versions add "${KUBE_SECRET}" --data-file="${KUBECONFIG_OUT}" --project="${PROJECT_ID}" >/dev/null
else
  gcloud secrets create "${KUBE_SECRET}" --data-file="${KUBECONFIG_OUT}" --project="${PROJECT_ID}"
fi

# --set-secrets mounts the kubeconfig as a file and (optionally) exposes the
# Gemini key as an env var in a single flag.
KUBE_MOUNT_PATH="/etc/clustermind/${KUBE_SECRET}"
SECRETS_LIST="${KUBE_MOUNT_PATH}=${KUBE_SECRET}:latest"
if [[ -n "${GEMINI_API_KEY:-}" ]]; then
  GEMINI_SECRET="${SERVICE}-gemini-api-key"
  info "Storing GEMINI_API_KEY in Secret Manager (${GEMINI_SECRET})..."
  printf '%s' "${GEMINI_API_KEY}" > "${KUBECONFIG_OUT}.gemini"
  if gcloud secrets describe "${GEMINI_SECRET}" --project="${PROJECT_ID}" >/dev/null 2>&1; then
    gcloud secrets versions add "${GEMINI_SECRET}" --data-file="${KUBECONFIG_OUT}.gemini" --project="${PROJECT_ID}" >/dev/null
  else
    gcloud secrets create "${GEMINI_SECRET}" --data-file="${KUBECONFIG_OUT}.gemini" --project="${PROJECT_ID}"
  fi
  rm -f "${KUBECONFIG_OUT}.gemini"
  SECRETS_LIST+=",GEMINI_API_KEY=${GEMINI_SECRET}:latest"
else
  warn "GEMINI_API_KEY not set — AI features will use deterministic fallbacks."
fi

# ── 5. Runtime SA: grant secret accessor ────────────────────────────────────
if [[ -n "${RUNTIME_SA:-}" ]]; then
  SA_EMAIL="${RUNTIME_SA}"
else
  PROJECT_NUMBER="$(gcloud projects describe "${PROJECT_ID}" --format='value(projectNumber)')"
  SA_EMAIL="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"
fi
info "Granting secretAccessor to ${SA_EMAIL}..."
gcloud secrets add-iam-policy-binding "${KUBE_SECRET}" \
  --member="serviceAccount:${SA_EMAIL}" \
  --role="roles/secretmanager.secretAccessor" \
  --project="${PROJECT_ID}" >/dev/null
if [[ -n "${GEMINI_API_KEY:-}" ]]; then
  gcloud secrets add-iam-policy-binding "${SERVICE}-gemini-api-key" \
    --member="serviceAccount:${SA_EMAIL}" \
    --role="roles/secretmanager.secretAccessor" \
    --project="${PROJECT_ID}" >/dev/null
fi

# ── 6. Deploy ───────────────────────────────────────────────────────────────
info "Deploying to Cloud Run (source build via Cloud Build)..."
DEPLOY_ARGS=(
  deploy "${SERVICE}"
  --source=.
  --region="${REGION}"
  --project="${PROJECT_ID}"
  --platform=managed
  --port=8080
  --memory=512Mi
  --cpu=1
  --timeout=300
  --concurrency=8
  --min-instances=0
  --max-instances=3
  --service-account="${SA_EMAIL}"
  --set-env-vars="KUBECONFIG=${KUBE_MOUNT_PATH}"
  --set-secrets="${SECRETS_LIST}"
)
if [[ -n "${NAMESPACE:-}" ]]; then
  DEPLOY_ARGS+=(--update-env-vars="CLUSTERMIND_NAMESPACE=${NAMESPACE}")
fi
if [[ "${ALLOW_UNAUTH}" == "1" ]]; then
  DEPLOY_ARGS+=(--allow-unauthenticated)
fi

gcloud run "${DEPLOY_ARGS[@]}"

URL="$(gcloud run services describe "${SERVICE}" --region="${REGION}" --project="${PROJECT_ID}" --format='value(status.url)')"
info "Deployed ✓  ${URL}"
info "API health: ${URL}/health   UI: ${URL}/"
