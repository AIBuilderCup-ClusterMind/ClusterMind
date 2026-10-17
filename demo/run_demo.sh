#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
KIND_CLUSTER="clustermind-demo"

FAIL=0
HEAL=0
CONTEXT=""

usage() {
  cat <<EOF
Usage: run_demo.sh [options]

Options:
  --context NAME   kubectl context to use (default: kind ${KIND_CLUSTER} if it
                   exists, else create it with kind, else current context)
  --fail           Break payment-deployment (bad image) to generate an incident
  --heal           Restore payment-deployment to the healthy image
  -h, --help       Show this help

Steps (no flags): create/select cluster → apply demo/mock_cluster.yaml
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --context) CONTEXT="${2:?--context needs a value}"; shift 2 ;;
    --fail) FAIL=1; shift ;;
    --heal) HEAL=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 1 ;;
  esac
done

PAYMENT_IMAGE="registry.k8s.io/e2e-test-images/agnhost:2.53"

# ── Select cluster context ──────────────────────────────────────────────────
if [[ -z "${CONTEXT}" ]]; then
  if kubectl config get-contexts "kind-${KIND_CLUSTER}" -o name >/dev/null 2>&1; then
    CONTEXT="kind-${KIND_CLUSTER}"
  elif command -v kind >/dev/null 2>&1; then
    echo "1. Creating kind cluster '${KIND_CLUSTER}'..."
    kind create cluster --name "${KIND_CLUSTER}" >/dev/null
    CONTEXT="kind-${KIND_CLUSTER}"
  else
    CONTEXT="$(kubectl config current-context)"
    echo "1. kind not found — using current context '${CONTEXT}'"
  fi
else
  echo "1. Using explicit context '${CONTEXT}'"
fi
export KUBECONFIG="${KUBECONFIG:-${HOME}/.kube/config}"

echo ""
if [[ ${HEAL} -eq 1 ]]; then
  echo "2. Restoring payment-deployment..."
  kubectl --context "${CONTEXT}" -n demo set image deployment/payment-deployment payment="${PAYMENT_IMAGE}"
  exit 0
fi

if [[ ${FAIL} -eq 1 ]]; then
  echo "2. Breaking payment-deployment (simulated failure)..."
  kubectl --context "${CONTEXT}" -n demo set image deployment/payment-deployment payment="nginx:does-not-exist"
  echo "   → ImagePullBackOff will surface as a sev1 incident within ~30s."
  echo "   → Undo with: $0 --heal"
  exit 0
fi

echo "2. Applying demo namespace and K8s resources..."
kubectl --context "${CONTEXT}" apply -f "${SCRIPT_DIR}/mock_cluster.yaml" >/dev/null

echo ""
echo "3. Waiting for workloads to become Ready..."
kubectl --context "${CONTEXT}" -n demo rollout status deployment/frontend-deployment --timeout=120s >/dev/null 2>&1 || true
kubectl --context "${CONTEXT}" -n demo rollout status deployment/api-deployment --timeout=120s >/dev/null 2>&1 || true

echo ""
echo "4. Building ClusterMind backend..."
(cd "${REPO_ROOT}" && go build -o bin/clustermind ./cmd/clustermind)

echo ""
echo "✅ Demo setup complete (context: ${CONTEXT})"
cat <<EOF

Next steps:

  Terminal 1 — backend:
    ./bin/clustermind serve --port 8080 --namespace demo

  Terminal 2 — frontend:
    cd web && npm run dev     # → http://localhost:5173

Demo commands:

  # Twin snapshot (nodes + edges + changes + incidents)
  curl -s http://localhost:8080/api/v1/twin/snapshot | head -c 400

  # payment-svc failure impact
  curl -s -X POST http://localhost:8080/api/v1/impact \\
    -H 'Content-Type: application/json' \\
    -d '{"target": "Service/demo/payment-svc"}'

  # Simulate a payment failure (generates an incident)
  $0 --fail        # undo: $0 --heal

  # Investigation report (use the incident ID from /api/v1/twin/snapshot)
  curl -s -X POST http://localhost:8080/api/v1/investigate \\
    -H 'Content-Type: application/json' \\
    -d '{"incidentId": "inc-deployment-demo-payment-deployment"}'
EOF
