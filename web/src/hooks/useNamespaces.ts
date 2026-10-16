import { useQuery } from '@tanstack/react-query'
import { fetchNamespaceResources, fetchNamespaces } from '../api/client'
import type { NamespaceListResponse, NamespaceResourcesResponse } from '../types'

/**
 * Fetches every namespace with aggregated resource counts and status rollups.
 */
export function useNamespaces(): {
  data: NamespaceListResponse | undefined
  isLoading: boolean
  error: Error | null
} {
  const { data, isLoading, error } = useQuery({
    queryKey: ['namespaces'],
    queryFn: fetchNamespaces,
    refetchInterval: 30_000,
    retry: 2,
  })
  return { data, isLoading, error: error as Error | null }
}

/**
 * Fetches every resource in the given namespace with normalized status.
 * When `kind` is provided, the backend filters to that Kind only.
 */
export function useNamespaceResources(
  namespace: string | null,
  kind?: string,
): {
  data: NamespaceResourcesResponse | undefined
  isLoading: boolean
  error: Error | null
} {
  const { data, isLoading, error } = useQuery({
    queryKey: ['namespace-resources', namespace, kind ?? ''],
    queryFn: () => fetchNamespaceResources(namespace!, kind),
    enabled: !!namespace,
    refetchInterval: 30_000,
    retry: 2,
  })
  return { data, isLoading, error: error as Error | null }
}
