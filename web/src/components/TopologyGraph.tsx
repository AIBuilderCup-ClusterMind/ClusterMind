import { useMemo } from 'react'
import {
  Background,
  BackgroundVariant,
  Controls,
  Handle,
  MarkerType,
  MiniMap,
  Position,
  ReactFlow,
  type Edge,
  type Node,
  type NodeProps,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import { cn } from '../lib/utils'
import { kindIconFor, statusDot } from './cm/primitives'
import { useTwin } from '../hooks/useTwin'
import { useImpact } from '../hooks/useImpact'
import { useSelection } from '../hooks/useSelection'
import type { HealthStatus, TwinNode } from '../types'

type ImpactLevel = 'critical' | 'direct' | 'indirect' | 'none'

interface ResourceNodeData extends Record<string, unknown> {
  resource: TwinNode
  impactLevel: ImpactLevel
}

const impactBorder: Record<ImpactLevel, string> = {
  critical: 'border-critical bg-critical/10 animate-pulse',
  direct: 'border-warning bg-warning/10',
  indirect: 'border-dashed border-warning/70 bg-warning/5',
  none: 'border-border-strong bg-surface-2 hover:border-primary/50',
}

function ResourceNodeView({ data, selected }: NodeProps<Node<ResourceNodeData>>) {
  const { resource: r, impactLevel } = data
  const Icon = kindIconFor(r.kind)
  return (
    <div
      title={`${r.kind}: ${r.name} (${r.status})`}
      className={cn(
        'w-[196px] rounded-md border px-2.5 py-2 transition-all',
        selected ? 'ring-2 ring-primary/40' : undefined,
        impactBorder[impactLevel],
      )}
      style={impactLevel === 'none' ? { backgroundColor: 'var(--color-surface-2)' } : undefined}
    >
      <Handle
        type="target"
        position={Position.Top}
        className="!size-1.5 !border-0 !bg-border-strong"
      />
      <div className="flex items-center gap-2">
        <div className="grid size-6 shrink-0 place-items-center rounded-sm bg-background text-primary">
          <Icon className="size-3.5" />
        </div>
        <div className="min-w-0 flex-1">
          <p className="label-xs !text-[9px]">{r.kind}</p>
          <p className="truncate font-mono text-[11.5px] text-foreground">{r.name}</p>
        </div>
        <span
          className={cn('size-2 shrink-0 rounded-full', statusDot[r.status as HealthStatus])}
          aria-label={r.status}
        />
      </div>
      {r.metrics.length > 0 && (
        <div className="mt-1.5 flex gap-3 border-t pt-1.5">
          {r.metrics.slice(0, 2).map((m) => (
            <div key={m.label} className="min-w-0">
              <p className="text-[9px] text-muted-foreground">{m.label}</p>
              <p className="font-mono text-[11px]">{m.value}</p>
            </div>
          ))}
        </div>
      )}
      <Handle
        type="source"
        position={Position.Bottom}
        className="!size-1.5 !border-0 !bg-border-strong"
      />
    </div>
  )
}

const nodeTypes = { resource: ResourceNodeView }

export function TopologyGraph() {
  const { data: snapshot, isLoading, error } = useTwin()
  const { result: impactResult } = useImpact()
  const { selectedId, select, clear } = useSelection()

  const impactById = useMemo(() => {
    const map = new Map<string, ImpactLevel>()
    for (const n of impactResult?.graph.nodes ?? []) {
      map.set(n.id, n.data.impactLevel as ImpactLevel)
    }
    return map
  }, [impactResult])

  const hotEdges = useMemo(() => {
    const set = new Set<string>()
    for (const e of impactResult?.graph.edges ?? []) {
      if (e.animated) set.add(`${e.source}->${e.target}`)
    }
    return set
  }, [impactResult])

  const { nodes, edges } = useMemo(() => {
    const rfNodes: Node<ResourceNodeData>[] = (snapshot?.nodes ?? []).map((r) => ({
      id: r.id,
      type: 'resource',
      position: r.position,
      selected: r.id === selectedId,
      data: {
        resource: r,
        impactLevel: impactById.get(r.id) ?? 'none',
      },
    }))

    const hasImpact = impactResult != null
    const rfEdges: Edge[] = (snapshot?.edges ?? []).map((e) => {
      const hot = hotEdges.has(`${e.source}->${e.target}`)
      const color = hot ? 'var(--color-critical)' : 'var(--color-border-strong)'
      return {
        id: e.id,
        source: e.source,
        target: e.target,
        label: e.protocol,
        animated: hot,
        style: {
          stroke: color,
          strokeWidth: hot ? 1.8 : 1.2,
          opacity: hasImpact && !hot ? 0.35 : 1,
        },
        markerEnd: { type: MarkerType.ArrowClosed, color, width: 16, height: 16 },
        labelBgPadding: [4, 2] as [number, number],
      }
    })

    return { nodes: rfNodes, edges: rfEdges }
  }, [snapshot, impactById, hotEdges, impactResult, selectedId])

  if (isLoading) {
    return <div className="h-full w-full grid-bg" aria-busy="true" />
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-6 text-center">
        <div className="space-y-2">
          <p className="text-sm font-medium text-critical">Failed to load cluster snapshot</p>
          <p className="text-xs text-muted-foreground">{error.message}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="h-full w-full">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        onNodeClick={(_, n) => select(n.id)}
        onPaneClick={() => clear()}
        nodesConnectable={false}
        fitView
        fitViewOptions={{ padding: 0.15 }}
        minZoom={0.2}
        maxZoom={2}
        colorMode="dark"
        style={{ background: 'transparent' }}
      >
        <Background
          variant={BackgroundVariant.Dots}
          gap={22}
          size={1}
          color="var(--color-border-strong)"
        />
        <Controls showInteractive={false} position="bottom-left" />
        <MiniMap
          pannable
          zoomable
          nodeStrokeWidth={3}
          className="!border !border-border !rounded-md"
          maskColor="oklch(0.165 0.025 258 / 0.7)"
        />
      </ReactFlow>
    </div>
  )
}
