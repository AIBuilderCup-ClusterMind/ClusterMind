// Package demo embeds the mock cluster manifest used by demo mode.
package demo

import _ "embed"

// MockClusterYAML is the demo cluster manifest (demo/mock_cluster.yaml).
// It is loaded by probe.NewDemoCollector when CLUSTERMIND_DEMO is set,
// letting ClusterMind run without any reachable Kubernetes cluster.
//
//go:embed mock_cluster.yaml
var MockClusterYAML []byte
