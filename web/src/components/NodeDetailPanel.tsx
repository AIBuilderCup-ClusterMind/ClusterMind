import { useMemo } from 'react'
import { ArrowDown, ArrowUp, Clock, Play, X } from 'lucide-react'
import { cn } from '../lib/utils'
import { Button, StateMessage, StatusBadge, kindIconFor } from './cm/primitives'
import { useImpact } from '../hooks/useImpact'
import { useSelection } from '../hooks/useSelection'
import { useTwin } from '../hooks/useTwin'
import type { TwinEdge, TwinNode } from '../types'

// ── impact level → badge ───────────────────────────────────────────────────

const impactBadge: Record<string, string> = {
  critical: 'bg-critical/10 text-critical border-critical/30',
  direct: 'bg-warning/10 text-warning border-warning/30',
  indirect: 'bg-warning/10 text-warning border-warning/30',
  none: 'bg-muted text-muted-foreground border-border-strong',
}

// ── section helpers ────────────────────────────────────────────────────────

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1.5">
      <p className="label-xs">{title}</p>
      {children}
    </div>
  )
}

function InfoRow({ label, value }: { label: string; value: React.ReactNode }) {
  if (value === null || value === undefined || value === '') return null
  return (
    <div className="flex items-start gap-2 text-xs">
      <span className="w-20 shrink-0 text-muted-foreground">{label}</span>
      <span className="break-all font-mono text-foreground">{value}</span>
    </div>
  )
}

// ── connection list ────────────────────────────────────────────────────────

function ConnectionList({
  edges,
  direction,
  onSelect,
}: {
  edges: TwinEdge[]
  direction: 'upstream' | 'downstream'
  onSelect: (id: string) => void
}) {
  const Icon = direction === 'upstream' ? ArrowUp : ArrowDown
  const label = direction === 'upstream' ? 'Upstream' : 'Downstream'

  if (edges.length === 0) {
    return <p className="text-xs italic text-muted-foreground">No {label} connections.</p>
  }

  return (
    <div className="space-y-1">
      <div className="flex items-center gap-1 text-[11px] text-muted-foreground">
        <Icon className="size-3" />
        {label} ({edges.length})
      </div>
      {edges.map((e) => {
        const peerId = direction === 'upstream' ? e.source : e.target
        const parts = peerId.split('/')
        const kind = parts[0]
        const name = parts.slice(2).join('/') || parts[1]
        const PeerIcon = kindIconFor(kind)
        return (
          <button
            key={e.id}
            onClick={() => onSelect(peerId)}
            title={peerId}
            className="flex w-full items-center gap-1.5 rounded-md border border-border/60 bg-background px-2.5 py-1.5 text-left text-xs text-foreground transition-colors hover:border-primary/60"
          >
            <PeerIcon className="size-3.5 shrink-0 text-primary" />
            <span className="min-w-0 flex-1 truncate font-mono">{name}</span>
            <span className="shrink-0 font-mono text-[9px] text-muted-foreground">{kind}</span>
          </button>
        )
      })}
    </div>
  )
}

// ── main panel ─────────────────────────────────────────────────────────────

export function NodeDetailPanel() {
  const { selectedId, clear, select } = useSelection()
  const { data: snapshot } = useTwin()
  const { result: impactResult, setTarget, analyze } = useImpact()

  const node: TwinNode | undefined = useMemo(
    () => snapshot?.nodes.find((n) => n.id === selectedId),
    [snapshot, selectedId],
  )

  const impactLevel = useMemo(() => {
    if (!selectedId || !impactResult) return 'none'
    return impactResult.graph.nodes.find((n) => n.id === selectedId)?.data.impactLevel ?? 'none'
  }, [selectedId, impactResult])

  const { upstream, downstream } = useMemo(() => {
    if (!selectedId) return { upstream: [], downstream: [] }
    return {
      upstream: (snapshot?.edges ?? []).filter((e) => e.target === selectedId),
      downstream: (snapshot?.edges ?? []).filter((e) => e.source === selectedId),
    }
  }, [snapshot, selectedId])

  if (!selectedId || !node) {
    return (
      <StateMessage
        kind="empty"
        title="No resource selected"
        detail="Click a node in the topology to inspect it."
      />
    )
  }

  const Icon = kindIconFor(node.kind)

  const handleAnalyze = () => {
    setTarget(selectedId)
    void analyze()
  }

  return (
    <div className="flex h-full flex-col overflow-hidden">
      {/* Header */}
      <div className="flex items-start gap-2 border-b bg-background px-3 py-3">
        <div className="mt-0.5 grid size-8 shrink-0 place-items-center rounded-sm bg-surface-2 text-primary">
          <Icon className="size-4" />
        </div>
        <div className="min-w-0 flex-1">
          <p className="truncate font-mono text-sm">{node.name}</p>
          <p className="label-xs truncate">
            {node.kind} · {node.namespace}
          </p>
        </div>
        <button
          onClick={clear}
          title="Close details"
          className="rounded-sm p-1 text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
        >
          <X className="size-3.5" />
        </button>
      </div>

      {/* Scrollable body */}
      <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-3 py-3">
        <Section title="Status">
          <div className="space-y-1.5">
            <div className="flex items-center gap-2">
              <StatusBadge status={node.status} />
              {node.metrics.map((m) => (
                <span key={m.label} className="font-mono text-[11px] text-muted-foreground">
                  {m.label}: {m.value}
                </span>
              ))}
            </div>
            {node.reason && <p className="text-xs text-foreground">Reason: {node.reason}</p>}
            {node.metrics.find((m) => m.label === 'Age') && (
              <p className="flex items-center gap-1 text-[11px] text-muted-foreground">
                <Clock className="size-3" />
                Age: {node.metrics.find((m) => m.label === 'Age')?.value}
              </p>
            )}
          </div>
        </Section>

        <Section title="Metadata">
          <div className="space-y-1">
            <InfoRow label="Kind" value={node.kind} />
            <InfoRow label="Name" value={node.name} />
            <InfoRow label="Namespace" value={node.namespace} />
            <InfoRow label="Resource ID" value={node.id} />
            {node.replicas !== undefined && <InfoRow label="Replicas" value={`×${node.replicas}`} />}
          </div>
        </Section>

        {Object.keys(node.labels).length > 0 && (
          <Section title="Labels">
            <div className="flex flex-wrap gap-1">
              {Object.entries(node.labels).map(([k, v]) => (
                <span
                  key={k}
                  className="rounded-sm bg-muted px-1.5 py-0.5 font-mono text-[10.5px] text-muted-foreground"
                >
                  {k}={v}
                </span>
              ))}
            </div>
          </Section>
        )}

        <Section title="Impact Level">
          <span
            className={cn(
              'inline-flex rounded-sm border px-2 py-0.5 font-mono text-[10px] uppercase tracking-wider',
              impactBadge[impactLevel] ?? impactBadge.none,
            )}
          >
            {impactLevel}
          </span>
        </Section>

        <Section title="Dependencies">
          <div className="space-y-3">
            <ConnectionList edges={upstream} direction="upstream" onSelect={select} />
            <ConnectionList edges={downstream} direction="downstream" onSelect={select} />
          </div>
        </Section>
      </div>

      {/* Footer action */}
      <div className="border-t p-3">
        <Button variant="primary" onClick={handleAnalyze} className="w-full justify-center">
          <Play className="size-3.5" />
          Analyze Impact
        </Button>
      </div>
    </div>
  )
}
