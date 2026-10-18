import { NavLink, useNavigate } from 'react-router-dom'
import { Activity, Beaker, ExternalLink, LayoutGrid, Network, RefreshCw, Siren } from 'lucide-react'
import type { ReactNode } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { cn } from '../../lib/utils'
import { useTwin } from '../../hooks/useTwin'

const INTRO_URL = 'https://clustermind-intro-607541401069.asia-northeast3.run.app'

const nav = [
  { to: '/', label: 'Overview', icon: LayoutGrid },
  { to: '/topology', label: 'Topology', icon: Network },
  { to: '/incidents', label: 'Incidents', icon: Siren },
  { to: '/investigation', label: 'Investigation', icon: Beaker },
] as const

export function AppShell({ children }: { children: ReactNode }) {
  const { data: twin, isLoading } = useTwin()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const [refreshing, setRefreshing] = useState(false)

  const openIncidents = twin?.incidents.filter((i) => i.status === 'open').length ?? 0

  const refresh = () => {
    setRefreshing(true)
    void queryClient.invalidateQueries().finally(() => setRefreshing(false))
  }

  return (
    <div className="flex h-screen min-h-0 overflow-hidden">
      <aside className="flex w-14 shrink-0 flex-col border-r bg-surface md:w-52">
        <div className="flex h-12 items-center gap-2 border-b px-3">
          <div className="grid size-7 place-items-center rounded-sm bg-primary/15 text-primary">
            <Activity className="size-4" />
          </div>
          <div className="hidden leading-tight md:block">
            <p className="text-sm font-semibold tracking-tight">ClusterMind</p>
            <p className="font-mono text-[10px] text-muted-foreground">digital twin</p>
          </div>
        </div>
        <nav aria-label="Primary" className="flex flex-1 flex-col gap-0.5 p-2">
          {nav.map(({ to, label, icon: Icon }) => (
            <NavLink
              key={to}
              to={to}
              title={label}
              end={to === '/'}
              className={({ isActive }) =>
                cn(
                  'flex h-8 items-center gap-2.5 rounded-md px-2 text-[13px] text-muted-foreground transition-colors hover:bg-accent hover:text-foreground',
                  isActive && 'bg-accent text-foreground',
                )
              }
            >
              <Icon className="size-4 shrink-0" />
              <span className="hidden flex-1 md:inline">{label}</span>
              {to === '/incidents' && openIncidents > 0 && (
                <span
                  title={`${openIncidents} open incident(s)`}
                  className="grid min-w-4 shrink-0 place-items-center rounded-full bg-critical px-1 py-0.5 font-mono text-[9px] font-semibold text-white"
                >
                  {openIncidents}
                </span>
              )}
            </NavLink>
          ))}
        </nav>
        <button
          onClick={() => navigate('/topology')}
          className="hidden border-t p-3 text-left font-mono text-[10px] leading-relaxed text-muted-foreground hover:text-foreground md:block"
        >
          Live cluster · read-only probe
        </button>
      </aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-12 shrink-0 items-center gap-4 border-b px-4">
          <Ctx label="cluster" value={twin?.cluster ?? '…'} />
          <Ctx label="namespace" value={twin?.namespace ?? '…'} />
          <div className="ml-auto flex items-center gap-3">
            <a
              href={(import.meta.env.VITE_INTRO_URL as string | undefined) || INTRO_URL}
              target="_blank"
              rel="noopener noreferrer"
              title="Intro page"
              aria-label="Open intro page"
              className="grid size-8 place-items-center rounded-md border border-border-strong text-muted-foreground hover:bg-accent hover:text-foreground"
            >
              <ExternalLink className="size-3.5" />
            </a>
            <button
              onClick={refresh}
              title="Refresh snapshot"
              aria-label="Refresh snapshot"
              className="grid size-8 place-items-center rounded-md border border-border-strong text-muted-foreground hover:bg-accent hover:text-foreground"
            >
              <RefreshCw
                className={cn('size-3.5', (isLoading || refreshing) && 'animate-spin')}
              />
            </button>
          </div>
        </header>
        <main className="min-h-0 flex-1 overflow-auto">{children}</main>
      </div>
    </div>
  )
}

function Ctx({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center gap-2 text-[13px]">
      <span className="label-xs">{label}</span>
      <span className="rounded-sm border bg-surface-2 px-2 py-0.5 font-mono text-xs">{value}</span>
    </div>
  )
}
