import { useState } from 'react'
import type { ReactNode } from 'react'
import type { OrganizationNode } from '../types'

function formatMoney(value: number): string {
  return value.toLocaleString(undefined, { style: 'currency', currency: 'USD', maximumFractionDigits: 0 })
}

export type SelectedScope = { scopeType: string; scopeId: number; label: string } | null

interface Props {
  orgs: OrganizationNode[]
  selected: SelectedScope
  onSelect: (scope: SelectedScope) => void
}

export function HierarchyTree({ orgs, selected, onSelect }: Props) {
  return (
    <div className="card" style={{ padding: 12, overflowY: 'auto' }}>
      {orgs.map((org) => (
        <TreeNode
          key={`org-${org.id}`}
          label={org.name}
          totalCost={org.totalCost}
          depth={0}
          scopeType="org"
          scopeId={org.id}
          selected={selected}
          onSelect={onSelect}
        >
          {org.products.map((product) => (
            <TreeNode
              key={`product-${product.id}`}
              label={product.name}
              totalCost={product.totalCost}
              depth={1}
              scopeType="product"
              scopeId={product.id}
              selected={selected}
              onSelect={onSelect}
            >
              {product.projects.map((project) => (
                <TreeNode
                  key={`project-${project.id}`}
                  label={project.name}
                  totalCost={project.totalCost}
                  depth={2}
                  scopeType="project"
                  scopeId={project.id}
                  selected={selected}
                  onSelect={onSelect}
                >
                  {project.environments.map((env) => (
                    <TreeNode
                      key={`env-${env.id}`}
                      label={env.name}
                      totalCost={env.totalCost}
                      depth={3}
                      scopeType="environment"
                      scopeId={env.id}
                      selected={selected}
                      onSelect={onSelect}
                    >
                      {env.regions.map((region) => (
                        <TreeNode
                          key={`region-${region.id}`}
                          label={region.name}
                          totalCost={region.totalCost}
                          depth={4}
                          scopeType="region"
                          scopeId={region.id}
                          selected={selected}
                          onSelect={onSelect}
                        >
                          {region.cloudAccounts.map((acct) => (
                            <TreeNode
                              key={`acct-${acct.id}`}
                              label={`${acct.name} (${acct.provider})`}
                              totalCost={acct.totalCost}
                              depth={5}
                              scopeType="account"
                              scopeId={acct.id}
                              selected={selected}
                              onSelect={onSelect}
                              badge={!acct.active ? 'inactive' : acct.lastSyncErr ? 'sync error' : undefined}
                            />
                          ))}
                        </TreeNode>
                      ))}
                    </TreeNode>
                  ))}
                </TreeNode>
              ))}
            </TreeNode>
          ))}
        </TreeNode>
      ))}
      {orgs.length === 0 && <div style={{ color: 'var(--text-muted)', fontSize: 14 }}>No organizations yet.</div>}
    </div>
  )
}

interface TreeNodeProps {
  label: string
  totalCost: number
  depth: number
  scopeType: string
  scopeId: number
  selected: SelectedScope
  onSelect: (scope: SelectedScope) => void
  children?: ReactNode
  badge?: string
}

function TreeNode({ label, totalCost, depth, scopeType, scopeId, selected, onSelect, children, badge }: TreeNodeProps) {
  const [open, setOpen] = useState(depth === 0)
  const hasChildren = Boolean(children && (Array.isArray(children) ? children.length > 0 : true))
  const isSelected = selected?.scopeType === scopeType && selected?.scopeId === scopeId

  return (
    <div>
      <div
        onClick={() => onSelect({ scopeType, scopeId, label })}
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 6,
          padding: '5px 6px',
          paddingLeft: 8 + depth * 16,
          borderRadius: 6,
          cursor: 'pointer',
          background: isSelected ? 'var(--gridline)' : 'transparent',
          fontSize: depth === 0 ? 14 : 13,
          fontWeight: depth === 0 ? 600 : 400,
        }}
      >
        {hasChildren ? (
          <span
            onClick={(e) => {
              e.stopPropagation()
              setOpen((o) => !o)
            }}
            style={{ width: 14, color: 'var(--text-muted)' }}
          >
            {open ? '▾' : '▸'}
          </span>
        ) : (
          <span style={{ width: 14 }} />
        )}
        <span style={{ flex: 1 }}>{label}</span>
        {badge && (
          <span style={{ fontSize: 11, color: 'var(--status-critical)', marginRight: 6 }}>{badge}</span>
        )}
        <span style={{ color: 'var(--text-secondary)', fontVariantNumeric: 'tabular-nums' }}>{formatMoney(totalCost)}</span>
      </div>
      {open && hasChildren && <div>{children}</div>}
    </div>
  )
}
