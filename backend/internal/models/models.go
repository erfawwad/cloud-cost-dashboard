package models

import "time"

// Role is the access level granted to a User.
type Role string

const (
	RoleAdmin   Role = "admin"   // manage orgs, providers, credentials, users
	RoleManager Role = "manager" // read-only across the whole organization
	RoleViewer  Role = "viewer"  // read-only, scoped to one Product or Project
)

// ScopeType restricts a Viewer's visibility to a subtree of the hierarchy.
type ScopeType string

const (
	ScopeNone    ScopeType = "none" // admins/managers: no restriction
	ScopeProduct ScopeType = "product"
	ScopeProject ScopeType = "project"
)

type User struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Email        string `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string `gorm:"not null" json:"-"`
	Role         Role   `gorm:"not null" json:"role"`
	ScopeType    ScopeType `gorm:"not null;default:none" json:"scopeType"`
	ScopeID      *uint     `json:"scopeId,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

// Organization is the top of the cost hierarchy.
type Organization struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"uniqueIndex;not null" json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	Products  []Product `json:"products,omitempty"`
}

type Product struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	OrganizationID uint      `gorm:"not null;index" json:"organizationId"`
	Name           string    `gorm:"not null" json:"name"`
	CreatedAt      time.Time `json:"createdAt"`
	Projects       []Project `json:"projects,omitempty"`
}

type Project struct {
	ID           uint          `gorm:"primaryKey" json:"id"`
	ProductID    uint          `gorm:"not null;index" json:"productId"`
	Name         string        `gorm:"not null" json:"name"`
	CreatedAt    time.Time     `json:"createdAt"`
	Environments []Environment `json:"environments,omitempty"`
}

type Environment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ProjectID uint      `gorm:"not null;index" json:"projectId"`
	Name      string    `gorm:"not null" json:"name"` // prod / staging / dev / ...
	CreatedAt time.Time `json:"createdAt"`
	Regions   []Region  `json:"regions,omitempty"`
}

type Region struct {
	ID            uint          `gorm:"primaryKey" json:"id"`
	EnvironmentID uint          `gorm:"not null;index" json:"environmentId"`
	Name          string        `gorm:"not null" json:"name"` // e.g. us-east-1, westeurope, free-form
	CreatedAt     time.Time     `json:"createdAt"`
	CloudAccounts []CloudAccount `json:"cloudAccounts,omitempty"`
}

// ProviderKey identifies which adapter/integration handles a CloudAccount.
type ProviderKey string

const (
	ProviderAWS     ProviderKey = "aws"
	ProviderAzure   ProviderKey = "azure"
	ProviderGCP     ProviderKey = "gcp"
	ProviderOCI     ProviderKey = "oci"
	ProviderContabo ProviderKey = "contabo" // no cost API -> CSV/manual import
	ProviderGeneric ProviderKey = "generic" // any other provider -> CSV/manual import
)

// ProviderCredential holds provider-specific auth material, encrypted at rest.
type ProviderCredential struct {
	ID               uint        `gorm:"primaryKey" json:"id"`
	Provider         ProviderKey `gorm:"not null" json:"provider"`
	Name             string      `gorm:"not null" json:"name"`
	EncryptedPayload string      `gorm:"not null" json:"-"` // JSON blob of provider-specific fields, encrypted
	CreatedAt        time.Time   `json:"createdAt"`
}

// CloudAccount maps one real-world cloud account/subscription/tenancy to a
// place in the org hierarchy, and to the credential used to sync its costs.
type CloudAccount struct {
	ID           uint        `gorm:"primaryKey" json:"id"`
	RegionID     uint        `gorm:"not null;index" json:"regionId"`
	Provider     ProviderKey `gorm:"not null" json:"provider"`
	Name         string      `gorm:"not null" json:"name"`
	ExternalID   string      `gorm:"not null" json:"externalId"` // AWS account id / Azure subscription id / GCP project id / OCI tenancy OCID / free text
	CredentialID *uint       `json:"credentialId,omitempty"`
	Active       bool        `gorm:"not null;default:true" json:"active"`
	LastSyncAt   *time.Time  `json:"lastSyncAt,omitempty"`
	LastSyncErr  string      `json:"lastSyncErr,omitempty"`
	CreatedAt    time.Time   `json:"createdAt"`
}

// CostRecord is one day's spend for one service under one CloudAccount.
// The unique index makes re-syncing the same day/service idempotent (upsert).
type CostRecord struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	CloudAccountID uint      `gorm:"not null;uniqueIndex:idx_cost_unique" json:"cloudAccountId"`
	Date           time.Time `gorm:"not null;uniqueIndex:idx_cost_unique" json:"date"`
	ServiceName    string    `gorm:"not null;uniqueIndex:idx_cost_unique" json:"serviceName"`
	Amount         float64   `gorm:"not null" json:"amount"`
	Currency       string    `gorm:"not null;default:USD" json:"currency"`
	CreatedAt      time.Time `json:"createdAt"`
}
