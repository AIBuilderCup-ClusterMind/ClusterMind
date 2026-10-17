import { useQuery } from '@tanstack/react-query'
import { fetchTwinSnapshot } from '../api/client'
import type { TwinSnapshot } from '../types'

/**
 * Fetches the digital-twin snapshot (nodes with health/metrics + edges)
 * and polls for changes every 30 seconds.
 */
export function useTwin(): {
  data: TwinSnapshot | undefined
  isLoading: boolean
  error: Error | null
} {
  const { data, isLoading, error } = useQuery({
    queryKey: ['twin-snapshot'],
    queryFn: fetchTwinSnapshot,
    refetchInterval: 30_000,
    retry: 2,
  })
  return { data, isLoading, error: error as Error | null }
}
