import { create } from 'zustand'

interface IncidentStore {
  /** Currently selected incident ID (shared across Incidents/Investigation views). */
  selectedIncidentId: string | null
  select: (id: string | null) => void
}

export const useIncident = create<IncidentStore>((set) => ({
  selectedIncidentId: null,
  select: (id) => set({ selectedIncidentId: id }),
}))
