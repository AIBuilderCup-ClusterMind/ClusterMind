import { useEffect, useState } from 'react'
import { AlertTriangle, Info } from 'lucide-react'
import { clsx } from 'clsx'
import { ImpactPanel } from './ImpactPanel'
import { NodeDetailPanel } from './NodeDetailPanel'
import { useSelection } from '../hooks/useSelection'

type Tab = 'details' | 'impact'

const tabConfig: { key: Tab; label: string; icon: React.ReactNode }[] = [
  { key: 'details', label: 'Details', icon: <Info className="w-3.5 h-3.5" /> },
  { key: 'impact', label: 'Impact', icon: <AlertTriangle className="w-3.5 h-3.5" /> },
]

export function RightPanel() {
  const [tab, setTab] = useState<Tab>('details')
  const { selectedId } = useSelection()

  // Auto-switch to Details when a node is selected.
  useEffect(() => {
    if (selectedId) {
      setTab('details')
    }
  }, [selectedId])

  return (
    <div className="w-80 flex flex-col border-l border-slate-200 bg-white overflow-hidden">
      {/* Tab bar */}
      <div className="flex border-b border-slate-200 bg-slate-50 flex-shrink-0">
        {tabConfig.map(({ key, label, icon }) => (
          <button
            key={key}
            onClick={() => setTab(key)}
            className={clsx(
              'flex-1 flex items-center justify-center gap-1.5 px-3 py-2.5 text-xs font-medium transition-colors border-b-2 -mb-px',
              tab === key
                ? 'border-blue-500 text-blue-600 bg-white'
                : 'border-transparent text-slate-500 hover:text-slate-700 hover:bg-slate-100',
            )}
          >
            {icon}
            {label}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {tab === 'details' ? <NodeDetailPanel /> : <ImpactPanel />}
      </div>
    </div>
  )
}
