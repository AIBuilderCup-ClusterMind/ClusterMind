import { useMutation, type UseMutationResult } from '@tanstack/react-query'
import { investigate } from '../api/client'
import type { InvestigationResult } from '../types'

/**
 * Generates an incident investigation report (Gemini-backed with a
 * deterministic fallback on the server).
 */
export function useInvestigate(
  incidentId: string | null,
): UseMutationResult<InvestigationResult, Error, void> {
  return useMutation({
    mutationFn: () => {
      if (!incidentId) throw new Error('No incident selected')
      return investigate(incidentId)
    },
    retry: 1,
  })
}
