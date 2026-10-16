import type {
  GraphResult,
  ImpactResponse,
  NamespaceListResponse,
  NamespaceResourcesResponse,
  SnapshotSummary,
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
