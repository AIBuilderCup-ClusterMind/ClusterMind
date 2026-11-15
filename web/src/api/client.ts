import type {
  AISettings,
  GraphResult,
  ImpactResponse,
  InvestigationResult,
  ListModelsRequest,
  ListModelsResponse,
  NamespaceListResponse,
  NamespaceResourcesResponse,
  SnapshotSummary,
  TestSettingsRequest,
  TestSettingsResponse,
  TwinSnapshot,
  UpdateSettingsRequest,
} from '../types'

const BASE = (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? ''

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })
  const data = (await res.json()) as T & { error?: string }
  if (!res.ok) {
    throw new Error((data as { error?: string }).error ?? `HTTP ${res.status}`)
  }
  return data
}

/** Fetch the digital-twin snapshot (identity + health + metrics per node). */
export function fetchTwinSnapshot(): Promise<TwinSnapshot> {
  return request<TwinSnapshot>('/api/v1/twin/snapshot')
}

/** Fetch React Flow-compatible cluster graph (nodes + edges). */
export function fetchGraph(): Promise<GraphResult> {
  return request<GraphResult>('/api/v1/cluster/graph')
}

/** Fetch lightweight cluster resource count summary. */
export function fetchSnapshot(): Promise<SnapshotSummary> {
  return request<SnapshotSummary>('/api/v1/cluster/snapshot')
}

/** Run failure impact analysis for the given target resource ID. */
export function analyzeImpact(target: string): Promise<ImpactResponse> {
  return request<ImpactResponse>('/api/v1/impact', {
    method: 'POST',
    body: JSON.stringify({ target }),
  })
}

/** Generate a structured investigation report for a derived incident. */
export function investigate(incidentId: string): Promise<InvestigationResult> {
  return request<InvestigationResult>('/api/v1/investigate', {
    method: 'POST',
    body: JSON.stringify({ incidentId }),
  })
}

/** Fetch all namespaces with aggregated resource counts and status rollups. */
export function fetchNamespaces(): Promise<NamespaceListResponse> {
  return request<NamespaceListResponse>('/api/v1/namespaces')
}

/** Fetch every resource in a namespace with normalized health status. */
export function fetchNamespaceResources(
  namespace: string,
  kind?: string,
): Promise<NamespaceResourcesResponse> {
  const qs = kind ? `?kind=${encodeURIComponent(kind)}` : ''
  return request<NamespaceResourcesResponse>(
    `/api/v1/namespaces/${encodeURIComponent(namespace)}/resources${qs}`,
  )
}

/** Fetch current AI provider/model settings and metadata. */
export function fetchSettings(): Promise<AISettings> {
  return request<AISettings>('/api/v1/settings')
}

/** Update AI provider, model, and/or API key. */
export function updateSettings(req: UpdateSettingsRequest): Promise<AISettings> {
  return request<AISettings>('/api/v1/settings', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

/** Test AI provider connection and model availability. */
export function testSettings(req: TestSettingsRequest): Promise<TestSettingsResponse> {
  return request<TestSettingsResponse>('/api/v1/settings/test', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

/** Fetch live available models for the given provider and API key. */
export function fetchAvailableModels(req: ListModelsRequest): Promise<ListModelsResponse> {
  return request<ListModelsResponse>('/api/v1/settings/models', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}


