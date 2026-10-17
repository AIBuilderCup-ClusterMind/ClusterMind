import { cn } from '../lib/utils'

type Severity = 'critical' | 'high' | 'medium' | 'low'

const styles: Record<Severity, string> = {
  critical: 'bg-critical/10 text-critical border-critical/40',
  high: 'bg-warning/10 text-warning border-warning/40',
  medium: 'bg-warning/10 text-warning border-warning/30',
  low: 'bg-healthy/10 text-healthy border-healthy/30',
}

interface SeverityBadgeProps {
  severity: Severity
  className?: string
}

export function SeverityBadge({ severity, className }: SeverityBadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center rounded-sm border px-2 py-0.5 font-mono text-[10px] font-semibold uppercase tracking-wider',
        styles[severity],
        className,
      )}
    >
      {severity}
    </span>
  )
}
