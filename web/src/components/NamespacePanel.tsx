import { useState } from 'react'
import { clsx } from 'clsx'
import {
  Box,
  ChevronDown,
  ChevronRight,
  Globe,
  Network,
  Package,
  Server,
  Shield,
  Users,
} from 'lucide-react'
import { useImpact } from '../hooks/useImpact'
import { useNamespaceResources, useNamespaces } from '../hooks/useNamespaces'
import type { ResourceStatus, StatusPhase } from '../types'

// ── phase → colour / label ──────────────────────────────────────────────────

const phaseDot: Record<StatusPhase, string> = {
  Healthy:   'bg-emerald-400',
  Degraded:  'bg-red-500',
  Warning:   'bg-amber-400',
  Pending:   'bg-sky-400',
  Completed: 'bg-slate-300',
  Unknown:   'bg-slate-300',
}

const phaseBadge: Record<StatusPhase, string> = {
  Healthy:   'bg-emerald-50 text-emerald-700 border-emerald-200',
  Degraded:  'bg-red-50 text-red-700 border-red-200',
  Warning:   'bg-amber-50 text-amber-700 border-amber-200',
  Pending:   'bg-sky-50 text-sky-700 border-sky-200',
  Completed: 'bg-slate-50 text-slate-500 border-slate-200',
  Unknown:   'bg-slate-50 text-slate-500 border-slate-200',
}

// ── kind → icon ─────────────────────────────────────────────────────────────

const kindIcon: Record<string, React.ReactNode> = {
  Ingress:                   <Globe className="w-3 h-3 text-purple-500" />,
  Service:                   <Network className="w-3 h-3 text-blue-500" />,
  Deployment:                <Package className="w-3 h-3 text-green-500" />,
  StatefulSet:               <Server className="w-3 h-3 text-teal-500" />,
  DaemonSet:                 <Shield className="w-3 h-3 text-indigo-500" />,
  Pod:                       <Box className="w-3 h-3 text-cyan-500" />,
  ConfigMap:                 <Box className="w-3 h-3 text-slate-400" />,
  Secret:                    <Box className="w-3 h-3 text-amber-500" />,
  HorizontalPodAutoscaler:   <Users className="w-3 h-3 text-orange-500" />,
  NetworkPolicy:             <Shield className="w-3 h-3 text-rose-500" />,
}

function kindGroupIcon(kind: string): React.ReactNode {
  return kindIcon[kind] ?? <Box className="w-3 h-3 text-slate-400" />
}

// ── helpers ─────────────────────────────────────────────────────────────────

function buildResourceId(r: ResourceStatus): string {
  return `${r.kind}/${r.namespace}/${r.name}`
}

function groupByKind(resources: ResourceStatus[]): Record<string, ResourceStatus[]> {
  return resources.reduce<Record<string, ResourceStatus[]>>((acc, r) => {
    ;(acc[r.kind] ??= []).push(r)
    return acc
  }, {})
}

// ── component ───────────────────────────────────────────────────────────────

export function NamespacePanel() {
  const { data, isLoading, error } = useNamespaces()
  const [expanded, setExpanded] = useState<string | null>(null)
  const { target, setTarget } = useImpact()

  if (isLoading) {
    return (
      <div className="p-4 text-slate-400 text-sm animate-pulse">
        Loading namespaces…
      </div>
    )
  }

  if (error) {
    return (
      <div className="p-4 text-red-500 text-xs">
        Failed to load namespaces: {error.message}
      </div>
    )
  }

  const namespaces = data?.namespaces ?? []

  return (
    <div className="h-full overflow-y-auto">
      <div className="px-3 py-2 text-xs font-semibold text-slate-400 uppercase tracking-wider border-b border-slate-100 sticky top-0 bg-white z-10">
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
          onToggle={() =>
            setExpanded((prev) => (prev === ns.name ? null : ns.name))
          }
          target={target}
          setTarget={setTarget}
        />
      ))}

      {namespaces.length === 0 && (
        <div className="p-4 text-slate-400 text-xs">No namespaces found.</div>
      )}
    </div>
  )
}

// ── namespace row (expandable) ──────────────────────────────────────────────

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
    <div className="border-b border-slate-50">
      <button
        onClick={onToggle}
        className="w-full flex items-center gap-1.5 px-2 py-2 text-xs hover:bg-slate-50 transition-colors"
      >
        {expanded ? (
          <ChevronDown className="w-3 h-3 text-slate-400 flex-shrink-0" />
        ) : (
          <ChevronRight className="w-3 h-3 text-slate-400 flex-shrink-0" />
        )}
        <span
          className={clsx('w-1.5 h-1.5 rounded-full flex-shrink-0', phaseDot[phase])}
        />
        <span className="font-mono font-medium text-slate-700 truncate flex-1 text-left">
          {name}
        </span>
        <span
          className={clsx(
            'px-1 py-0.5 rounded text-[9px] font-medium border flex-shrink-0',
            phaseBadge[phase],
          )}
        >
          {phase}
        </span>
        <span className="text-[9px] text-slate-400 flex-shrink-0 ml-0.5">
          {resourceCount} · {age}
        </span>
      </button>

      {expanded && (
        <NamespaceResources
          namespace={name}
          target={target}
          setTarget={setTarget}
        />
      )}
    </div>
  )
}

// ── resources under an expanded namespace ───────────────────────────────────

interface NamespaceResourcesProps {
  namespace: string
  target: string | null
  setTarget: (t: string) => void
}

function NamespaceResources({
  namespace,
  target,
  setTarget,
}: NamespaceResourcesProps) {
  const { data, isLoading, error } = useNamespaceResources(namespace)

  if (isLoading) {
    return (
      <div className="px-8 py-2 text-slate-400 text-[11px] animate-pulse">
        Loading resources…
      </div>
    )
  }

  if (error) {
    return (
      <div className="px-8 py-2 text-red-500 text-[11px]">
        {error.message}
      </div>
    )
  }

  const resources = data?.resources ?? []
  if (resources.length === 0) {
    return (
      <div className="px-8 py-2 text-slate-400 text-[11px]">
        No resources in this namespace.
      </div>
    )
  }

  const grouped = groupByKind(resources)
  // Sort kinds for stable order.
  const kinds = Object.keys(grouped).sort()

  return (
    <div className="pb-2">
      {kinds.map((kind) => (
        <div key={kind}>
          <div className="flex items-center gap-1.5 px-6 py-1 text-[10px] font-medium text-slate-400">
            {kindGroupIcon(kind)}
            {kind} ({grouped[kind].length})
          </div>
          {grouped[kind].map((r) => {
            const id = buildResourceId(r)
            return (
              <button
                key={id}
                onClick={() => setTarget(id)}
                title={r.message}
                className={clsx(
                  'w-full flex items-center gap-1.5 px-8 py-1 text-[11px] transition-colors',
                  target === id
                    ? 'bg-blue-50 text-blue-700 font-medium'
                    : 'text-slate-600 hover:bg-slate-50',
                )}
              >
                <span
                  className={clsx(
                    'w-1.5 h-1.5 rounded-full flex-shrink-0',
                    phaseDot[r.phase],
                  )}
                />
                <span className="truncate flex-1 text-left">{r.name}</span>
                <span className="text-[9px] text-slate-400 flex-shrink-0">
                  {r.ready}
                </span>
              </button>
            )
          })}
        </div>
      ))}
    </div>
  )
}
