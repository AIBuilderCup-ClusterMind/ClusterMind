import { create } from 'zustand'
import { analyzeImpact } from '../api/client'
import type { ImpactResponse } from '../types'

interface ImpactStore {
  target: string | null
  result: ImpactResponse | null
  isLoading: boolean
  error: string | null
  setTarget: (t: string) => void
  analyze: () => Promise<void>
  clear: () => void
}

export const useImpact = create<ImpactStore>((set, get) => ({
  target: null,
  result: null,
  isLoading: false,
  error: null,

  setTarget: (t) => set({ target: t, result: null, error: null }),

  analyze: async () => {
    const { target } = get()
    if (!target) return
    set({ isLoading: true, error: null })
    try {
      const result = await analyzeImpact(target)
      set({ result, isLoading: false })
    } catch (err) {
      set({
        error: err instanceof Error ? err.message : 'Analysis failed',
        isLoading: false,
      })
    }
  },

  clear: () => set({ target: null, result: null, error: null }),
}))
