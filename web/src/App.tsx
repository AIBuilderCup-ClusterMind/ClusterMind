import { Layers } from 'lucide-react'
import { ClusterGraph } from './components/ClusterGraph'
import { NamespacePanel } from './components/NamespacePanel'
import { RightPanel } from './components/RightPanel'
import { useCluster } from './hooks/useCluster'

function Header() {
  const { nodes } = useCluster()
  const namespaces = [...new Set(nodes.map((n) => n.data.namespace).filter(Boolean))]

  return (
    <header className="h-12 flex items-center justify-between px-4 border-b border-slate-200 bg-white shadow-sm flex-shrink-0">
      <div className="flex items-center gap-2.5">
        <div className="w-7 h-7 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-lg flex items-center justify-center">
          <Layers className="w-4 h-4 text-white" />
        </div>
        <div>
          <h1 className="text-sm font-bold text-slate-800 leading-none">ClusterMind</h1>
          <p className="text-[10px] text-slate-400 leading-none mt-0.5">
            AI Infrastructure Digital Twin
          </p>
        </div>
      </div>
      {namespaces.length > 0 && (
        <div className="flex items-center gap-1.5">
          {namespaces.map((ns) => (
            <span
              key={ns}
              className="text-[10px] bg-slate-100 text-slate-500 px-2 py-0.5 rounded-full font-mono"
            >
              {ns}
            </span>
          ))}
        </div>
      )}
    </header>
  )
}

export default function App() {
  return (
    <div className="h-screen flex flex-col bg-slate-50 overflow-hidden">
      <Header />
      <div className="flex flex-1 min-h-0">
        {/* Left: Namespace & resource status sidebar */}
        <aside className="w-72 flex-shrink-0 border-r border-slate-200 bg-white overflow-hidden">
          <NamespacePanel />
        </aside>

        {/* Center: React Flow canvas */}
        <main className="flex-1 flex overflow-hidden">
          <ClusterGraph />
        </main>

        {/* Right: Details / Impact tabs */}
        <RightPanel />
      </div>
    </div>
  )
}
