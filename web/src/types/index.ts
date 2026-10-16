// Shared TypeScript types — mirrors Go DTO structs 1:1

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
