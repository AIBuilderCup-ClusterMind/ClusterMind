import { create } from 'zustand'
import type { FlowNodeData } from '../types'

interface SelectionStore {
  selectedId: string | null
  selectedData: FlowNodeData | null
  select: (id: string, data: FlowNodeData) => void
  clear: () => void
}

export const useSelection = create<SelectionStore>((set) => ({
  selectedId: null,
  selectedData: null,

  select: (id, data) => set({ selectedId: id, selectedData: data }),
  clear: () => set({ selectedId: null, selectedData: null }),
}))
