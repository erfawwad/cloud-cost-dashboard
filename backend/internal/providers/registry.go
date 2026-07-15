package providers

import "cloudcostdash/internal/models"

// Registry returns the CostProvider for a given provider key, or nil if the
// provider is manual-import-only (Contabo, Generic) or unknown.
func Registry() map[models.ProviderKey]CostProvider {
	return map[models.ProviderKey]CostProvider{
		models.ProviderAWS:   &AWSProvider{},
		models.ProviderAzure: &AzureProvider{},
		models.ProviderGCP:   &GCPProvider{},
		models.ProviderOCI:   &OCIProvider{},
		// Contabo and Generic have no live cost API — costs are added via
		// CSV import (see csv.go / the /api/costs/import endpoint) instead
		// of the scheduled sync loop.
	}
}
