export type Role = 'admin' | 'manager' | 'viewer'
export type ScopeType = 'none' | 'product' | 'project'
export type ProviderKey = 'aws' | 'azure' | 'gcp' | 'oci' | 'contabo' | 'generic'

export interface User {
  id: number
  email: string
  role: Role
  scopeType: ScopeType
  scopeId?: number
}

export interface CloudAccountNode {
  id: number
  name: string
  provider: ProviderKey
  externalId: string
  active: boolean
  lastSyncAt?: string
  lastSyncErr?: string
  totalCost: number
}

export interface RegionNode {
  id: number
  name: string
  totalCost: number
  cloudAccounts: CloudAccountNode[]
}

export interface EnvironmentNode {
  id: number
  name: string
  totalCost: number
  regions: RegionNode[]
}

export interface ProjectNode {
  id: number
  name: string
  totalCost: number
  environments: EnvironmentNode[]
}

export interface ProductNode {
  id: number
  name: string
  totalCost: number
  projects: ProjectNode[]
}

export interface OrganizationNode {
  id: number
  name: string
  totalCost: number
  products: ProductNode[]
}

export interface CostPoint {
  label: string
  amount: number
}

export interface ProviderCredential {
  id: number
  provider: ProviderKey
  name: string
  createdAt: string
}

export const PROVIDERS: { key: ProviderKey; label: string; liveSync: boolean }[] = [
  { key: 'aws', label: 'AWS', liveSync: true },
  { key: 'azure', label: 'Azure', liveSync: true },
  { key: 'gcp', label: 'GCP', liveSync: true },
  { key: 'oci', label: 'OCI', liveSync: true },
  { key: 'contabo', label: 'Contabo', liveSync: false },
  { key: 'generic', label: 'Other / Custom', liveSync: false },
]
