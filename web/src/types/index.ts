// Shared TypeScript types — mirrors Go DTO structs 1:1

// ── Digital twin snapshot (GET /api/v1/twin/snapshot) ───────────────────────

export type HealthStatus = 'healthy' | 'degraded' | 'unhealthy' | 'unknown'

export interface TwinMetric {
  label: string
  value: string
}

export interface TwinNode {
  id: string
  name: string
  kind: string // lowercased K8s kind, e.g. "deployment"
  namespace: string
  status: HealthStatus
  reason?: string
  metrics: TwinMetric[]
  labels: Record<string, string>
  position: { x: number; y: number }
  replicas?: number
}

export interface TwinEdge {
  id: string
  source: string
  target: string
  protocol: string
}

export type EvidenceSource = 'events' | 'deployment' | 'alert'

export interface TwinEvidence {
  id: string
  time: string // RFC3339
  source: EvidenceSource
  resourceId: string
  title: string
  detail: string
}

export interface TwinChange {
  id: string
  time: string // RFC3339
  actor: string
  resourceId: string
  description: string
}

export type IncidentSeverity = 'sev1' | 'sev2' | 'sev3'

export interface TwinIncident {
  id: string
  title: string
  severity: IncidentSeverity
  status: 'open' | 'resolved'
  startedAt: string
  primaryResourceId: string
  timeline: TwinEvidence[]
}

export interface TwinSnapshot {
  cluster: string
  namespace: string
  capturedAt: string
  nodes: TwinNode[]
  edges: TwinEdge[]
  changes: TwinChange[]
  incidents: TwinIncident[]
}

// ── Investigation (POST /api/v1/investigate) ────────────────────────────────

export type InvestigationConfidence = 'high' | 'medium' | 'low'

export interface PossibleCause {
  cause: string
  confidence: InvestigationConfidence
}

export interface SupportingEvidenceRef {
  evidenceId: string
  note: string
}

export interface InvestigationResult {
  incidentId: string
  generatedBy: 'gemini' | 'fallback'
  summary: string
  observedFacts: string[]
  possibleCauses: PossibleCause[]
  supportingEvidence: SupportingEvidenceRef[]
  nextChecks: string[]
}

// ── Legacy graph result (GET /api/v1/cluster/graph) ─────────────────────────

export interface ResourceNode {
  id: string
  kind: string
  name: string
  namespace: string
  labels: Record<string, string>
  meta: Record<string, unknown>
}

export interface DependencyEdge {
  source: string
  target: string
  edgeType: string
  confidence: number
  direction: string
}

export type ImpactLevel = 'critical' | 'direct' | 'indirect' | 'none'

export interface FlowNodeData extends Record<string, unknown> {
  label: string
  kind: string
  namespace: string
  impactLevel: ImpactLevel
  replicas?: number
}

export interface FlowNode {
  id: string
  type: string
  position: { x: number; y: number }
  data: FlowNodeData
}

export interface FlowEdge {
  id: string
  source: string
  target: string
  type: string
  animated: boolean
  label?: string
}

export interface GraphResult {
  nodes: FlowNode[]
  edges: FlowEdge[]
}

export interface ImpactRequest {
  target: string
}

export interface ImpactResponse {
  target: string
  affectedResources: string[]
  propagationPaths: string[][]
  severity: 'critical' | 'high' | 'medium' | 'low'
  explanation: string
  graph: GraphResult
}

export interface SnapshotSummary {
  namespaces: number
  deployments: number
  statefulSets: number
  daemonSets: number
  pods: number
  services: number
  ingresses: number
  hpas: number
}

// ── Namespace & resource status types ────────────────────────────────────────

export type StatusPhase =
  | 'Healthy'
  | 'Degraded'
  | 'Warning'
  | 'Pending'
  | 'Completed'
  | 'Unknown'

export interface ResourceStatus {
  kind: string
  name: string
  namespace: string
  phase: StatusPhase
  ready: string
  reason: string
  message: string
  age: string
}

export interface NamespaceResourceStats {
  name: string
  phase: StatusPhase
  resourceCount: number
  counts: Record<string, number>
  phaseCounts: Record<StatusPhase, number>
  age: string
}

export interface NamespaceListResponse {
  namespaces: NamespaceResourceStats[]
}

export interface NamespaceResourcesResponse {
  namespace: string
  resources: ResourceStatus[]
}
