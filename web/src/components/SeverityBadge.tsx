import { clsx } from 'clsx'

type Severity = 'critical' | 'high' | 'medium' | 'low'

const styles: Record<Severity, string> = {
  critical: 'bg-red-100 text-red-700 border-red-300',
  high:     'bg-orange-100 text-orange-700 border-orange-300',
  medium:   'bg-yellow-100 text-yellow-700 border-yellow-300',
  low:      'bg-green-100 text-green-700 border-green-300',
}

const labels: Record<Severity, string> = {
  critical: '🔴 CRITICAL',
  high:     '🟠 HIGH',
  medium:   '🟡 MEDIUM',
  low:      '🟢 LOW',
}

interface SeverityBadgeProps {
  severity: Severity
  className?: string
}

export function SeverityBadge({ severity, className }: SeverityBadgeProps) {
  return (
    <span
      className={clsx(
        'inline-flex items-center px-3 py-1 rounded-full text-xs font-semibold border',
        styles[severity],
        className,
      )}
    >
      {labels[severity]}
    </span>
  )
}
