import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { fetchSettings, updateSettings } from '../api/client'
import type { AISettings, UpdateSettingsRequest } from '../types'

/**
 * Hook to read and update AI Provider / Model settings.
 */
export function useSettings() {
  const queryClient = useQueryClient()

  const query = useQuery<AISettings>({
    queryKey: ['ai-settings'],
    queryFn: fetchSettings,
    staleTime: 60_000,
  })

  const updateMutation = useMutation({
    mutationFn: (req: UpdateSettingsRequest) => updateSettings(req),
    onSuccess: (data) => {
      queryClient.setQueryData(['ai-settings'], data)
    },
  })

  return {
    settings: query.data,
    isLoading: query.isLoading,
    error: query.error as Error | null,
    updateSettings: updateMutation.mutateAsync,
    isUpdating: updateMutation.isPending,
  }
}
