import { AlertTriangle, Play, X, ChevronRight } from 'lucide-react'
import { clsx } from 'clsx'
import { useCluster } from '../hooks/useCluster'
import { useImpact } from '../hooks/useImpact'
import { SeverityBadge } from './SeverityBadge'

export function ImpactPanel() {
  const { nodes } = useCluster()
  const { target, result, isLoading, error, setTarget, analyze, clear } = useImpact()

  const handleAnalyze = () => {
    void analyze()
  }

  return (
    <div className="w-80 flex flex-col border-l border-slate-200 bg-white overflow-hidden">
      {/* Header */}
      <div className="px-4 py-3 border-b border-slate-100 bg-slate-50">
        <div className="flex items-center gap-2">
          <AlertTriangle className="w-4 h-4 text-orange-500" />
          <h2 className="text-sm font-semibold text-slate-700">Impact Simulator</h2>
        </div>
        <p className="text-xs text-slate-400 mt-0.5">
          Select a resource to simulate failure
        </p>
      </div>

      {/* Resource selector */}
      <div className="px-4 py-3 border-b border-slate-100 space-y-2">
        <label className="text-xs font-medium text-slate-500">Target Resource</label>
        <select
          value={target ?? ''}
          onChange={(e) => setTarget(e.target.value)}
          className="w-full text-xs border border-slate-200 rounded-lg px-3 py-2 bg-white text-slate-700 focus:outline-none focus:ring-2 focus:ring-blue-400 focus:border-transparent"
        >
          <option value="">Select resource…</option>
          {nodes.map((n) => (
            <option key={n.id} value={n.id}>
              [{n.data.kind}] {n.data.label}
            </option>
          ))}
        </select>

        <div className="flex gap-2">
          <button
            onClick={handleAnalyze}
            disabled={!target || isLoading}
            className={clsx(
              'flex-1 flex items-center justify-center gap-1.5 px-3 py-2 rounded-lg text-xs font-medium transition-all',
              target && !isLoading
                ? 'bg-blue-600 text-white hover:bg-blue-700 shadow-sm'
                : 'bg-slate-100 text-slate-400 cursor-not-allowed',
            )}
          >
            {isLoading ? (
              <>
                <div className="w-3 h-3 border-2 border-white border-t-transparent rounded-full animate-spin" />
                Analyzing…
              </>
            ) : (
              <>
                <Play className="w-3 h-3" />
                Analyze
              </>
            )}
          </button>
          {result && (
            <button
              onClick={clear}
              className="px-3 py-2 rounded-lg text-xs font-medium text-slate-500 hover:bg-slate-100 transition-colors"
            >
              <X className="w-3 h-3" />
            </button>
          )}
        </div>

        {error && (
          <p className="text-xs text-red-500 bg-red-50 rounded-lg px-3 py-2">{error}</p>
        )}
      </div>

      {/* Result */}
      {result && (
        <div className="flex-1 overflow-y-auto px-4 py-3 space-y-4">
          {/* Severity */}
          <div className="space-y-1">
            <p className="text-xs font-medium text-slate-500">Severity</p>
            <SeverityBadge severity={result.severity} />
          </div>

          {/* Affected resources */}
          <div className="space-y-1">
            <p className="text-xs font-medium text-slate-500">
              Affected Resources ({result.affectedResources.length})
            </p>
            <div className="space-y-1">
              {result.affectedResources.map((id) => (
                <div
                  key={id}
                  className="flex items-center gap-1.5 text-xs text-slate-600 bg-slate-50 rounded-lg px-2.5 py-1.5"
                >
                  <ChevronRight className="w-3 h-3 text-slate-400 flex-shrink-0" />
                  <span className="font-mono truncate">{id}</span>
                </div>
              ))}
              {result.affectedResources.length === 0 && (
                <p className="text-xs text-slate-400 italic">No affected resources detected.</p>
              )}
            </div>
          </div>

          {/* AI explanation */}
          <div className="space-y-1">
            <p className="text-xs font-medium text-slate-500">AI Analysis</p>
            <div className="text-xs text-slate-600 bg-slate-50 rounded-xl px-3 py-3 leading-relaxed border border-slate-100">
              {result.explanation}
            </div>
          </div>

          {/* Propagation paths */}
          {result.propagationPaths.length > 0 && (
            <div className="space-y-1">
              <p className="text-xs font-medium text-slate-500">
                Propagation Paths ({result.propagationPaths.length})
              </p>
              <div className="space-y-1 max-h-40 overflow-y-auto">
                {result.propagationPaths.map((path, i) => (
                  <div key={i} className="flex items-center gap-1 text-[10px] text-slate-500">
                    {path.map((node, j) => (
                      <span key={j} className="flex items-center gap-0.5">
                        <span className="font-mono bg-slate-100 px-1 py-0.5 rounded">
                          {node.split('/').at(-1)}
                        </span>
                        {j < path.length - 1 && (
                          <ChevronRight className="w-2.5 h-2.5 text-slate-300" />
                        )}
                      </span>
                    ))}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Empty state */}
      {!result && !isLoading && (
        <div className="flex-1 flex flex-col items-center justify-center text-center px-4 py-8 space-y-2">
          <AlertTriangle className="w-8 h-8 text-slate-200" />
          <p className="text-xs text-slate-400">
            Select a resource and click Analyze to simulate a failure and see the blast radius.
          </p>
        </div>
      )}
    </div>
  )
}
