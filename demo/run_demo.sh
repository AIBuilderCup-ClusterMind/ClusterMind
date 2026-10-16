#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "=== ClusterMind Demo Setup ==="
echo ""
echo "1. Creating kind cluster 'clustermind-demo'..."
kind create cluster --name clustermind-demo 2>/dev/null || echo "   ℹ️  Cluster already exists, skipping creation"

echo ""
echo "2. Applying demo namespace and K8s resources..."
kubectl apply -f "${SCRIPT_DIR}/mock_cluster.yaml"

echo ""
echo "3. Waiting for resources to be created (5 seconds)..."
sleep 5

echo ""
echo "4. Verifying demo namespace..."
kubectl get namespace demo -o jsonpath='{.metadata.name}' || echo "Error: demo namespace not found"
echo ""

echo "✅ Demo cluster setup complete!"
echo ""
echo "Next steps:"
echo ""
echo "  Terminal 1 - Start ClusterMind backend:"
echo "    bin/clustermind serve --port 8080 --namespace demo"
echo ""
echo "  Terminal 2 - Start React frontend:"
echo "    cd web && npm run dev"
echo ""
echo "  Web UI: http://localhost:5173"
echo "  API: http://localhost:8080"
echo ""
echo "Demo commands:"
echo ""
echo "  View cluster snapshot:"
echo "    curl http://localhost:8080/api/v1/cluster/snapshot"
echo ""
echo "  Analyze payment-svc failure impact:"
echo "    curl -X POST http://localhost:8080/api/v1/impact \\"
echo "      -H 'Content-Type: application/json' \\"
echo "      -d '{\"target\": \"Service/demo/payment-svc\"}'"
echo ""
echo "  Get dependency graph:"
echo "    curl http://localhost:8080/api/v1/cluster/graph"
echo ""
