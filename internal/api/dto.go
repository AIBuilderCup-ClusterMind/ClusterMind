// Package api defines REST API request/response DTOs.
package api

import (
	"github.com/clustermind/clustermind/internal/probe"
	"github.com/clustermind/clustermind/internal/render"
)

// ImpactRequest is the POST /api/v1/impact request body.
type ImpactRequest struct {
	Target string `json:"target" binding:"required"` // e.g. "Service/demo/payment-svc"
}

// ImpactResponse is the POST /api/v1/impact response body.
type ImpactResponse struct {
	Target            string              `json:"target"`
	AffectedResources []string            `json:"affectedResources"`
	PropagationPaths  [][]string          `json:"propagationPaths"`
	Severity          string              `json:"severity"`
	Explanation       string              `json:"explanation"`
	Graph             *render.GraphResult `json:"graph"`
}

// SnapshotSummary is a lightweight summary returned by GET /api/v1/cluster/snapshot.
type SnapshotSummary struct {
	Namespaces  int `json:"namespaces"`
	Deployments int `json:"deployments"`
	StatefulSets int `json:"statefulSets"`
	DaemonSets  int `json:"daemonSets"`
	Pods        int `json:"pods"`
	Services    int `json:"services"`
	Ingresses   int `json:"ingresses"`
	HPAs        int `json:"hpas"`
}

// ErrorResponse is a generic error response body.
type ErrorResponse struct {
	Error string `json:"error"`
}

// NamespaceListResponse is the GET /api/v1/namespaces response body.
type NamespaceListResponse struct {
	Namespaces []probe.NamespaceResourceStats `json:"namespaces"`
}

// NamespaceResourcesResponse is the GET /api/v1/namespaces/:namespace/resources response body.
type NamespaceResourcesResponse struct {
	Namespace string                   `json:"namespace"`
	Resources []probe.ResourceStatus   `json:"resources"`
}
