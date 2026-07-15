import { useEffect, useState } from 'react'
import type { CSSProperties, FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { OrganizationNode, ProviderCredential, ProviderKey } from '../types'
import { PROVIDERS } from '../types'

const CREDENTIAL_FIELDS: Record<ProviderKey, { key: string; label: string }[]> = {
  aws: [
    { key: 'access_key_id', label: 'Access Key ID' },
    { key: 'secret_access_key', label: 'Secret Access Key' },
  ],
  azure: [
    { key: 'tenant_id', label: 'Tenant ID' },
    { key: 'client_id', label: 'Client ID' },
    { key: 'client_secret', label: 'Client Secret' },
  ],
  gcp: [
    { key: 'service_account_json', label: 'Service Account JSON (paste full key file)' },
    { key: 'bq_project', label: 'BigQuery Project' },
    { key: 'bq_dataset', label: 'BigQuery Dataset' },
    { key: 'bq_table', label: 'BigQuery Table' },
  ],
  oci: [
    { key: 'user_ocid', label: 'User OCID' },
    { key: 'fingerprint', label: 'Fingerprint' },
    { key: 'private_key_pem', label: 'Private Key (PEM)' },
    { key: 'region', label: 'Region' },
  ],
  contabo: [],
  generic: [],
}

function providerLabel(key: ProviderKey): string {
  return PROVIDERS.find((p) => p.key === key)?.label ?? key
}

export function AdminPage() {
  const [orgs, setOrgs] = useState<OrganizationNode[]>([])
  const [credentials, setCredentials] = useState<ProviderCredential[]>([])
  const [message, setMessage] = useState<string | null>(null)

  function refresh() {
    api.get<OrganizationNode[]>('/api/tree').then((res) => setOrgs(res.data))
    api.get<ProviderCredential[]>('/api/credentials').then((res) => setCredentials(res.data))
  }

  useEffect(refresh, [])

  function notify(msg: string) {
    setMessage(msg)
    setTimeout(() => setMessage(null), 4000)
  }

  return (
    <div style={{ maxWidth: 900, margin: '0 auto', padding: 24, display: 'flex', flexDirection: 'column', gap: 24 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <h1 style={{ fontSize: 18, margin: 0 }}>Admin</h1>
        <Link to="/" style={{ fontSize: 13, color: 'var(--series-1)' }}>
          ← Back to dashboard
        </Link>
      </div>

      {message && (
        <div className="card" style={{ padding: 12, fontSize: 13, borderColor: 'var(--series-1)' }}>
          {message}
        </div>
      )}

      <HierarchySection orgs={orgs} onChange={refresh} notify={notify} />
      <CredentialsSection credentials={credentials} onChange={refresh} notify={notify} />
      <CloudAccountsSection orgs={orgs} credentials={credentials} onChange={refresh} notify={notify} />
      <UsersSection orgs={orgs} notify={notify} />
    </div>
  )
}

// ---- hierarchy -------------------------------------------------------------

function HierarchySection({ orgs, onChange, notify }: { orgs: OrganizationNode[]; onChange: () => void; notify: (m: string) => void }) {
  const [orgName, setOrgName] = useState('')
  const [productOrgId, setProductOrgId] = useState('')
  const [productName, setProductName] = useState('')
  const [projectProductId, setProjectProductId] = useState('')
  const [projectName, setProjectName] = useState('')
  const [envProjectId, setEnvProjectId] = useState('')
  const [envName, setEnvName] = useState('')
  const [regionEnvId, setRegionEnvId] = useState('')
  const [regionName, setRegionName] = useState('')

  const products = orgs.flatMap((o) => o.products)
  const projects = products.flatMap((p) => p.projects)
  const environments = projects.flatMap((p) => p.environments)

  async function submit(e: FormEvent, path: string, body: object, reset: () => void) {
    e.preventDefault()
    try {
      await api.post(path, body)
      reset()
      onChange()
      notify('Created.')
    } catch (err: any) {
      notify(err?.response?.data?.error ?? 'Failed to create')
    }
  }

  return (
    <section className="card" style={{ padding: 20 }}>
      <h2 style={{ fontSize: 15, marginTop: 0 }}>Hierarchy: Organization → Product → Project → Environment → Region</h2>
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
        <form
          onSubmit={(e) => submit(e, '/api/organizations', { name: orgName }, () => setOrgName(''))}
          style={{ display: 'flex', gap: 8 }}
        >
          <input placeholder="Organization name" value={orgName} onChange={(e) => setOrgName(e.target.value)} required style={{ flex: 1 }} />
          <SubmitButton />
        </form>

        <form
          onSubmit={(e) =>
            submit(e, '/api/products', { organizationId: Number(productOrgId), name: productName }, () => setProductName(''))
          }
          style={{ display: 'flex', gap: 8 }}
        >
          <select value={productOrgId} onChange={(e) => setProductOrgId(e.target.value)} required>
            <option value="">Organization…</option>
            {orgs.map((o) => (
              <option key={o.id} value={o.id}>
                {o.name}
              </option>
            ))}
          </select>
          <input placeholder="Product name" value={productName} onChange={(e) => setProductName(e.target.value)} required style={{ flex: 1 }} />
          <SubmitButton />
        </form>

        <form
          onSubmit={(e) =>
            submit(e, '/api/projects', { productId: Number(projectProductId), name: projectName }, () => setProjectName(''))
          }
          style={{ display: 'flex', gap: 8 }}
        >
          <select value={projectProductId} onChange={(e) => setProjectProductId(e.target.value)} required>
            <option value="">Product…</option>
            {products.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
          <input placeholder="Project name" value={projectName} onChange={(e) => setProjectName(e.target.value)} required style={{ flex: 1 }} />
          <SubmitButton />
        </form>

        <form
          onSubmit={(e) =>
            submit(e, '/api/environments', { projectId: Number(envProjectId), name: envName }, () => setEnvName(''))
          }
          style={{ display: 'flex', gap: 8 }}
        >
          <select value={envProjectId} onChange={(e) => setEnvProjectId(e.target.value)} required>
            <option value="">Project…</option>
            {projects.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
          <input placeholder="Environment (prod/staging/dev)" value={envName} onChange={(e) => setEnvName(e.target.value)} required style={{ flex: 1 }} />
          <SubmitButton />
        </form>

        <form
          onSubmit={(e) =>
            submit(e, '/api/regions', { environmentId: Number(regionEnvId), name: regionName }, () => setRegionName(''))
          }
          style={{ display: 'flex', gap: 8 }}
        >
          <select value={regionEnvId} onChange={(e) => setRegionEnvId(e.target.value)} required>
            <option value="">Environment…</option>
            {environments.map((e) => (
              <option key={e.id} value={e.id}>
                {e.name}
              </option>
            ))}
          </select>
          <input placeholder="Region (e.g. us-east-1)" value={regionName} onChange={(e) => setRegionName(e.target.value)} required style={{ flex: 1 }} />
          <SubmitButton />
        </form>
      </div>
    </section>
  )
}

// ---- credentials ------------------------------------------------------------

function CredentialsSection({
  credentials,
  onChange,
  notify,
}: {
  credentials: ProviderCredential[]
  onChange: () => void
  notify: (m: string) => void
}) {
  const [provider, setProvider] = useState<ProviderKey>('aws')
  const [name, setName] = useState('')
  const [fields, setFields] = useState<Record<string, string>>({})

  async function submit(e: FormEvent) {
    e.preventDefault()
    try {
      await api.post('/api/credentials', { provider, name, fields })
      setName('')
      setFields({})
      onChange()
      notify('Credential saved (encrypted).')
    } catch (err: any) {
      notify(err?.response?.data?.error ?? 'Failed to save credential')
    }
  }

  async function remove(id: number) {
    try {
      await api.delete(`/api/credentials/${id}`)
      onChange()
    } catch (err: any) {
      notify(err?.response?.data?.error ?? 'Failed to delete credential')
    }
  }

  return (
    <section className="card" style={{ padding: 20 }}>
      <h2 style={{ fontSize: 15, marginTop: 0 }}>Provider credentials</h2>
      <form onSubmit={submit} style={{ display: 'flex', flexDirection: 'column', gap: 8, maxWidth: 480 }}>
        <div style={{ display: 'flex', gap: 8 }}>
          <select
            value={provider}
            onChange={(e) => {
              setProvider(e.target.value as ProviderKey)
              setFields({})
            }}
          >
            {PROVIDERS.map((p) => (
              <option key={p.key} value={p.key}>
                {p.label}
              </option>
            ))}
          </select>
          <input placeholder="Credential name" value={name} onChange={(e) => setName(e.target.value)} required style={{ flex: 1 }} />
        </div>
        {CREDENTIAL_FIELDS[provider].length === 0 ? (
          <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>
            {provider === 'contabo' ? 'Contabo has no cost API — no credential needed, use CSV import.' : 'No live API — costs are added via CSV import.'}
          </div>
        ) : (
          CREDENTIAL_FIELDS[provider].map((f) => (
            <input
              key={f.key}
              placeholder={f.label}
              value={fields[f.key] ?? ''}
              onChange={(e) => setFields((prev) => ({ ...prev, [f.key]: e.target.value }))}
              required
            />
          ))
        )}
        <SubmitButton label="Save credential" />
      </form>

      <table style={{ width: '100%', marginTop: 16, fontSize: 13, borderCollapse: 'collapse' }}>
        <tbody>
          {credentials.map((c) => (
            <tr key={c.id} style={{ borderTop: '1px solid var(--gridline)' }}>
              <td style={{ padding: '6px 0' }}>{c.name}</td>
              <td>{providerLabel(c.provider)}</td>
              <td style={{ textAlign: 'right' }}>
                <button onClick={() => remove(c.id)} style={linkButtonStyle}>
                  Delete
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  )
}

// ---- cloud accounts ---------------------------------------------------------

function CloudAccountsSection({
  orgs,
  credentials,
  onChange,
  notify,
}: {
  orgs: OrganizationNode[]
  credentials: ProviderCredential[]
  onChange: () => void
  notify: (m: string) => void
}) {
  const regions = orgs.flatMap((o) => o.products.flatMap((p) => p.projects.flatMap((pr) => pr.environments.flatMap((e) => e.regions))))
  const accounts = regions.flatMap((r) => r.cloudAccounts.map((a) => ({ ...a, regionName: r.name })))

  const [regionId, setRegionId] = useState('')
  const [provider, setProvider] = useState<ProviderKey>('aws')
  const [name, setName] = useState('')
  const [externalId, setExternalId] = useState('')
  const [credentialId, setCredentialId] = useState('')

  async function submit(e: FormEvent) {
    e.preventDefault()
    try {
      await api.post('/api/cloud-accounts', {
        regionId: Number(regionId),
        provider,
        name,
        externalId,
        credentialId: credentialId ? Number(credentialId) : null,
      })
      setName('')
      setExternalId('')
      onChange()
      notify('Cloud account created.')
    } catch (err: any) {
      notify(err?.response?.data?.error ?? 'Failed to create cloud account')
    }
  }

  async function syncNow(id: number) {
    try {
      await api.post(`/api/cloud-accounts/${id}/sync-now`)
      notify('Sync complete.')
      onChange()
    } catch (err: any) {
      notify(err?.response?.data?.error ?? 'Sync failed')
    }
  }

  async function importCsv(id: number, file: File) {
    const form = new FormData()
    form.append('file', file)
    try {
      const res = await api.post(`/api/cloud-accounts/${id}/import-csv`, form)
      notify(`Imported ${res.data.imported} rows.`)
      onChange()
    } catch (err: any) {
      notify(err?.response?.data?.error ?? 'Import failed')
    }
  }

  return (
    <section className="card" style={{ padding: 20 }}>
      <h2 style={{ fontSize: 15, marginTop: 0 }}>Cloud accounts</h2>
      <form onSubmit={submit} style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginBottom: 16 }}>
        <select value={regionId} onChange={(e) => setRegionId(e.target.value)} required>
          <option value="">Region…</option>
          {regions.map((r) => (
            <option key={r.id} value={r.id}>
              {r.name}
            </option>
          ))}
        </select>
        <select value={provider} onChange={(e) => setProvider(e.target.value as ProviderKey)}>
          {PROVIDERS.map((p) => (
            <option key={p.key} value={p.key}>
              {p.label}
            </option>
          ))}
        </select>
        <input placeholder="Account name" value={name} onChange={(e) => setName(e.target.value)} required />
        <input placeholder="Account/Subscription/Project ID" value={externalId} onChange={(e) => setExternalId(e.target.value)} />
        <select value={credentialId} onChange={(e) => setCredentialId(e.target.value)}>
          <option value="">No credential</option>
          {credentials
            .filter((c) => c.provider === provider)
            .map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
        </select>
        <SubmitButton label="Add account" />
      </form>

      <table style={{ width: '100%', fontSize: 13, borderCollapse: 'collapse' }}>
        <tbody>
          {accounts.map((a) => (
            <tr key={a.id} style={{ borderTop: '1px solid var(--gridline)' }}>
              <td style={{ padding: '6px 0' }}>{a.name}</td>
              <td>{providerLabel(a.provider)}</td>
              <td style={{ color: 'var(--text-muted)' }}>{a.regionName}</td>
              <td style={{ color: a.lastSyncErr ? 'var(--status-critical)' : 'var(--text-muted)' }}>
                {a.lastSyncErr || (a.lastSyncAt ? `synced ${new Date(a.lastSyncAt).toLocaleString()}` : 'never synced')}
              </td>
              <td style={{ textAlign: 'right', whiteSpace: 'nowrap' }}>
                {PROVIDERS.find((p) => p.key === a.provider)?.liveSync ? (
                  <button onClick={() => syncNow(a.id)} style={linkButtonStyle}>
                    Sync now
                  </button>
                ) : (
                  <label style={{ ...linkButtonStyle, cursor: 'pointer' }}>
                    Import CSV
                    <input
                      type="file"
                      accept=".csv"
                      style={{ display: 'none' }}
                      onChange={(e) => {
                        const file = e.target.files?.[0]
                        if (file) importCsv(a.id, file)
                        e.target.value = ''
                      }}
                    />
                  </label>
                )}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <div style={{ fontSize: 12, color: 'var(--text-muted)', marginTop: 8 }}>
        CSV format: header row <code>date,service_name,amount,currency</code>, dates as YYYY-MM-DD.
      </div>
    </section>
  )
}

// ---- users -------------------------------------------------------------

function UsersSection({ orgs, notify }: { orgs: OrganizationNode[]; notify: (m: string) => void }) {
  const products = orgs.flatMap((o) => o.products)
  const projects = products.flatMap((p) => p.projects)

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [role, setRole] = useState<'admin' | 'manager' | 'viewer'>('viewer')
  const [scopeType, setScopeType] = useState<'none' | 'product' | 'project'>('none')
  const [scopeId, setScopeId] = useState('')

  async function submit(e: FormEvent) {
    e.preventDefault()
    try {
      await api.post('/api/users', {
        email,
        password,
        role,
        scopeType: role === 'viewer' ? scopeType : 'none',
        scopeId: role === 'viewer' && scopeType !== 'none' ? Number(scopeId) : null,
      })
      setEmail('')
      setPassword('')
      notify('User created.')
    } catch (err: any) {
      notify(err?.response?.data?.error ?? 'Failed to create user')
    }
  }

  const scopeOptions = scopeType === 'product' ? products : scopeType === 'project' ? projects : []

  return (
    <section className="card" style={{ padding: 20 }}>
      <h2 style={{ fontSize: 15, marginTop: 0 }}>Users</h2>
      <form onSubmit={submit} style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
        <input placeholder="Email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
        <input placeholder="Password" type="password" value={password} onChange={(e) => setPassword(e.target.value)} required />
        <select value={role} onChange={(e) => setRole(e.target.value as typeof role)}>
          <option value="admin">Admin</option>
          <option value="manager">Manager</option>
          <option value="viewer">Viewer</option>
        </select>
        {role === 'viewer' && (
          <>
            <select
              value={scopeType}
              onChange={(e) => {
                setScopeType(e.target.value as typeof scopeType)
                setScopeId('')
              }}
            >
              <option value="none">No scope (sees nothing)</option>
              <option value="product">Scoped to product</option>
              <option value="project">Scoped to project</option>
            </select>
            {scopeType !== 'none' && (
              <select value={scopeId} onChange={(e) => setScopeId(e.target.value)} required>
                <option value="">Choose…</option>
                {scopeOptions.map((o) => (
                  <option key={o.id} value={o.id}>
                    {o.name}
                  </option>
                ))}
              </select>
            )}
          </>
        )}
        <SubmitButton label="Create user" />
      </form>
    </section>
  )
}

// ---- shared bits -------------------------------------------------------

function SubmitButton({ label = 'Add' }: { label?: string }) {
  return (
    <button
      type="submit"
      style={{ padding: '6px 12px', background: 'var(--series-1)', color: 'white', border: 'none', borderRadius: 6, cursor: 'pointer', fontSize: 13 }}
    >
      {label}
    </button>
  )
}

const linkButtonStyle: CSSProperties = {
  background: 'none',
  border: 'none',
  color: 'var(--series-1)',
  cursor: 'pointer',
  fontSize: 13,
  padding: 0,
}
