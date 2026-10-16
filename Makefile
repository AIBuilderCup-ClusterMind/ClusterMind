.PHONY: build run-backend run-frontend demo test clean tidy help

help:
	@echo "ClusterMind - Kubernetes Cluster Analysis"
	@echo ""
	@echo "Available targets:"
	@echo "  build           Build ClusterMind backend binary"
	@echo "  run-backend     Build and run backend API server"
	@echo "  run-frontend    Start React frontend dev server"
	@echo "  demo            Setup kind cluster with demo resources"
	@echo "  test            Run Go tests"
	@echo "  clean           Remove build artifacts"
	@echo "  tidy            Run go mod tidy"
	@echo ""

build:
	go build -o bin/clustermind ./cmd/clustermind

run-backend: build
	CLUSTERMIND_NAMESPACE=demo ./bin/clustermind serve --port 8080

run-frontend:
	cd web && npm run dev

demo:
	bash demo/run_demo.sh

test:
	go test -v ./...

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

clean:
	rm -rf bin/
	rm -rf web/dist/
	rm -f coverage.out coverage.html

tidy:
	go mod tidy
