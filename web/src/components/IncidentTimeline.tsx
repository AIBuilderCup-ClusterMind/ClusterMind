import { CircleDot, Radio, Siren, TriangleAlert } from 'lucide-react'
import { cn } from '../lib/utils'
import { StateMessage } from './cm/primitives'
import type { TwinIncident } from '../types'

const sourceMeta = {
  events: { label: 'event', icon: Radio, className: 'text-warning border-warning/30 bg-warning/10' },
  deployment: { label: 'rollout', icon: TriangleAlert, className: 'text-primary border-primary/30 bg-primary/10' },
  alert: { label: 'alert', icon: Siren, className: 'text-critical border-critical/40 bg-critical/10' },
} as const

export function formatEventTime(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString(undefined, {
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  })
}

/**
 * Vertical timeline of evidence entries for one incident.
 * Each entry maps 1:1 to a backend TwinEvidence row.
 */
export function IncidentTimeline({ incident }: { incident: TwinIncident | null }) {
  if (!incident) {
    return (
      <StateMessage
        kind="empty"
        title="No incident selected"
        detail="Pick an incident to inspect its evidence timeline."
      />
    )
  }

  if (incident.timeline.length === 0) {
    return (
      <StateMessage kind="empty" title="No evidence recorded" detail="This incident has no timeline entries yet." />
    )
  }

  return (
    <ol className="relative space-y-0 overflow-y-auto p-3">
      <span className="absolute bottom-4 left-[22px] top-4 w-px bg-border" aria-hidden />
      {incident.timeline.map((ev) => {
        const meta = sourceMeta[ev.source] ?? sourceMeta.events
        const Icon = meta.icon
        return (
          <li key={ev.id} className="relative flex gap-3 pb-4 pl-0 last:pb-0">
            <span
              className={cn(
                'relative z-10 mt-0.5 grid size-[18px] shrink-0 place-items-center rounded-full border bg-surface',
                meta.className,
              )}
            >
              <Icon className="size-2.5" />
            </span>
            <div className="min-w-0 flex-1">
              <div className="flex items-baseline justify-between gap-2">
                <p className="truncate text-xs font-medium" title={ev.title}>
                  {ev.title}
                </p>
                <time
                  className="shrink-0 font-mono text-[10px] text-muted-foreground"
                  dateTime={ev.time}
                  title={ev.time}
                >
                  {formatEventTime(ev.time)}
                </time>
              </div>
              <p className="label-xs mt-0.5 flex items-center gap-1.5 !text-[9px]">
                <CircleDot className="size-2.5" />
                {meta.label}
                <span className="text-muted-foreground/60">·</span>
                <span className="truncate font-mono" title={ev.resourceId}>
                  {ev.resourceId}
                </span>
              </p>
              {ev.detail && (
                <p className="mt-1 break-words rounded-md border border-border/60 bg-background px-2 py-1.5 font-mono text-[10px] leading-relaxed text-muted-foreground">
                  {ev.detail}
                </p>
              )}
            </div>
          </li>
        )
      })}
    </ol>
  )
}
