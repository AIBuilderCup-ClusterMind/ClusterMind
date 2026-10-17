import { useEffect, useState } from 'react'
import { AlertTriangle, Info } from 'lucide-react'
import { cn } from '../lib/utils'
import { PageHeader, Panel } from '../components/cm/primitives'
import { TopologyGraph } from '../components/TopologyGraph'
import { NamespacePanel } from '../components/NamespacePanel'
import { NodeDetailPanel } from '../components/NodeDetailPanel'
import { ImpactPanel } from '../components/ImpactPanel'
import { useImpact } from '../hooks/useImpact'
import { useSelection } from '../hooks/useSelection'

type Tab = 'details' | 'impact'

const tabConfig: { key: Tab; label: string; icon: React.ReactNode }[] = [
  { key: 'details', label: 'Details', icon: <Info className="size-3.5" /> },
  { key: 'impact', label: 'Impact', icon: <AlertTriangle className="size-3.5" /> },
]

export function Topology() {
  const { selectedId } = useSelection()
  const { target, result: impactResult } = useImpact()
  const [tab, setTab] = useState<Tab>('details')

  useEffect(() => {
    if (selectedId) setTab('details')
  }, [selectedId])

  useEffect(() => {
    if (target) setTab('impact')
  }, [target])

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        title="Topology"
        subtitle="Directed edges show who calls whom. Click a node to inspect it."
      />
      <div className="grid min-h-[520px] flex-1 gap-3 px-5 pb-5 lg:grid-cols-[260px_1fr_320px]">
        <Panel title="Namespaces" bodyClassName="overflow-hidden">
          <NamespacePanel />
        </Panel>

        <Panel className="relative overflow-hidden">
          <div className="absolute inset-0">
            <TopologyGraph />
          </div>
          <Legend showImpact={impactResult != null} />
        </Panel>

        <Panel
          className="min-h-[420px]"
          title={tab === 'details' ? 'Resource details' : 'Impact analysis'}
          actions={
            <div className="flex gap-1">
              {tabConfig.map(({ key, label, icon }) => (
                <button
                  key={key}
                  onClick={() => setTab(key)}
                  title={label}
                  className={cn(
                    'rounded-sm px-2 py-1 text-[11px] transition-colors',
                    tab === key
                      ? 'bg-accent text-foreground'
                      : 'text-muted-foreground hover:bg-accent hover:text-foreground',
                  )}
                >
                  <span className="flex items-center gap-1">
                    {icon}
                    {label}
                  </span>
                </button>
              ))}
            </div>
          }
          bodyClassName="overflow-hidden"
        >
          <div className={cn('h-full min-h-0', tab === 'details' ? 'block' : 'hidden')}>
            <NodeDetailPanel />
          </div>
          <div className={cn('h-full min-h-0', tab === 'impact' ? 'block' : 'hidden')}>
            <ImpactPanel />
          </div>
        </Panel>
      </div>
    </div>
  )
}

function Legend({ showImpact }: { showImpact: boolean }) {
  return (
    <div className="pointer-events-none absolute right-3 top-3 flex gap-3 rounded-md border bg-background/90 px-2.5 py-1.5 font-mono text-[10px] text-muted-foreground">
      <span className="flex items-center gap-1">
        <i className="size-1.5 rounded-full bg-healthy" />
        healthy
      </span>
      <span className="flex items-center gap-1">
        <i className="size-1.5 rounded-full bg-warning" />
        degraded
      </span>
      <span className="flex items-center gap-1">
        <i className="size-1.5 rounded-full bg-critical" />
        unhealthy
      </span>
      {showImpact && (
        <span className="flex items-center gap-1 border-l pl-3">
          <i className="size-2 rounded-sm border border-dashed border-warning" />
          potentially affected
        </span>
      )}
    </div>
  )
}
