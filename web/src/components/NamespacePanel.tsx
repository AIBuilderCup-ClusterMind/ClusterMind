import { useState } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { cn } from '../lib/utils'
import { kindIconFor } from './cm/primitives'
import { useImpact } from '../hooks/useImpact'
import { useNamespaceResources, useNamespaces } from '../hooks/useNamespaces'
import type { ResourceStatus, StatusPhase } from '../types'

// ── phase → dot / badge ────────────────────────────────────────────────────

const phaseDot: Record<StatusPhase, string> = {
  Healthy: 'bg-healthy',
  Degraded: 'bg-critical',
  Warning: 'bg-warning',
  Pending: 'bg-primary',
  Completed: 'bg-unknown',
  Unknown: 'bg-unknown',
}

const phaseBadge: Record<StatusPhase, string> = {
  Healthy: 'bg-healthy/10 text-healthy border-healthy/30',
  Degraded: 'bg-critical/10 text-critical border-critical/30',
  Warning: 'bg-warning/10 text-warning border-warning/30',
  Pending: 'bg-primary/10 text-primary border-primary/30',
  Completed: 'bg-muted text-muted-foreground border-border-strong',
  Unknown: 'bg-muted text-muted-foreground border-border-strong',
}

// ── helpers ────────────────────────────────────────────────────────────────

function buildResourceId(r: ResourceStatus): string {
  return `${r.kind}/${r.namespace}/${r.name}`
}

function groupByKind(resources: ResourceStatus[]): Record<string, ResourceStatus[]> {
  return resources.reduce<Record<string, ResourceStatus[]>>((acc, r) => {
    ;(acc[r.kind] ??= []).push(r)
    return acc
  }, {})
}

// ── component ──────────────────────────────────────────────────────────────

export function NamespacePanel() {
  const { data, isLoading, error } = useNamespaces()
  const [expanded, setExpanded] = useState<string | null>(null)
  const { target, setTarget } = useImpact()

  if (isLoading) {
    return <div className="p-4 text-sm text-muted-foreground animate-pulse">Loading namespaces…</div>
  }

  if (error) {
    return <div className="p-4 text-xs text-critical">Failed to load namespaces: {error.message}</div>
  }

  const namespaces = data?.namespaces ?? []

  return (
    <div className="h-full overflow-y-auto">
      <div className="label-xs sticky top-0 z-10 border-b bg-surface px-3 py-2">
        Namespaces ({namespaces.length})
      </div>

      {namespaces.map((ns) => (
        <NamespaceRow
          key={ns.name}
          name={ns.name}
          phase={ns.phase}
          resourceCount={ns.resourceCount}
          age={ns.age}
          expanded={expanded === ns.name}
          onToggle={() => setExpanded((prev) => (prev === ns.name ? null : ns.name))}
          target={target}
          setTarget={setTarget}
        />
      ))}

      {namespaces.length === 0 && (
        <div className="p-4 text-xs text-muted-foreground">No namespaces found.</div>
      )}
    </div>
  )
}

// ── namespace row (expandable) ─────────────────────────────────────────────

interface NamespaceRowProps {
  name: string
  phase: StatusPhase
  resourceCount: number
  age: string
  expanded: boolean
  onToggle: () => void
  target: string | null
  setTarget: (t: string) => void
}

function NamespaceRow({
  name,
  phase,
  resourceCount,
  age,
  expanded,
  onToggle,
  target,
  setTarget,
}: NamespaceRowProps) {
  return (
    <div className="border-b border-border/60">
      <button
        onClick={onToggle}
        className="flex w-full items-center gap-1.5 px-2 py-2 text-xs transition-colors hover:bg-accent"
      >
        {expanded ? (
          <ChevronDown className="size-3 shrink-0 text-muted-foreground" />
        ) : (
          <ChevronRight className="size-3 shrink-0 text-muted-foreground" />
        )}
        <span className={cn('size-1.5 shrink-0 rounded-full', phaseDot[phase])} />
        <span className="min-w-0 flex-1 truncate text-left font-mono font-medium text-foreground">
          {name}
        </span>
        <span
          className={cn(
            'flex-shrink-0 rounded-sm border px-1 py-0.5 font-mono text-[9px] font-medium',
            phaseBadge[phase],
          )}
        >
          {phase}
        </span>
        <span className="ml-0.5 flex-shrink-0 font-mono text-[9px] text-muted-foreground">
          {resourceCount} · {age}
        </span>
      </button>

      {expanded && (
        <NamespaceResources namespace={name} target={target} setTarget={setTarget} />
      )}
    </div>
  )
}

// ── resources under an expanded namespace ──────────────────────────────────

interface NamespaceResourcesProps {
  namespace: string
  target: string | null
  setTarget: (t: string) => void
}

function NamespaceResources({ namespace, target, setTarget }: NamespaceResourcesProps) {
  const { data, isLoading, error } = useNamespaceResources(namespace)

  if (isLoading) {
    return <div className="animate-pulse px-8 py-2 text-[11px] text-muted-foreground">Loading resources…</div>
  }

  if (error) {
    return <div className="px-8 py-2 text-[11px] text-critical">{error.message}</div>
  }

  const resources = data?.resources ?? []
  if (resources.length === 0) {
    return <div className="px-8 py-2 text-[11px] text-muted-foreground">No resources in this namespace.</div>
  }

  const grouped = groupByKind(resources)
  const kinds = Object.keys(grouped).sort()

  return (
    <div className="pb-2">
      {kinds.map((kind) => {
        const Icon = kindIconFor(kind)
        return (
          <div key={kind}>
            <div className="flex items-center gap-1.5 px-6 py-1 text-[10px] font-medium text-muted-foreground">
              <Icon className="size-3" />
              {kind} ({grouped[kind].length})
            </div>
            {grouped[kind].map((r) => {
              const id = buildResourceId(r)
              return (
                <button
                  key={id}
                  onClick={() => setTarget(id)}
                  title={r.message}
                  className={cn(
                    'flex w-full items-center gap-1.5 px-8 py-1 text-[11px] transition-colors',
                    target === id
                      ? 'bg-accent font-medium text-primary'
                      : 'text-muted-foreground hover:bg-accent hover:text-foreground',
                  )}
                >
                  <span className={cn('size-1.5 shrink-0 rounded-full', phaseDot[r.phase])} />
                  <span className="min-w-0 flex-1 truncate text-left font-mono">{r.name}</span>
                  <span className="shrink-0 font-mono text-[9px] text-muted-foreground">
                    {r.ready}
                  </span>
                </button>
              )
            })}
          </div>
        )
      })}
    </div>
  )
}
