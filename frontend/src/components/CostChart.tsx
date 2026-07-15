import { Bar, BarChart, CartesianGrid, Cell, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts'
import type { CostPoint } from '../types'

const SERIES_COLORS = [
  'var(--series-1)',
  'var(--series-2)',
  'var(--series-3)',
  'var(--series-4)',
  'var(--series-5)',
  'var(--series-6)',
  'var(--series-7)',
  'var(--series-8)',
]

function formatMoney(value: number): string {
  return value.toLocaleString(undefined, { style: 'currency', currency: 'USD', maximumFractionDigits: 0 })
}

interface ChartTooltipProps {
  active?: boolean
  payload?: readonly { value?: unknown }[]
  label?: string
}

function ChartTooltip({ active, payload, label }: ChartTooltipProps) {
  if (!active || !payload?.length) return null
  return (
    <div
      style={{
        background: 'var(--surface-1)',
        border: '1px solid var(--border)',
        borderRadius: 6,
        padding: '8px 12px',
        fontSize: 13,
        color: 'var(--text-primary)',
      }}
    >
      <div style={{ color: 'var(--text-secondary)', marginBottom: 4 }}>{label}</div>
      <div style={{ fontWeight: 600 }}>{formatMoney(Number(payload[0].value))}</div>
    </div>
  )
}

// Single magnitude over time (e.g. daily total spend) — one hue, sequential job.
export function DailyCostChart({ points }: { points: CostPoint[] }) {
  if (points.length === 0) {
    return <EmptyState message="No cost data yet for this range." />
  }
  return (
    <ResponsiveContainer width="100%" height={260}>
      <BarChart data={points} margin={{ top: 8, right: 8, bottom: 0, left: 0 }}>
        <CartesianGrid stroke="var(--gridline)" vertical={false} />
        <XAxis dataKey="label" tick={{ fill: 'var(--text-muted)', fontSize: 12 }} axisLine={{ stroke: 'var(--baseline)' }} tickLine={false} />
        <YAxis tick={{ fill: 'var(--text-muted)', fontSize: 12 }} axisLine={false} tickLine={false} tickFormatter={formatMoney} width={70} />
        <Tooltip content={(props) => <ChartTooltip {...(props as ChartTooltipProps)} />} cursor={{ fill: 'var(--gridline)' }} />
        <Bar dataKey="amount" fill="var(--series-1)" radius={[4, 4, 0, 0]} maxBarSize={28} />
      </BarChart>
    </ResponsiveContainer>
  )
}

// Cost broken down by service — identity/categorical job, fixed hue order, direct labels via legend below.
export function ServiceBreakdownChart({ points }: { points: CostPoint[] }) {
  if (points.length === 0) {
    return <EmptyState message="No service breakdown for this range." />
  }
  const top = points.slice(0, 8)
  const other = points.slice(8).reduce((sum, p) => sum + p.amount, 0)
  const bars = other > 0 ? [...top, { label: 'Other', amount: other }] : top

  return (
    <div>
      <ResponsiveContainer width="100%" height={260}>
        <BarChart data={bars} layout="vertical" margin={{ top: 8, right: 16, bottom: 0, left: 0 }}>
          <CartesianGrid stroke="var(--gridline)" horizontal={false} />
          <XAxis type="number" tick={{ fill: 'var(--text-muted)', fontSize: 12 }} axisLine={false} tickLine={false} tickFormatter={formatMoney} />
          <YAxis
            type="category"
            dataKey="label"
            tick={{ fill: 'var(--text-primary)', fontSize: 12 }}
            axisLine={{ stroke: 'var(--baseline)' }}
            tickLine={false}
            width={140}
          />
          <Tooltip content={(props) => <ChartTooltip {...(props as ChartTooltipProps)} />} cursor={{ fill: 'var(--gridline)' }} />
          <Bar dataKey="amount" radius={[0, 4, 4, 0]} maxBarSize={22}>
            {bars.map((_, i) => (
              <Cell key={i} fill={SERIES_COLORS[i % SERIES_COLORS.length]} />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  )
}

function EmptyState({ message }: { message: string }) {
  return (
    <div style={{ height: 260, display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--text-muted)', fontSize: 14 }}>
      {message}
    </div>
  )
}

export const seriesColors = SERIES_COLORS
