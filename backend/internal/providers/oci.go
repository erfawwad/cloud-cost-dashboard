package providers

import (
	"context"
	"fmt"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/usageapi"
)

// OCIProvider reads daily per-service cost from the OCI Usage API, at the
// tenancy level.
//
// Required IAM policy for the credential's user: read access to usage-report
// resources in the tenancy (read-only).
//
// Expected AccountConfig.Credential fields:
//   user_ocid, fingerprint, private_key_pem, region
//   private_key_passphrase (optional, only if the key is encrypted)
// AccountConfig.ExternalID must be the tenancy OCID.
type OCIProvider struct{}

func (p *OCIProvider) Name() string { return "oci" }

func (p *OCIProvider) FetchCosts(ctx context.Context, account AccountConfig, r DateRange) ([]CostRecord, error) {
	tenancyOCID := account.ExternalID
	userOCID := account.Credential["user_ocid"]
	fingerprint := account.Credential["fingerprint"]
	privateKeyPEM := account.Credential["private_key_pem"]
	region := account.Credential["region"]
	if tenancyOCID == "" || userOCID == "" || fingerprint == "" || privateKeyPEM == "" || region == "" {
		return nil, fmt.Errorf("oci: missing tenancy ocid / user_ocid / fingerprint / private_key_pem / region credential fields")
	}

	var passphrase *string
	if pp := account.Credential["private_key_passphrase"]; pp != "" {
		passphrase = &pp
	}

	provider := common.NewRawConfigurationProvider(tenancyOCID, userOCID, region, fingerprint, privateKeyPEM, passphrase)

	client, err := usageapi.NewUsageapiClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("oci: client: %w", err)
	}

	req := usageapi.RequestSummarizedUsagesRequest{
		RequestSummarizedUsagesDetails: usageapi.RequestSummarizedUsagesDetails{
			TenantId:         common.String(tenancyOCID),
			TimeUsageStarted: &common.SDKTime{Time: r.Start},
			TimeUsageEnded:   &common.SDKTime{Time: r.End},
			Granularity:      usageapi.RequestSummarizedUsagesDetailsGranularityDaily,
			GroupBy:          []string{"service"},
		},
	}

	resp, err := client.RequestSummarizedUsages(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("oci: RequestSummarizedUsages: %w", err)
	}

	var records []CostRecord
	for _, item := range resp.Items {
		if item.TimeUsageStarted == nil || item.ComputedAmount == nil {
			continue
		}
		serviceName := "unknown"
		if item.Service != nil {
			serviceName = *item.Service
		}
		currency := "USD"
		if item.Currency != nil {
			currency = *item.Currency
		}
		records = append(records, CostRecord{
			Date:        item.TimeUsageStarted.Time,
			ServiceName: serviceName,
			Amount:      float64(*item.ComputedAmount),
			Currency:    currency,
		})
	}

	return records, nil
}
