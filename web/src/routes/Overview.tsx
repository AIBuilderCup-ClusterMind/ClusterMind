import { Link } from 'react-router-dom'
import { Panel, PageHeader, StateMessage, StatusBadge, kindIconFor } from '../components/cm/primitives'
import { NamespacePanel } from '../components/NamespacePanel'
import { useTwin } from '../hooks/useTwin'
import type { HealthStatus } from '../types'

export function Overview() {
  const { data: snapshot, isLoading, error } = useTwin()

  if (isLoading) {
    return <StateMessage kind="loading" title="Loading cluster snapshot…" />
  }

  if (error) {
    return (
      <StateMessage
        kind="error"
        title="Couldn't load cluster snapshot"
        detail={error.message}
      />
    )
  }

  if (!snapshot || snapshot.nodes.length === 0) {
    return (
      <StateMessage
        kind="empty"
        title="No resources discovered"
        detail="This cluster has no workloads matching the current filter."
      />
    )
  }

  const count = (st: HealthStatus) => snapshot.nodes.filter((n) => n.status === st).length
  const kinds = Object.entries(
    snapshot.nodes.reduce<Record<string, number>>((acc, n) => {
      acc[n.kind] = (acc[n.kind] ?? 0) + 1
      return acc
    }, {}),
  ).sort(([a], [b]) => a.localeCompare(b))

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        title="Overview"
        subtitle={`Snapshot of ${snapshot.cluster}/${snapshot.namespace} · live cluster data`}
      />
      <div className="grid gap-3 px-5 pb-5 lg:grid-cols-4">
        <Stat label="Resources" value={snapshot.nodes.length} />
        <Stat label="Healthy" value={count('healthy')} tone="text-healthy" />
        <Stat label="Degraded" value={count('degraded')} tone="text-warning" />
        <Stat label="Unhealthy" value={count('unhealthy')} tone="text-critical" />

        <Panel title="Service health" className="lg:col-span-2" bodyClassName="overflow-auto">
          <table className="w-full text-[13px]">
            <thead>
              <tr className="label-xs text-left">
                <th className="px-3 py-2 font-normal">Resource</th>
                <th className="font-normal">Kind</th>
                <th className="font-normal">Key metric</th>
                <th className="px-3 font-normal">Status</th>
              </tr>
            </thead>
            <tbody>
              {snapshot.nodes
                .filter((n) => n.kind !== 'pod')
                .map((n) => (
                  <tr key={n.id} className="border-t border-border/60">
                    <td className="px-3 py-2">
                      <Link
                        to="/topology"
                        className="font-mono text-xs hover:text-primary"
                      >
                        {n.name}
                      </Link>
                    </td>
                    <td className="text-xs text-muted-foreground">{n.kind}</td>
                    <td className="font-mono text-xs">
                      {n.metrics[0] ? `${n.metrics[0].label}: ${n.metrics[0].value}` : '—'}
                    </td>
                    <td className="px-3">
                      <StatusBadge status={n.status} />
                    </td>
                  </tr>
                ))}
            </tbody>
          </table>
        </Panel>

        <div className="flex flex-col gap-3 lg:col-span-2">
          <Panel title="Discovered resources">
            <div className="grid grid-cols-3 gap-px bg-border">
              {kinds.map(([k, v]) => {
                const Icon = kindIconFor(k)
                return (
                  <div key={k} className="flex items-center gap-2 bg-surface p-3">
                    <Icon className="size-4 text-primary" />
                    <span className="text-xs text-muted-foreground">{k}</span>
                    <span className="ml-auto font-mono text-sm">{v}</span>
                  </div>
                )
              })}
            </div>
          </Panel>
          <Panel title="Namespaces" className="min-h-[280px] flex-1" bodyClassName="overflow-hidden">
            <NamespacePanel />
          </Panel>
        </div>
      </div>
    </div>
  )
}

function Stat({ label, value, tone }: { label: string; value: number; tone?: string }) {
  return (
    <div className="rounded-md border bg-surface px-4 py-3">
      <p className="label-xs">{label}</p>
      <p className={`mt-1 font-mono text-2xl ${tone ?? ''}`}>{value}</p>
    </div>
  )
}
