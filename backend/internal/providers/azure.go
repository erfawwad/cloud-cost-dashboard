package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/costmanagement/armcostmanagement"
)

// AzureProvider reads daily per-service cost from Azure Cost Management.
//
// Required role on the credential's service principal: Cost Management
// Reader on the subscription (read-only).
//
// Expected AccountConfig.Credential fields:
//   tenant_id, client_id, client_secret
// AccountConfig.ExternalID must be the Azure subscription id.
type AzureProvider struct{}

func (p *AzureProvider) Name() string { return "azure" }

func (p *AzureProvider) FetchCosts(ctx context.Context, account AccountConfig, r DateRange) ([]CostRecord, error) {
	tenantID := account.Credential["tenant_id"]
	clientID := account.Credential["client_id"]
	clientSecret := account.Credential["client_secret"]
	if tenantID == "" || clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("azure: missing tenant_id/client_id/client_secret credential fields")
	}
	if account.ExternalID == "" {
		return nil, fmt.Errorf("azure: missing subscription id (CloudAccount.ExternalID)")
	}

	cred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	if err != nil {
		return nil, fmt.Errorf("azure: credential: %w", err)
	}

	client, err := armcostmanagement.NewQueryClient(cred, nil)
	if err != nil {
		return nil, fmt.Errorf("azure: client: %w", err)
	}

	scope := fmt.Sprintf("/subscriptions/%s", account.ExternalID)

	result, err := client.Usage(ctx, scope, armcostmanagement.QueryDefinition{
		Type:      to.Ptr(armcostmanagement.ExportTypeUsage),
		Timeframe: to.Ptr(armcostmanagement.TimeframeTypeCustom),
		TimePeriod: &armcostmanagement.QueryTimePeriod{
			From: to.Ptr(r.Start),
			To:   to.Ptr(r.End),
		},
		Dataset: &armcostmanagement.QueryDataset{
			Granularity: to.Ptr(armcostmanagement.GranularityTypeDaily),
			Aggregation: map[string]*armcostmanagement.QueryAggregation{
				"totalCost": {
					Name:     to.Ptr("Cost"),
					Function: to.Ptr(armcostmanagement.FunctionTypeSum),
				},
			},
			Grouping: []*armcostmanagement.QueryGrouping{
				{
					Type: to.Ptr(armcostmanagement.QueryColumnTypeDimension),
					Name: to.Ptr("ServiceName"),
				},
			},
		},
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("azure: Usage query: %w", err)
	}

	if result.Properties == nil {
		return nil, nil
	}

	colIndex := map[string]int{}
	for i, col := range result.Properties.Columns {
		if col != nil && col.Name != nil {
			colIndex[*col.Name] = i
		}
	}
	costIdx, hasCost := colIndex["Cost"]
	dateIdx, hasDate := colIndex["UsageDate"]
	serviceIdx, hasService := colIndex["ServiceName"]
	currencyIdx, hasCurrency := colIndex["Currency"]
	if !hasCost || !hasDate || !hasService {
		return nil, fmt.Errorf("azure: unexpected result columns: %v", colIndex)
	}

	var records []CostRecord
	for _, row := range result.Properties.Rows {
		if len(row) <= costIdx || len(row) <= dateIdx || len(row) <= serviceIdx {
			continue
		}
		amount, ok := toFloat(row[costIdx])
		if !ok {
			continue
		}
		dateNum, ok := toFloat(row[dateIdx])
		if !ok {
			continue
		}
		date, err := time.Parse("20060102", fmt.Sprintf("%.0f", dateNum))
		if err != nil {
			continue
		}
		serviceName, _ := row[serviceIdx].(string)
		currency := "USD"
		if hasCurrency && len(row) > currencyIdx {
			if c, ok := row[currencyIdx].(string); ok && c != "" {
				currency = c
			}
		}
		records = append(records, CostRecord{
			Date:        date,
			ServiceName: serviceName,
			Amount:      amount,
			Currency:    currency,
		})
	}

	return records, nil
}

func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}
