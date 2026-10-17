import type { ButtonHTMLAttributes, ReactNode } from 'react'
import {
  AlertTriangle,
  Box,
  Boxes,
  CloudOff,
  Database,
  FileCode,
  Globe,
  Inbox,
  Layers,
  Loader2,
  Network,
  ScanLine,
  Server,
  Shield,
  Zap,
} from 'lucide-react'
import { cn } from '../../lib/utils'
import type { HealthStatus } from '../../types'

// ── Kind → icon ─────────────────────────────────────────────────────────────

const baseKindIcon: Record<string, typeof Globe> = {
  ingress: Globe,
  service: Network,
  deployment: Layers,
  statefulset: Server,
  daemonset: Shield,
  pod: Box,
  hpa: ScanLine,
  horizontalpodautoscaler: ScanLine,
  configmap: FileCode,
  secret: FileCode,
  networkpolicy: Shield,
  namespace: Boxes,
  database: Database,
  cache: Zap,
}

const fallbackIcon = Box

export function kindIconFor(kind: string): typeof Globe {
  return baseKindIcon[kind.toLowerCase()] ?? fallbackIcon
}

// ── Health status ───────────────────────────────────────────────────────────

const statusStyle: Record<HealthStatus, string> = {
  healthy: 'text-healthy border-healthy/30 bg-healthy/10',
  degraded: 'text-warning border-warning/30 bg-warning/10',
  unhealthy: 'text-critical border-critical/40 bg-critical/10',
  unknown: 'text-unknown border-unknown/30 bg-unknown/10',
}

export const statusDot: Record<HealthStatus, string> = {
  healthy: 'bg-healthy',
  degraded: 'bg-warning',
  unhealthy: 'bg-critical',
  unknown: 'bg-unknown',
}

export function StatusBadge({ status }: { status: HealthStatus }) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 rounded-sm border px-1.5 py-0.5 font-mono text-[10px] uppercase tracking-wider',
        statusStyle[status],
      )}
    >
      <span className={cn('size-1.5 rounded-full', statusDot[status])} />
      {status}
    </span>
  )
}

// ── Button ──────────────────────────────────────────────────────────────────

type Variant = 'primary' | 'secondary' | 'ghost'

const variants: Record<Variant, string> = {
  primary: 'bg-primary text-primary-foreground hover:bg-primary/90',
  secondary: 'border border-border-strong bg-surface-2 text-foreground hover:bg-accent',
  ghost: 'text-muted-foreground hover:bg-accent hover:text-foreground',
}

export function Button({
  variant = 'secondary',
  loading,
  children,
  className,
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: Variant; loading?: boolean }) {
  return (
    <button
      {...props}
      disabled={props.disabled || loading}
      className={cn(
        'inline-flex h-8 items-center gap-2 rounded-md px-3 text-[13px] font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50',
        variants[variant],
        className,
      )}
    >
      {loading && <Loader2 className="size-3.5 animate-spin" />}
      {children}
    </button>
  )
}

// ── Panel ───────────────────────────────────────────────────────────────────

export function Panel({
  title,
  actions,
  children,
  className,
  bodyClassName,
}: {
  title?: ReactNode
  actions?: ReactNode
  children: ReactNode
  className?: string
  bodyClassName?: string
}) {
  return (
    <section className={cn('flex min-h-0 flex-col rounded-md border bg-surface', className)}>
      {title && (
        <header className="flex h-10 shrink-0 items-center justify-between gap-2 border-b px-3">
          <h2 className="label-xs">{title}</h2>
          {actions}
        </header>
      )}
      <div className={cn('min-h-0 flex-1', bodyClassName)}>{children}</div>
    </section>
  )
}

// ── State message (loading / empty / error) ─────────────────────────────────

export function StateMessage({
  kind,
  title,
  detail,
  action,
}: {
  kind: 'loading' | 'empty' | 'error' | 'unavailable'
  title: string
  detail?: string
  action?: ReactNode
}) {
  const Icon = { loading: Loader2, empty: Inbox, error: AlertTriangle, unavailable: CloudOff }[
    kind
  ]
  return (
    <div
      role={kind === 'error' ? 'alert' : 'status'}
      className="flex h-full min-h-32 flex-col items-center justify-center gap-2 p-6 text-center"
    >
      <Icon
        className={cn(
          'size-5',
          kind === 'loading' && 'animate-spin text-primary',
          kind === 'error' ? 'text-critical' : 'text-muted-foreground',
        )}
      />
      <p className="text-sm font-medium">{title}</p>
      {detail && <p className="max-w-xs text-xs text-muted-foreground">{detail}</p>}
      {action}
    </div>
  )
}

// ── Page header ─────────────────────────────────────────────────────────────

export function PageHeader({
  title,
  subtitle,
  actions,
}: {
  title: string
  subtitle?: string
  actions?: ReactNode
}) {
  return (
    <div className="flex flex-wrap items-end justify-between gap-3 px-5 pb-4 pt-5">
      <div>
        <h1 className="text-lg font-semibold tracking-tight">{title}</h1>
        {subtitle && <p className="text-[13px] text-muted-foreground">{subtitle}</p>}
      </div>
      {actions && <div className="flex gap-2">{actions}</div>}
    </div>
  )
}
