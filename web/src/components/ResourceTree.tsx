import { Globe, Package, Network, Box, Server, Shield } from 'lucide-react'
import { clsx } from 'clsx'
import { useCluster } from '../hooks/useCluster'
import { useImpact } from '../hooks/useImpact'
import type { FlowNode } from '../types'

const kindIcon: Record<string, React.ReactNode> = {
  Ingress:     <Globe className="w-3.5 h-3.5 text-purple-500" />,
  Service:     <Network className="w-3.5 h-3.5 text-blue-500" />,
  Deployment:  <Package className="w-3.5 h-3.5 text-green-500" />,
  StatefulSet: <Server className="w-3.5 h-3.5 text-teal-500" />,
  DaemonSet:   <Shield className="w-3.5 h-3.5 text-indigo-500" />,
}

function groupByKind(nodes: FlowNode[]): Record<string, FlowNode[]> {
  return nodes.reduce<Record<string, FlowNode[]>>((acc, n) => {
    const kind = n.data.kind || n.type
    if (!acc[kind]) acc[kind] = []
    acc[kind].push(n)
    return acc
  }, {})
}

export function ResourceTree() {
  const { nodes, isLoading } = useCluster()
  const { target, setTarget } = useImpact()

  if (isLoading) {
    return (
      <div className="p-4 text-slate-400 text-sm animate-pulse">
        Loading resources…
      </div>
    )
  }

  const grouped = groupByKind(nodes)

  return (
    <div className="h-full overflow-y-auto">
      <div className="px-3 py-2 text-xs font-semibold text-slate-400 uppercase tracking-wider border-b border-slate-100">
        Resources ({nodes.length})
      </div>
      {Object.entries(grouped).map(([kind, kindNodes]) => (
        <div key={kind} className="mb-1">
          <div className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-slate-500">
            {kindIcon[kind] ?? <Box className="w-3.5 h-3.5 text-slate-400" />}
            {kind} ({kindNodes.length})
          </div>
          {kindNodes.map((node) => (
            <button
              key={node.id}
              onClick={() => setTarget(node.id)}
              className={clsx(
                'w-full text-left px-4 py-1.5 text-xs truncate transition-colors',
                target === node.id
                  ? 'bg-blue-50 text-blue-700 font-medium'
                  : 'text-slate-600 hover:bg-slate-50',
              )}
              title={node.id}
            >
              {node.data.label}
              {node.data.namespace && (
                <span className="ml-1 text-slate-400">/{node.data.namespace}</span>
              )}
            </button>
          ))}
        </div>
      ))}
    </div>
  )
}
