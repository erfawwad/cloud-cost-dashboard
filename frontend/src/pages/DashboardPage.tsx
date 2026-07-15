import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import { useAuth } from '../context/AuthContext'
import { HierarchyTree } from '../components/HierarchyTree'
import type { SelectedScope } from '../components/HierarchyTree'
import { DailyCostChart, ServiceBreakdownChart } from '../components/CostChart'
import type { CostPoint, OrganizationNode } from '../types'

type RangePreset = '7' | '30' | '90'

function toISODate(d: Date): string {
  return d.toISOString().slice(0, 10)
}

function rangeFromPreset(preset: RangePreset): { start: string; end: string } {
  const end = new Date()
  const start = new Date()
  start.setDate(start.getDate() - Number(preset))
  return { start: toISODate(start), end: toISODate(end) }
}

function formatMoney(value: number): string {
  return value.toLocaleString(undefined, { style: 'currency', currency: 'USD', maximumFractionDigits: 0 })
}

export function DashboardPage() {
  const { user, logout } = useAuth()
  const [preset, setPreset] = useState<RangePreset>('30')
  const [orgs, setOrgs] = useState<OrganizationNode[]>([])
  const [selected, setSelected] = useState<SelectedScope>(null)
  const [dailyPoints, setDailyPoints] = useState<CostPoint[]>([])
  const [servicePoints, setServicePoints] = useState<CostPoint[]>([])
  const [loading, setLoading] = useState(true)

  const range = useMemo(() => rangeFromPreset(preset), [preset])

  useEffect(() => {
    setLoading(true)
    api
      .get<OrganizationNode[]>('/api/tree', { params: range })
      .then((res) => {
        setOrgs(res.data)
        if (!selected && res.data.length > 0) {
          setSelected({ scopeType: 'org', scopeId: res.data[0].id, label: res.data[0].name })
        }
      })
      .finally(() => setLoading(false))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [preset])

  useEffect(() => {
    if (!selected) return
    const params = { ...range, scopeType: selected.scopeType, scopeId: selected.scopeId }
    api.get<CostPoint[]>('/api/costs/timeseries', { params: { ...params, groupBy: 'day' } }).then((res) => setDailyPoints(res.data))
    api.get<CostPoint[]>('/api/costs/timeseries', { params: { ...params, groupBy: 'service' } }).then((res) => setServicePoints(res.data))
  }, [selected, range])

  const selectedTotal = servicePoints.reduce((sum, p) => sum + p.amount, 0)

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh' }}>
      <header
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '12px 20px',
          borderBottom: '1px solid var(--border)',
        }}
      >
        <h1 style={{ fontSize: 16, margin: 0 }}>Cloud Cost Dashboard</h1>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <div style={{ display: 'flex', gap: 4 }}>
            {(['7', '30', '90'] as RangePreset[]).map((p) => (
              <button
                key={p}
                onClick={() => setPreset(p)}
                style={{
                  padding: '5px 10px',
                  fontSize: 13,
                  borderRadius: 6,
                  border: '1px solid var(--border)',
                  background: preset === p ? 'var(--series-1)' : 'var(--surface-1)',
                  color: preset === p ? 'white' : 'var(--text-primary)',
                  cursor: 'pointer',
                }}
              >
                {p}d
              </button>
            ))}
          </div>
          <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>
            {user?.email} · {user?.role}
          </span>
          {user?.role === 'admin' && (
            <Link to="/admin" style={{ fontSize: 13, color: 'var(--series-1)' }}>
              Admin
            </Link>
          )}
          <button
            onClick={logout}
            style={{ padding: '5px 10px', fontSize: 13, borderRadius: 6, border: '1px solid var(--border)', background: 'var(--surface-1)', cursor: 'pointer' }}
          >
            Sign out
          </button>
        </div>
      </header>

      <div style={{ flex: 1, display: 'grid', gridTemplateColumns: '320px 1fr', gap: 16, padding: 16, overflow: 'hidden' }}>
        <HierarchyTree orgs={orgs} selected={selected} onSelect={setSelected} />

        <div style={{ overflowY: 'auto', display: 'flex', flexDirection: 'column', gap: 16 }}>
          {loading && <div style={{ color: 'var(--text-muted)' }}>Loading…</div>}

          {selected && (
            <>
              <div className="card" style={{ padding: 20 }}>
                <div style={{ fontSize: 13, color: 'var(--text-secondary)' }}>{selected.label}</div>
                <div style={{ fontSize: 32, fontWeight: 600, marginTop: 4 }}>{formatMoney(selectedTotal)}</div>
                <div style={{ fontSize: 12, color: 'var(--text-muted)', marginTop: 2 }}>
                  {range.start} to {range.end}
                </div>
              </div>

              <div className="card" style={{ padding: 20 }}>
                <div style={{ fontSize: 14, fontWeight: 600, marginBottom: 12 }}>Daily spend</div>
                <DailyCostChart points={dailyPoints} />
              </div>

              <div className="card" style={{ padding: 20 }}>
                <div style={{ fontSize: 14, fontWeight: 600, marginBottom: 12 }}>Top services</div>
                <ServiceBreakdownChart points={servicePoints} />
              </div>
            </>
          )}

          {!selected && !loading && (
            <div style={{ color: 'var(--text-muted)' }}>Select an organization, product, project, environment, region, or cloud account to see its cost.</div>
          )}
        </div>
      </div>
    </div>
  )
}
