import { useCallback } from 'react'
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  Handle,
  Position,
  type NodeProps,
  BackgroundVariant,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { Globe, Package, Network, Box, Server } from 'lucide-react'
import { clsx } from 'clsx'
import { useCluster } from '../hooks/useCluster'
import { useImpact } from '../hooks/useImpact'
import { useSelection } from '../hooks/useSelection'
import type { FlowNodeData } from '../types'

// ── Impact-level → Tailwind classes ──────────────────────────────────────────

const impactClasses: Record<string, string> = {
  critical: 'border-red-500 bg-red-50 shadow-lg shadow-red-100 animate-pulse',
  direct:   'border-orange-400 bg-orange-50 shadow-md shadow-orange-100',
  indirect: 'border-yellow-400 bg-yellow-50',
  none:     'border-slate-200 bg-white hover:border-slate-400 hover:shadow-sm',
}

// ── Shared node chrome ────────────────────────────────────────────────────────

interface NodeShellProps {
  data: FlowNodeData
  icon: React.ReactNode
  accentColor: string
  selected?: boolean
}

function NodeShell({ data, icon, accentColor, selected }: NodeShellProps) {
  return (
    <div
      className={clsx(
        'px-3 py-2 rounded-xl border-2 min-w-[140px] max-w-[200px] transition-all cursor-pointer select-none',
        impactClasses[data.impactLevel] ?? impactClasses.none,
        selected && 'ring-2 ring-blue-500 ring-offset-2 ring-offset-slate-50',
      )}
    >
      <Handle type="target" position={Position.Top} className="!bg-slate-300" />
      <div className="flex items-center gap-2 mb-0.5">
        <span className={accentColor}>{icon}</span>
        <span className="font-mono text-xs font-semibold text-slate-800 truncate">
          {data.label}
        </span>
      </div>
      <div className="flex items-center justify-between">
        <span className="text-[10px] text-slate-400 truncate">{data.namespace}</span>
        {data.replicas !== undefined && (
          <span className="text-[10px] text-slate-400">×{data.replicas}</span>
        )}
      </div>
      <Handle type="source" position={Position.Bottom} className="!bg-slate-300" />
    </div>
  )
}

// ── Custom node types ─────────────────────────────────────────────────────────

function ServiceNode({ data, selected }: NodeProps) {
  return (
    <NodeShell
      data={data as FlowNodeData}
      icon={<Network className="w-4 h-4" />}
      accentColor="text-blue-500"
      selected={selected}
    />
  )
}

function DeploymentNode({ data, selected }: NodeProps) {
  return (
    <NodeShell
      data={data as FlowNodeData}
      icon={<Package className="w-4 h-4" />}
      accentColor="text-green-500"
      selected={selected}
    />
  )
}

function IngressNode({ data, selected }: NodeProps) {
  return (
    <NodeShell
      data={data as FlowNodeData}
      icon={<Globe className="w-4 h-4" />}
      accentColor="text-purple-500"
      selected={selected}
    />
  )
}

function StatefulSetNode({ data, selected }: NodeProps) {
  return (
    <NodeShell
      data={data as FlowNodeData}
      icon={<Server className="w-4 h-4" />}
      accentColor="text-teal-500"
      selected={selected}
    />
  )
}

function DefaultNode({ data, selected }: NodeProps) {
  return (
    <NodeShell
      data={data as FlowNodeData}
      icon={<Box className="w-4 h-4" />}
      accentColor="text-slate-500"
      selected={selected}
    />
  )
}

const nodeTypes = {
  service:     ServiceNode,
  deployment:  DeploymentNode,
  ingress:     IngressNode,
  statefulset: StatefulSetNode,
  default:     DefaultNode,
}

// ── Main graph canvas ─────────────────────────────────────────────────────────

export function ClusterGraph() {
  const { nodes: clusterNodes, edges: clusterEdges, isLoading, error } = useCluster()
  const { result } = useImpact()
  const { selectedId, select, clear } = useSelection()

  // When an impact analysis result exists, use its enriched graph (coloured nodes)
  const baseNodes = result?.graph.nodes ?? clusterNodes
  const displayEdges = result?.graph.edges ?? clusterEdges

  // Sync selection state into nodes for the ring highlight.
  const displayNodes = baseNodes.map((n) => ({
    ...n,
    selected: n.id === selectedId,
  }))

  const handleNodeClick = useCallback(
    (_: React.MouseEvent, node: { id: string; data: unknown }) => {
      select(node.id, node.data as FlowNodeData)
    },
    [select],
  )

  const handlePaneClick = useCallback(() => {
    clear()
  }, [clear])

  if (isLoading) {
    return (
      <div className="flex-1 flex items-center justify-center bg-slate-50">
        <div className="text-center space-y-3">
          <div className="w-10 h-10 border-4 border-blue-500 border-t-transparent rounded-full animate-spin mx-auto" />
          <p className="text-sm text-slate-500">Collecting cluster resources…</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex-1 flex items-center justify-center bg-slate-50">
        <div className="text-center space-y-2">
          <p className="text-red-500 font-medium">Failed to load cluster graph</p>
          <p className="text-sm text-slate-400">{error.message}</p>
        </div>
      </div>
    )
  }

  if (displayNodes.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center bg-slate-50">
        <p className="text-sm text-slate-400">No resources found in cluster.</p>
      </div>
    )
  }

  return (
    <div className="flex-1 h-full">
      <ReactFlow
        nodes={displayNodes}
        edges={displayEdges}
        nodeTypes={nodeTypes}
        onNodeClick={handleNodeClick}
        onPaneClick={handlePaneClick}
        fitView
        fitViewOptions={{ padding: 0.2 }}
        minZoom={0.1}
        maxZoom={2}
        attributionPosition="bottom-right"
      >
        <Background variant={BackgroundVariant.Dots} gap={16} size={1} color="#e2e8f0" />
        <Controls />
        <MiniMap
          nodeStrokeWidth={3}
          zoomable
          pannable
          className="!border-slate-200 !rounded-lg"
        />
      </ReactFlow>
    </div>
  )
}
