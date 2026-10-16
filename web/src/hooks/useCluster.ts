import { useQuery } from '@tanstack/react-query'
import { fetchGraph } from '../api/client'
import type { FlowEdge, FlowNode } from '../types'

/**
 * Fetches the cluster dependency graph and polls for changes every 30 seconds.
 */
export function useCluster(): {
  nodes: FlowNode[]
  edges: FlowEdge[]
  isLoading: boolean
  error: Error | null
} {
  const { data, isLoading, error } = useQuery({
    queryKey: ['cluster-graph'],
    queryFn: fetchGraph,
    refetchInterval: 30_000,
    retry: 2,
  })

  return {
    nodes: data?.nodes ?? [],
    edges: data?.edges ?? [],
    isLoading,
    error: error as Error | null,
  }
}
