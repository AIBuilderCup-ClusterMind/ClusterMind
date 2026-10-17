import { ChevronRight, Target, X } from 'lucide-react'
import { Button, Panel, StateMessage } from './cm/primitives'
import { useImpact } from '../hooks/useImpact'
import { useTwin } from '../hooks/useTwin'
import { SeverityBadge } from './SeverityBadge'

export function ImpactPanel() {
  const { data: snapshot } = useTwin()
  const { target, result, isLoading, error, setTarget, analyze, clear } = useImpact()
  const nodes = snapshot?.nodes ?? []

  return (
    <Panel
      title="Impact analysis"
      className="h-full rounded-none border-0"
      actions={
        result ? (
          <button
            onClick={clear}
            title="Clear result"
            className="rounded-sm p-1 text-muted-foreground hover:bg-accent hover:text-foreground"
          >
            <X className="size-3.5" />
          </button>
        ) : null
      }
      bodyClassName="flex flex-col min-h-0"
    >
      {/* Target selector */}
      <div className="space-y-2 border-b p-3">
        <label className="label-xs">Target resource</label>
        <select
          value={target ?? ''}
          onChange={(e) => setTarget(e.target.value)}
          className="w-full rounded-md border border-border-strong bg-surface-2 px-3 py-2 font-mono text-xs text-foreground focus:outline-none"
        >
          <option value="">Select resource…</option>
          {nodes.map((n) => (
            <option key={n.id} value={n.id}>
              [{n.kind}] {n.name}
            </option>
          ))}
        </select>

        <Button
          variant="primary"
          onClick={() => void analyze()}
          disabled={!target}
          loading={isLoading}
          className="w-full justify-center"
        >
          <Target className="size-3.5" />
          {isLoading ? 'Analyzing…' : 'Analyze'}
        </Button>

        {error && (
          <p className="rounded-md border border-critical/40 bg-critical/10 px-3 py-2 text-xs text-critical">
            {error}
          </p>
        )}
      </div>

      {/* Result */}
      {isLoading && <StateMessage kind="loading" title="Traversing dependency graph…" />}

      {!isLoading && !result && (
        <StateMessage
          kind="empty"
          title="Blast radius not computed"
          detail="Select a resource and run analysis to find affected components."
        />
      )}

      {!isLoading && result && (
        <div className="min-h-0 flex-1 space-y-4 overflow-y-auto p-3">
          <div>
            <p className="label-xs mb-1.5">Severity</p>
            <SeverityBadge severity={result.severity} />
          </div>

          <div>
            <p className="label-xs mb-1.5">Affected resources · {result.affectedResources.length}</p>
            <div className="space-y-1">
              {result.affectedResources.map((id) => (
                <div
                  key={id}
                  className="flex items-center gap-1.5 rounded-md border border-border/60 bg-background px-2.5 py-1.5 text-xs"
                >
                  <ChevronRight className="size-3 shrink-0 text-muted-foreground" />
                  <span className="truncate font-mono">{id}</span>
                </div>
              ))}
              {result.affectedResources.length === 0 && (
                <p className="text-xs italic text-muted-foreground">No affected resources detected.</p>
              )}
            </div>
          </div>

          {result.explanation && (
            <div>
              <p className="label-xs mb-1.5">AI analysis</p>
              <p className="rounded-md border border-border/60 bg-background px-3 py-3 text-xs leading-relaxed text-foreground">
                {result.explanation}
              </p>
            </div>
          )}

          {result.propagationPaths.length > 0 && (
            <div>
              <p className="label-xs mb-1.5">Propagation paths · {result.propagationPaths.length}</p>
              <ul className="max-h-40 space-y-1 overflow-y-auto">
                {result.propagationPaths.map((path, i) => (
                  <li key={i} className="flex items-center gap-1 font-mono text-[11px] text-muted-foreground">
                    {path.map((node, j) => (
                      <span key={j} className="flex items-center gap-0.5">
                        <span className="rounded-sm bg-muted px-1 py-0.5">{node.split('/').at(-1)}</span>
                        {j < path.length - 1 && <ChevronRight className="size-2.5" />}
                      </span>
                    ))}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </Panel>
  )
}
