import { create } from 'zustand'

interface SelectionStore {
  selectedId: string | null
  select: (id: string) => void
  clear: () => void
}

export const useSelection = create<SelectionStore>((set) => ({
  selectedId: null,

  select: (id) => set({ selectedId: id }),
  clear: () => set({ selectedId: null }),
}))
