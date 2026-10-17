import { useEffect, useMemo } from 'react'
import { ArrowRight, Siren } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { cn } from '../lib/utils'
import { Button, PageHeader, Panel, StateMessage } from '../components/cm/primitives'
import { IncidentTimeline, formatEventTime } from '../components/IncidentTimeline'
import { InvestigationPanel } from '../components/InvestigationPanel'
import { TopologyGraph } from '../components/TopologyGraph'
import { ImpactPanel } from '../components/ImpactPanel'
import { SeverityBadge } from '../components/SeverityBadge'
import { useImpact } from '../hooks/useImpact'
import { useIncident } from '../hooks/useIncident'
import { useTwin } from '../hooks/useTwin'
import type { IncidentSeverity } from '../types'

const sevBadge: Record<IncidentSeverity, 'critical' | 'high' | 'medium'> = {
  sev1: 'critical',
  sev2: 'high',
  sev3: 'medium',
}

/**
 * Incidents view: incident list + evidence timeline, topology with an
 * automatic blast-radius overlay for the selected incident, impact panel,
 * and a full-width investigation panel underneath.
 */
export function Incidents() {
  const { data: twin } = useTwin()
  const { selectedIncidentId, select } = useIncident()
  const { target, setTarget, analyze } = useImpact()
  const navigate = useNavigate()

  const incidents = useMemo(() => twin?.incidents ?? [], [twin])
  const selected = useMemo(
    () => incidents.find((i) => i.id === selectedIncidentId) ?? null,
    [incidents, selectedIncidentId],
  )

  // Auto-run impact analysis whenever the selection changes.
  useEffect(() => {
    if (selected && target !== selected.primaryResourceId) {
      setTarget(selected.primaryResourceId)
      void analyze()
    }
  }, [selected, target, setTarget, analyze])

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        title="Incidents"
        subtitle="Open incidents derived from Kubernetes events and resource health."
      />
      <div className="grid min-h-[480px] flex-1 gap-3 px-5 lg:grid-cols-[300px_1fr_320px]">
        {/* Left: incident list + timeline */}
        <div className="flex min-h-0 flex-col gap-3">
          <Panel
            title={`Open incidents · ${incidents.filter((i) => i.status === 'open').length}`}
            className="max-h-56 shrink-0"
            bodyClassName="overflow-y-auto"
          >
            {incidents.length === 0 ? (
              <StateMessage
                kind="empty"
                title="No open incidents"
                detail="No unhealthy or degraded resources right now — the cluster looks quiet."
              />
            ) : (
              <ul className="divide-y">
                {incidents.map((inc) => (
                  <li key={inc.id}>
                    <button
                      onClick={() => select(inc.id)}
                      className={cn(
                        'flex w-full flex-col gap-1 px-3 py-2 text-left transition-colors hover:bg-accent',
                        selectedIncidentId === inc.id && 'bg-accent',
                      )}
                    >
                      <div className="flex items-center gap-2">
                        <SeverityBadge severity={sevBadge[inc.severity]} />
                        <span className="truncate text-xs font-medium">{inc.title}</span>
                      </div>
                      <div className="flex items-center justify-between gap-2 font-mono text-[10px] text-muted-foreground">
                        <span className="truncate">{inc.primaryResourceId}</span>
                        <time dateTime={inc.startedAt}>{formatEventTime(inc.startedAt)}</time>
                      </div>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </Panel>

          <Panel
            title="Evidence timeline"
            className="min-h-0 flex-1"
            bodyClassName="overflow-hidden"
          >
            <IncidentTimeline incident={selected} />
          </Panel>
        </div>

        {/* Center: topology with auto impact overlay */}
        <Panel className="relative min-h-[420px] overflow-hidden">
          <div className="absolute inset-0">
            <TopologyGraph />
          </div>
          <div className="pointer-events-none absolute right-3 top-3 rounded-md border bg-background/90 px-2.5 py-1.5 font-mono text-[10px] text-muted-foreground">
            {selected ? `impact · ${selected.primaryResourceId}` : 'select an incident to overlay impact'}
          </div>
        </Panel>

        {/* Right: impact panel */}
        <Panel className="min-h-[420px]" bodyClassName="overflow-hidden">
          <ImpactPanel />
        </Panel>
      </div>

      {/* Bottom: investigation */}
      <div className="mt-3 px-5 pb-5">
        <Panel
          title="Investigation"
          className="min-h-[280px]"
          actions={
            selected ? (
              <Button
                variant="ghost"
                onClick={() => navigate('/investigation')}
                title="Open full investigation view"
              >
                <Siren className="size-3.5" />
                Full view
                <ArrowRight className="size-3" />
              </Button>
            ) : null
          }
          bodyClassName="flex flex-col overflow-hidden"
        >
          <InvestigationPanel incident={selected} />
        </Panel>
      </div>
    </div>
  )
}
