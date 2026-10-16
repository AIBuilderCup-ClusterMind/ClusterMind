#!/usr/bin/env bash
set -euo pipefail

# ── ClusterMind one-shot runner ──────────────────────────────────────────────
# Builds backend (Go) + frontend deps (npm), then runs both concurrently.
#
# Usage:
#   ./run.sh                      # default: backend :8080, frontend :5173, all namespaces
#   ./run.sh --namespace demo     # restrict to a single namespace
#   ./run.sh --port 9090 --skip-install
#   ./run.sh --help
# ──────────────────────────────────────────────────────────────────────────────

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${SCRIPT_DIR}"

# ── defaults ────────────────────────────────────────────────────────────────
PORT="${CLUSTERMIND_PORT:-8080}"
NAMESPACE="${CLUSTERMIND_NAMESPACE:-}"   # empty = all namespaces
FRONTEND_PORT=5173
SKIP_INSTALL=0
BACKEND_PID=""
FRONTEND_PID=""

# ── parse args ──────────────────────────────────────────────────────────────
usage() {
  cat <<EOF
ClusterMind — build & run

Usage: ./run.sh [options]

Options:
  --port <n>            Backend API port         (default: 8080)
  --namespace <name>    Restrict to one namespace (default: all)
  --skip-install        Skip npm install (frontend)
  -h, --help            Show this help

Environment:
  GEMINI_API_KEY        Gemini API key (optional, enables AI features)
  KUBECONFIG            kubeconfig path (default: ~/.kube/config)
EOF
  exit 0
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --port)        PORT="$2"; shift 2 ;;
    --namespace)   NAMESPACE="$2"; shift 2 ;;
    --skip-install) SKIP_INSTALL=1; shift ;;
    -h|--help)     usage ;;
    *) echo "Unknown option: $1" >&2; usage ;;
  esac
done

# ── helpers ─────────────────────────────────────────────────────────────────
info()  { printf '\033[1;34m▸\033[0m %s\n' "$*"; }
ok()    { printf '\033[1;32m✔\033[0m %s\n' "$*"; }
fail()  { printf '\033[1;31m✖\033[0m %s\n' "$*" >&2; exit 1; }

cleanup() {
  echo ""
  info "Shutting down..."
  [[ -n "${BACKEND_PID}"  ]] && kill "${BACKEND_PID}"  2>/dev/null || true
  [[ -n "${FRONTEND_PID}" ]] && kill "${FRONTEND_PID}" 2>/dev/null || true
  wait 2>/dev/null || true
  ok "Stopped."
}
trap cleanup EXIT INT TERM

# ── 1. prerequisites ────────────────────────────────────────────────────────
info "Checking prerequisites..."
command -v go   >/dev/null 2>&1 || fail "go not found. Install Go: https://go.dev/dl/"
command -v npm  >/dev/null 2>&1 || fail "npm not found. Install Node.js: https://nodejs.org/"
ok "go $(go version | awk '{print $3}')  |  npm $(npm --version)"

# ── 2. build backend ────────────────────────────────────────────────────────
info "Building backend → bin/clustermind ..."
go build -o bin/clustermind ./cmd/clustermind
ok "Backend binary ready."

# ── 3. frontend deps ────────────────────────────────────────────────────────
if [[ ${SKIP_INSTALL} -eq 0 ]]; then
  info "Installing frontend dependencies..."
  (cd web && npm install --no-fund --no-audit)
  ok "Frontend deps installed."
else
  info "Skipping npm install (--skip-install)"
fi

# ── 4. run backend ──────────────────────────────────────────────────────────
NS_LABEL="${NAMESPACE:-all namespaces}"
info "Starting backend  → http://localhost:${PORT}  (${NS_LABEL})"

# Build the backend command; only pass --namespace when set.
BACKEND_CMD=(./bin/clustermind serve --port "${PORT}")
if [[ -n "${NAMESPACE}" ]]; then
  BACKEND_CMD+=(--namespace "${NAMESPACE}")
fi

CLUSTERMIND_PORT="${PORT}" CLUSTERMIND_NAMESPACE="${NAMESPACE}" \
  "${BACKEND_CMD[@]}" &
BACKEND_PID=$!

# wait briefly, verify process still alive
sleep 1
kill -0 "${BACKEND_PID}" 2>/dev/null || fail "Backend failed to start. Check logs above."

# ── 5. run frontend ─────────────────────────────────────────────────────────
info "Starting frontend → http://localhost:${FRONTEND_PORT}"
(cd web && npm run dev -- --port "${FRONTEND_PORT}" --strictPort) &
FRONTEND_PID=$!

sleep 2
kill -0 "${FRONTEND_PID}" 2>/dev/null || fail "Frontend failed to start. Check logs above."

# ── 6. done ─────────────────────────────────────────────────────────────────
echo ""
ok "ClusterMind is running!"
echo "   Web UI : http://localhost:${FRONTEND_PORT}"
echo "   API    : http://localhost:${PORT}"
echo "   Scope  : ${NS_LABEL}"
echo ""
echo "   Ctrl+C to stop both servers."
echo ""

# wait for either process to exit
wait -n "${BACKEND_PID}" "${FRONTEND_PID}" 2>/dev/null || wait
