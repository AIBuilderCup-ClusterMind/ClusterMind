// Package api defines REST API request/response DTOs.
package api

import (
	"time"

	aiPkg "github.com/clustermind/clustermind/internal/ai"
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
	GeneratedBy       string              `json:"generatedBy,omitempty"`
	AffectedResources []string            `json:"affectedResources"`
	PropagationPaths  [][]string          `json:"propagationPaths"`
	Severity          string              `json:"severity"`
	Explanation       string              `json:"explanation"`
	Graph             *render.GraphResult `json:"graph"`
}


// SnapshotSummary is a lightweight summary returned by GET /api/v1/cluster/snapshot.
type SnapshotSummary struct {
	Namespaces   int `json:"namespaces"`
	Deployments  int `json:"deployments"`
	StatefulSets int `json:"statefulSets"`
	DaemonSets   int `json:"daemonSets"`
	Pods         int `json:"pods"`
	Services     int `json:"services"`
	Ingresses    int `json:"ingresses"`
	HPAs         int `json:"hpas"`
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
	Namespace string                 `json:"namespace"`
	Resources []probe.ResourceStatus `json:"resources"`
}

// ── Twin snapshot (digital-twin view contract) ───────────────────────────────

// TwinMetric is a single label/value metric shown on a topology node card.
type TwinMetric struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// TwinNode is a digital-twin view of one graph vertex: identity, normalized
// health status, display metrics and labels, and a computed grid position.
type TwinNode struct {
	ID        string            `json:"id"` // "{Kind}/{namespace}/{name}"
	Name      string            `json:"name"`
	Kind      string            `json:"kind"` // lowercased K8s kind, e.g. "deployment"
	Namespace string            `json:"namespace"`
	Status    string            `json:"status"` // "healthy"|"degraded"|"unhealthy"|"unknown"
	Reason    string            `json:"reason,omitempty"`
	Metrics   []TwinMetric      `json:"metrics"`
	Labels    map[string]string `json:"labels"`
	Position  render.Position   `json:"position"`
	Replicas  *int32            `json:"replicas,omitempty"`
}

// TwinEdge is a digital-twin view of one dependency edge.
type TwinEdge struct {
	ID       string `json:"id"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	Protocol string `json:"protocol"` // dependency type, e.g. "selector_match"
}

// TwinSnapshot is the full digital-twin payload consumed by the web UI.
type TwinSnapshot struct {
	Cluster    string         `json:"cluster"`
	Namespace  string         `json:"namespace"` // filter, or "all"
	CapturedAt time.Time      `json:"capturedAt"`
	Nodes      []TwinNode     `json:"nodes"`
	Edges      []TwinEdge     `json:"edges"`
	Changes    []TwinChange   `json:"changes"`
	Incidents  []TwinIncident `json:"incidents"`
}

// TwinEvidence is a single timeline entry for an incident.
type TwinEvidence struct {
	ID         string `json:"id"`
	Time       string `json:"time"`   // RFC3339
	Source     string `json:"source"` // "events" | "deployment" | "alert"
	ResourceID string `json:"resourceId"`
	Title      string `json:"title"`
	Detail     string `json:"detail"`
}

// TwinChange is a recent infrastructure change (e.g. deployment rollout).
type TwinChange struct {
	ID          string `json:"id"`
	Time        string `json:"time"` // RFC3339
	Actor       string `json:"actor"`
	ResourceID  string `json:"resourceId"`
	Description string `json:"description"`
}

// TwinIncident is a derived incident grouping degradation evidence.
type TwinIncident struct {
	ID                string         `json:"id"`
	Title             string         `json:"title"`
	Severity          string         `json:"severity"` // "sev1" | "sev2" | "sev3"
	Status            string         `json:"status"`   // "open"
	StartedAt         string         `json:"startedAt"`
	PrimaryResourceID string         `json:"primaryResourceId"`
	Timeline          []TwinEvidence `json:"timeline"`
}

// InvestigateRequest is the POST /api/v1/investigate request body.
type InvestigateRequest struct {
	IncidentID string `json:"incidentId" binding:"required"`
}

// ── AI Settings ─────────────────────────────────────────────────────────────

// ProviderInfo describes one supported AI provider and model choices.
type ProviderInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	DefaultModel string   `json:"defaultModel"`
	Models       []string `json:"models"`
	KeyHelpURL   string   `json:"keyHelpUrl"`
	KeyPrefix    string   `json:"keyPrefix"`
}

// SettingsResponse is returned by GET /api/v1/settings.
type SettingsResponse struct {
	Provider           string         `json:"provider"`
	Model              string         `json:"model"`
	HasAPIKey          bool           `json:"hasApiKey"`
	MaskedAPIKey       string         `json:"maskedApiKey"`
	SupportedProviders []ProviderInfo `json:"supportedProviders"`
}

// UpdateSettingsRequest is the POST /api/v1/settings request body.
type UpdateSettingsRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIKey   string `json:"apiKey"`
}

// TestSettingsRequest is the POST /api/v1/settings/test request body.
type TestSettingsRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIKey   string `json:"apiKey"`
}

// TestSettingsResponse is the response for POST /api/v1/settings/test.
type TestSettingsResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ListModelsRequest is the POST /api/v1/settings/models request body.
type ListModelsRequest struct {
	Provider string `json:"provider"`
	APIKey   string `json:"apiKey"`
}

// ListModelsResponse is the response for POST /api/v1/settings/models.
type ListModelsResponse struct {
	Provider string             `json:"provider"`
	Models   []aiPkg.ModelItem `json:"models"`
}


