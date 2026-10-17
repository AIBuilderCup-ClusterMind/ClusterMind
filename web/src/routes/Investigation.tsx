import { useMemo } from 'react'
import { ArrowLeft, Siren } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { Button, PageHeader, Panel, StateMessage } from '../components/cm/primitives'
import { IncidentTimeline, formatEventTime } from '../components/IncidentTimeline'
import { InvestigationPanel } from '../components/InvestigationPanel'
import { SeverityBadge } from '../components/SeverityBadge'
import { useIncident } from '../hooks/useIncident'
import { useTwin } from '../hooks/useTwin'
import type { IncidentSeverity } from '../types'

const sevBadge: Record<IncidentSeverity, 'critical' | 'high' | 'medium'> = {
  sev1: 'critical',
  sev2: 'high',
  sev3: 'medium',
}

/**
 * Investigation view: focused, full-height report for one incident.
 * The investigation runs automatically when an incident is selected.
 */
export function Investigation() {
  const { data: twin } = useTwin()
  const { selectedIncidentId } = useIncident()
  const navigate = useNavigate()

  const incident = useMemo(
    () => twin?.incidents.find((i) => i.id === selectedIncidentId) ?? null,
    [twin, selectedIncidentId],
  )

  if (!incident) {
    return (
      <div className="flex h-full flex-col">
        <PageHeader title="Investigation" subtitle="AI-assisted incident report." />
        <div className="flex min-h-0 flex-1 items-center justify-center px-5 pb-5">
          <StateMessage
            kind="empty"
            title="No incident selected"
            detail="Pick an open incident first — the investigation runs against its evidence timeline and blast radius."
            action={
              <Button variant="secondary" onClick={() => navigate('/incidents')}>
                <Siren className="size-3.5" />
                Browse incidents
              </Button>
            }
          />
        </div>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        title="Investigation"
        subtitle={incident.title}
        actions={
          <Button variant="ghost" onClick={() => navigate('/incidents')}>
            <ArrowLeft className="size-3.5" />
            Back to incidents
          </Button>
        }
      />
      <div className="grid min-h-[480px] flex-1 gap-3 px-5 pb-5 lg:grid-cols-[320px_1fr]">
        {/* Left: incident meta + timeline */}
        <div className="flex min-h-0 flex-col gap-3">
          <Panel title="Incident">
            <div className="space-y-3 p-3">
              <div className="flex items-center gap-2">
                <SeverityBadge severity={sevBadge[incident.severity]} />
                <span className="label-xs rounded-sm border bg-surface-2 px-1.5 py-0.5 font-mono">
                  {incident.status}
                </span>
              </div>
              <div className="space-y-1.5">
                <Row label="ID" value={incident.id} mono />
                <Row label="Resource" value={incident.primaryResourceId} mono />
                <Row label="Started" value={formatEventTime(incident.startedAt)} />
                <Row label="Evidence" value={`${incident.timeline.length} entries`} />
              </div>
            </div>
          </Panel>

          <Panel title="Evidence timeline" className="min-h-0 flex-1" bodyClassName="overflow-hidden">
            <IncidentTimeline incident={incident} />
          </Panel>
        </div>

        {/* Right: investigation report (auto-runs) */}
        <Panel
          title="Report"
          className="min-h-[480px]"
          bodyClassName="flex flex-col overflow-hidden"
        >
          <InvestigationPanel incident={incident} autoRun />
        </Panel>
      </div>
    </div>
  )
}

function Row({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex items-baseline justify-between gap-3">
      <span className="label-xs shrink-0">{label}</span>
      <span
        className={
          mono
            ? 'min-w-0 truncate text-right font-mono text-[11px]'
            : 'min-w-0 truncate text-right text-[11px]'
        }
        title={value}
      >
        {value}
      </span>
    </div>
  )
}
