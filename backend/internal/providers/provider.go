// Package providers defines the pluggable interface each cloud cost
// integration implements, so adding a new provider never requires touching
// the core sync/API/scheduler code.
package providers

import (
	"context"
	"time"
)

// CostRecord is one day's spend for one service, as returned by a provider.
type CostRecord struct {
	Date        time.Time
	ServiceName string
	Amount      float64
	Currency    string
}

// AccountConfig is everything an adapter needs to fetch costs for one
// real-world cloud account/subscription/tenancy.
type AccountConfig struct {
	ExternalID string            // AWS account id / Azure subscription id / GCP project id / OCI tenancy OCID
	Credential map[string]string // decrypted provider-specific fields (keys, secrets, tenant id, etc.)
}

type DateRange struct {
	Start time.Time
	End   time.Time
}

// CostProvider is implemented once per cloud provider. FetchCosts must return
// daily, per-service cost records for the given date range only — no
// aggregation, pagination handling, or hierarchy logic belongs here.
type CostProvider interface {
	Name() string
	FetchCosts(ctx context.Context, account AccountConfig, r DateRange) ([]CostRecord, error)
}

// ErrManualImportOnly is returned by adapters (Contabo, Generic) that have no
// live cost API — their numbers must come from CSV/manual import instead.
type ManualImportOnlyError struct{ Provider string }

func (e *ManualImportOnlyError) Error() string {
	return e.Provider + " has no cost API; import costs via CSV instead of scheduled sync"
}
