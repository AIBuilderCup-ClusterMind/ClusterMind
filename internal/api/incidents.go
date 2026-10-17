package api

import (
	"fmt"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"

	"github.com/clustermind/clustermind/internal/graph"
	"github.com/clustermind/clustermind/internal/probe"
)

const (
	maxIncidents = 10
	maxChanges   = 20
	// warningFreshness is how long a Warning event on an otherwise healthy
	// resource keeps an incident open (K8s retains events for ~1h).
	warningFreshness = 10 * time.Minute
)

// newestWarnTime returns the most recent timestamp among warning events.
func newestWarnTime(warns []normalizedEvent) time.Time {
	newest := warns[0].time
	for _, w := range warns[1:] {
		if w.time.After(newest) {
			newest = w.time
		}
	}
	return newest
}

// changeReasons are event reasons that indicate a deployment-related change.
var changeReasons = map[string]bool{
	"ScalingReplicaSet":       true,
	"DeploymentRollingUpdate": true,
	"DeploymentPaused":        true,
	"DeploymentComplete":      true,
	"DeploymentNotAvailable":  true,
}

// normalizedEvent is an event resolved to a graph resource.
type normalizedEvent struct {
	id         string
	time       time.Time
	eventType  string // "Normal" | "Warning"
	reason     string
	message    string
	actor      string
	resourceID string
}

// BuildTwinIncidents derives incidents and recent changes from the snapshot's
// K8s events and the health status of graph resources.
func BuildTwinIncidents(
	snap *probe.ClusterSnapshot,
	g *graph.DependencyGraph,
) ([]TwinIncident, []TwinChange) {
	nodeSet := map[string]bool{}
	for _, n := range g.Nodes {
		nodeSet[n.ID] = true
	}

	statusByID := map[string]probe.ResourceStatus{}
	for _, rs := range probe.ExtractResourceStatuses(snap) {
		statusByID[graph.NodeID(rs.Kind, rs.Namespace, rs.Name)] = rs
	}

	events := normalizeEvents(snap.Events, nodeSet)

	changes := buildChanges(events)
	incidents := buildIncidents(g, statusByID, events)

	return incidents, changes
}

// normalizeEvents resolves raw K8s events to graph resource IDs.
func normalizeEvents(raw []corev1.Event, nodeSet map[string]bool) []normalizedEvent {
	out := make([]normalizedEvent, 0, len(raw))
	for i := range raw {
		ev := &raw[i]
		id, ok := resolveEventResource(ev.InvolvedObject, nodeSet)
		if !ok {
			continue
		}
		t := eventTime(ev)
		out = append(out, normalizedEvent{
			id:         string(ev.UID),
			time:       t,
			eventType:  ev.Type,
			reason:     ev.Reason,
			message:    ev.Message,
			actor:      ev.Source.Component,
			resourceID: id,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].time.Before(out[j].time) })
	return out
}

// resolveEventResource maps an involved object to a graph node ID.
// ReplicaSet events are mapped to their parent Deployment (hash suffix trimmed).
func resolveEventResource(obj corev1.ObjectReference, nodeSet map[string]bool) (string, bool) {
	if obj.Kind == "" || obj.Name == "" {
		return "", false
	}
	exact := graph.NodeID(obj.Kind, obj.Namespace, obj.Name)
	if nodeSet[exact] {
		return exact, true
	}
	if obj.Kind == "ReplicaSet" {
		trimmed := trimPodTemplateHash(obj.Name)
		if trimmed != obj.Name {
			dep := graph.NodeID("Deployment", obj.Namespace, trimmed)
			if nodeSet[dep] {
				return dep, true
			}
		}
	}
	if obj.Kind == "Pod" {
		// Pod → ReplicaSet hash → Deployment (up to two trims)
		name := obj.Name
		for i := 0; i < 2; i++ {
			trimmed := trimPodTemplateHash(name)
			if trimmed == name {
				break
			}
			name = trimmed
			dep := graph.NodeID("Deployment", obj.Namespace, name)
			if nodeSet[dep] {
				return dep, true
			}
		}
	}
	return "", false
}

// trimPodTemplateHash removes a trailing pod-template-hash segment
// (e.g. "web-7d9f84b6c5" → "web").
func trimPodTemplateHash(name string) string {
	idx := strings.LastIndex(name, "-")
	if idx <= 0 {
		return name
	}
	suffix := name[idx+1:]
	if len(suffix) < 5 {
		return name
	}
	for _, r := range suffix {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9') {
			return name
		}
	}
	return name[:idx]
}

// eventTime returns the best-effort timestamp of an event.
func eventTime(ev *corev1.Event) time.Time {
	if !ev.LastTimestamp.IsZero() {
		return ev.LastTimestamp.Time
	}
	if !ev.EventTime.IsZero() {
		return ev.EventTime.Time
	}
	if !ev.FirstTimestamp.IsZero() {
		return ev.FirstTimestamp.Time
	}
	if !ev.CreationTimestamp.IsZero() {
		return ev.CreationTimestamp.Time
	}
	return time.Time{}
}

// buildChanges returns the most recent deployment-related changes.
func buildChanges(events []normalizedEvent) []TwinChange {
	changes := []TwinChange{}
	for _, ev := range events {
		if ev.eventType == "Warning" || !changeReasons[ev.reason] {
			continue
		}
		changes = append(changes, TwinChange{
			ID:          "chg-" + ev.id,
			Time:        ev.time.UTC().Format(time.RFC3339),
			Actor:       ev.actor,
			ResourceID:  ev.resourceID,
			Description: fmt.Sprintf("%s: %s", ev.reason, ev.message),
		})
	}
	// events are already time-ascending; take the most recent ones.
	if len(changes) > maxChanges {
		changes = changes[len(changes)-maxChanges:]
	}
	// Reverse to newest-first for display.
	for i, j := 0, len(changes)-1; i < j; i, j = i+1, j-1 {
		changes[i], changes[j] = changes[j], changes[i]
	}
	return changes
}

// buildIncidents creates one incident per unhealthy/degraded resource that has
// warning evidence (a synthetic alert entry is added when no events exist).
func buildIncidents(
	g *graph.DependencyGraph,
	statusByID map[string]probe.ResourceStatus,
	events []normalizedEvent,
) []TwinIncident {
	warningsByResource := map[string][]normalizedEvent{}
	for _, ev := range events {
		if ev.eventType == "Warning" {
			warningsByResource[ev.resourceID] = append(warningsByResource[ev.resourceID], ev)
		}
	}

	type candidate struct {
		node      graph.ResourceNode
		health    string
		startedAt time.Time
		timeline  []TwinEvidence
	}

	candidates := []candidate{}
	for _, n := range g.Nodes {
		rs, ok := statusByID[n.ID]
		if !ok {
			continue
		}
		health := healthFromPhase(rs.Phase)
		warns := warningsByResource[n.ID]
		// A resource is a candidate when it is unhealthy/degraded, or when it
		// is healthy but still carries fresh warning evidence (early signal).
		// Stale warnings on recovered resources are dropped so incidents clear.
		if health != "unhealthy" && health != "degraded" {
			if len(warns) == 0 || newestWarnTime(warns).Before(time.Now().Add(-warningFreshness)) {
				continue
			}
		}

		incID := incidentIDFor(n)

		var timeline []TwinEvidence
		var startedAt time.Time

		if len(warns) == 0 {
			detail := rs.Message
			if detail == "" {
				detail = rs.Reason
			}
			startedAt = time.Now().UTC()
			timeline = []TwinEvidence{{
				ID:         "ev-" + incID + "-0",
				Time:       startedAt.Format(time.RFC3339),
				Source:     "alert",
				ResourceID: n.ID,
				Title:      fmt.Sprintf("%s %s is %s", n.Kind, n.Name, health),
				Detail:     detail,
			}}
		} else {
			startedAt = warns[0].time.UTC()
			timeline = make([]TwinEvidence, 0, len(warns))
			for i, ev := range warns {
				src := "events"
				if changeReasons[ev.reason] {
					src = "deployment"
				}
				timeline = append(timeline, TwinEvidence{
					ID:         fmt.Sprintf("ev-%s-%d", incID, i),
					Time:       ev.time.UTC().Format(time.RFC3339),
					Source:     src,
					ResourceID: n.ID,
					Title:      ev.reason,
					Detail:     ev.message,
				})
			}
		}

		candidates = append(candidates, candidate{
			node:      n,
			health:    health,
			startedAt: startedAt,
			timeline:  timeline,
		})
	}

	// Newest incidents first; cap.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].startedAt.After(candidates[j].startedAt)
	})
	if len(candidates) > maxIncidents {
		candidates = candidates[:maxIncidents]
	}

	incidents := make([]TwinIncident, 0, len(candidates))
	for _, c := range candidates {
		severity := "sev3"
		switch c.health {
		case "unhealthy":
			severity = "sev1"
		case "degraded":
			severity = "sev2"
		}
		reason := ""
		if rs, ok := statusByID[c.node.ID]; ok && rs.Reason != "" {
			reason = rs.Reason
		}
		title := fmt.Sprintf("%s %s is %s", c.node.Kind, c.node.Name, c.health)
		if reason != "" {
			title = fmt.Sprintf("%s %s — %s", c.node.Kind, c.node.Name, reason)
		}
		if c.health == "healthy" && len(c.timeline) > 0 {
			// Healthy resource with warning evidence — lead with the event reason.
			title = fmt.Sprintf("%s %s — %s", c.node.Kind, c.node.Name, c.timeline[0].Title)
		}
		incidents = append(incidents, TwinIncident{
			ID:                incidentIDFor(c.node),
			Title:             title,
			Severity:          severity,
			Status:            "open",
			StartedAt:         c.startedAt.Format(time.RFC3339),
			PrimaryResourceID: c.node.ID,
			Timeline:          c.timeline,
		})
	}
	return incidents
}

// incidentIDFor builds a stable incident ID for a resource.
func incidentIDFor(n graph.ResourceNode) string {
	slug := strings.ToLower(fmt.Sprintf("inc-%s-%s-%s", n.Kind, n.Namespace, n.Name))
	slug = strings.NewReplacer("/", "-", " ", "-", ".", "-").Replace(slug)
	return slug
}
