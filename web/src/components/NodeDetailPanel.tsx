import { useMemo } from 'react'
import {
  ArrowDown,
  ArrowUp,
  Box,
  Clock,
  FileCode,
  Globe,
  Network,
  Package,
  Play,
  Server,
  Shield,
  Users,
  X,
} from 'lucide-react'
import { clsx } from 'clsx'
import { useCluster } from '../hooks/useCluster'
import { useImpact } from '../hooks/useImpact'
import { useNamespaceResources } from '../hooks/useNamespaces'
import { useSelection } from '../hooks/useSelection'
import type { FlowEdge, FlowNodeData, StatusPhase } from '../types'

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

const impactBadge: Record<string, string> = {
  critical: 'bg-red-50 text-red-700 border-red-200',
  direct:   'bg-orange-50 text-orange-700 border-orange-200',
  indirect: 'bg-yellow-50 text-yellow-700 border-yellow-200',
  none:     'bg-slate-50 text-slate-500 border-slate-200',
}

// ── kind → icon ─────────────────────────────────────────────────────────────

const kindIcon: Record<string, React.ReactNode> = {
  Ingress:                 <Globe className="w-4 h-4 text-purple-500" />,
  Service:                 <Network className="w-4 h-4 text-blue-500" />,
  Deployment:              <Package className="w-4 h-4 text-green-500" />,
  StatefulSet:             <Server className="w-4 h-4 text-teal-500" />,
  DaemonSet:               <Shield className="w-4 h-4 text-indigo-500" />,
  Pod:                     <Box className="w-4 h-4 text-cyan-500" />,
  ConfigMap:               <FileCode className="w-4 h-4 text-slate-400" />,
  Secret:                  <FileCode className="w-4 h-4 text-amber-500" />,
  HorizontalPodAutoscaler: <Users className="w-4 h-4 text-orange-500" />,
  NetworkPolicy:           <Shield className="w-4 h-4 text-rose-500" />,
}

function kindIconFor(kind: string): React.ReactNode {
  return kindIcon[kind] ?? <Box className="w-4 h-4 text-slate-400" />
}

// ── section helper ──────────────────────────────────────────────────────────

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1.5">
      <p className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider">
        {title}
      </p>
      {children}
    </div>
  )
}

function InfoRow({ label, value }: { label: string; value: React.ReactNode }) {
  if (value === null || value === undefined || value === '') return null
  return (
    <div className="flex items-start gap-2 text-xs">
      <span className="text-slate-400 w-20 flex-shrink-0">{label}</span>
      <span className="text-slate-700 font-mono break-all">{value}</span>
    </div>
  )
}

// ── connection list ─────────────────────────────────────────────────────────

function ConnectionList({
  edges,
  direction,
  onSelect,
}: {
  edges: FlowEdge[]
  direction: 'upstream' | 'downstream'
  onSelect: (id: string) => void
}) {
  const Icon = direction === 'upstream' ? ArrowUp : ArrowDown
  const label = direction === 'upstream' ? 'Upstream' : 'Downstream'

  if (edges.length === 0) {
    return (
      <p className="text-xs text-slate-400 italic">No {label} connections.</p>
    )
  }

  return (
    <div className="space-y-1">
      <div className="flex items-center gap-1 text-[11px] text-slate-500">
        <Icon className="w-3 h-3" />
        {label} ({edges.length})
      </div>
      {edges.map((e) => {
        const peerId = direction === 'upstream' ? e.source : e.target
        const parts = peerId.split('/')
        const kind = parts[0]
        const name = parts.slice(2).join('/') || parts[1]
        return (
          <button
            key={e.id}
            onClick={() => onSelect(peerId)}
            className="w-full flex items-center gap-1.5 text-xs text-slate-600 bg-slate-50 hover:bg-slate-100 rounded-lg px-2.5 py-1.5 transition-colors text-left"
            title={peerId}
          >
            <span className="flex-shrink-0">{kindIconFor(kind)}</span>
            <span className="font-mono truncate flex-1">{name}</span>
            <span className="text-[9px] text-slate-400 flex-shrink-0">{kind}</span>
          </button>
        )
      })}
    </div>
  )
}

// ── main panel ──────────────────────────────────────────────────────────────

export function NodeDetailPanel() {
  const { selectedId, selectedData, clear, select } = useSelection()
  const { nodes: graphNodes, edges } = useCluster()
  const { setTarget, analyze } = useImpact()

  // Parse "Kind/namespace/name" from the node ID.
  const parsed = useMemo(() => {
    if (!selectedId) return null
    const parts = selectedId.split('/')
    const kind = parts[0] ?? ''
    const namespace = parts[1] ?? ''
    const name = parts.slice(2).join('/') || (parts[1] ?? '')
    return { kind, namespace, name }
  }, [selectedId])

  // Fetch live status for this resource from the namespace-resources API.
  const { data: nsResources, isLoading: statusLoading } = useNamespaceResources(
    parsed?.namespace ?? null,
    parsed?.kind,
  )

  const liveStatus = useMemo(() => {
    if (!parsed || !nsResources) return null
    return (
      nsResources.resources.find(
        (r) => r.kind === parsed.kind && r.name === parsed.name,
      ) ?? null
    )
  }, [parsed, nsResources])

  // Compute upstream / downstream edges for the selected node.
  const { upstream, downstream } = useMemo(() => {
    if (!selectedId) return { upstream: [], downstream: [] }
    return {
      upstream: edges.filter((e) => e.target === selectedId),
      downstream: edges.filter((e) => e.source === selectedId),
    }
  }, [edges, selectedId])

  // Select a peer node by looking up its FlowNodeData in the current graph.
  const selectPeer = (peerId: string) => {
    const peer = graphNodes.find((n) => n.id === peerId)
    if (peer) {
      select(peerId, peer.data)
    }
  }

  // ── empty state ──────────────────────────────────────────────────────────
  if (!selectedId || !selectedData) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center text-center px-4 py-8 space-y-2">
        <Box className="w-8 h-8 text-slate-200" />
        <p className="text-xs text-slate-400">
          Click a node on the graph to view its details.
        </p>
      </div>
    )
  }

  const data: FlowNodeData = selectedData

  const handleAnalyze = () => {
    setTarget(selectedId)
    void analyze()
  }

  // ── detail view ──────────────────────────────────────────────────────────
  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      {/* Header */}
      <div className="px-4 py-3 border-b border-slate-100 bg-slate-50 flex items-start gap-2">
        <span className="mt-0.5 flex-shrink-0">{kindIconFor(data.kind)}</span>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-semibold text-slate-800 font-mono truncate">
            {data.label}
          </p>
          <p className="text-[11px] text-slate-400 font-mono truncate">
            {data.kind} · {data.namespace}
          </p>
        </div>
        <button
          onClick={clear}
          className="p-1 rounded-lg text-slate-400 hover:bg-slate-200 hover:text-slate-600 transition-colors flex-shrink-0"
          title="Close details"
        >
          <X className="w-3.5 h-3.5" />
        </button>
      </div>

      {/* Scrollable body */}
      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-4">
        {/* Status (live) */}
        <Section title="Status">
          {statusLoading && (
            <div className="flex items-center gap-2 text-xs text-slate-400 animate-pulse">
              <div className="w-3 h-3 border-2 border-slate-300 border-t-transparent rounded-full animate-spin" />
              Fetching live status…
            </div>
          )}
          {!statusLoading && liveStatus && (
            <div className="space-y-1.5">
              <div className="flex items-center gap-2">
                <span
                  className={clsx(
                    'w-2 h-2 rounded-full flex-shrink-0',
                    phaseDot[liveStatus.phase],
                  )}
                />
                <span
                  className={clsx(
                    'px-2 py-0.5 rounded-full text-[10px] font-semibold border',
                    phaseBadge[liveStatus.phase],
                  )}
                >
                  {liveStatus.phase}
                </span>
                {liveStatus.ready !== '—' && (
                  <span className="text-[11px] text-slate-500 font-mono">
                    {liveStatus.ready}
                  </span>
                )}
              </div>
              {liveStatus.reason && (
                <p className="text-xs text-slate-600">
                  <span className="text-slate-400">Reason: </span>
                  {liveStatus.reason}
                </p>
              )}
              {liveStatus.message && (
                <p className="text-xs text-slate-500 leading-relaxed">
                  {liveStatus.message}
                </p>
              )}
              {liveStatus.age && liveStatus.age !== '—' && (
                <p className="flex items-center gap-1 text-[11px] text-slate-400">
                  <Clock className="w-3 h-3" />
                  Age: {liveStatus.age}
                </p>
              )}
            </div>
          )}
          {!statusLoading && !liveStatus && (
            <p className="text-xs text-slate-400 italic">
              No live status available for this resource.
            </p>
          )}
        </Section>

        {/* Metadata */}
        <Section title="Metadata">
          <div className="space-y-1">
            <InfoRow label="Kind" value={data.kind} />
            <InfoRow label="Name" value={data.label} />
            <InfoRow label="Namespace" value={data.namespace} />
            <InfoRow label="Resource ID" value={selectedId} />
            {data.replicas !== undefined && (
              <InfoRow label="Replicas" value={`×${data.replicas}`} />
            )}
          </div>
        </Section>

        {/* Impact level */}
        <Section title="Impact Level">
          <span
            className={clsx(
              'inline-flex items-center px-2.5 py-1 rounded-full text-[11px] font-semibold border uppercase tracking-wide',
              impactBadge[data.impactLevel] ?? impactBadge.none,
            )}
          >
            {data.impactLevel}
          </span>
        </Section>

        {/* Connections */}
        <Section title="Dependencies">
          <div className="space-y-3">
            <ConnectionList
              edges={upstream}
              direction="upstream"
              onSelect={selectPeer}
            />
            <ConnectionList
              edges={downstream}
              direction="downstream"
              onSelect={selectPeer}
            />
          </div>
        </Section>
      </div>

      {/* Footer action */}
      <div className="px-4 py-3 border-t border-slate-100">
        <button
          onClick={handleAnalyze}
          className="w-full flex items-center justify-center gap-1.5 px-3 py-2 rounded-lg text-xs font-medium bg-blue-600 text-white hover:bg-blue-700 shadow-sm transition-colors"
        >
          <Play className="w-3 h-3" />
          Analyze Impact
        </button>
      </div>
    </div>
  )
}
