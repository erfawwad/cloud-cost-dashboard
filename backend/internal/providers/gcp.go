package providers

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/civil"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// GCPProvider reads daily per-service cost from a BigQuery Billing Export
// table. Unlike AWS/Azure, GCP has no direct "get my costs" REST API for
// itemized cost — Cloud Billing must be configured to export daily detailed
// cost to a BigQuery dataset first:
// https://cloud.google.com/billing/docs/how-to/export-data-bigquery-setup
//
// Required IAM role on the credential's service account: BigQuery Data
// Viewer + BigQuery Job User on the billing export project (read-only).
//
// Expected AccountConfig.Credential fields:
//   service_account_json  - full JSON key of the service account
//   bq_project            - GCP project that owns the billing export dataset
//   bq_dataset            - billing export dataset name
//   bq_table               - billing export table name
// AccountConfig.ExternalID must be the billed GCP project id (used to filter
// the shared billing export table down to this one CloudAccount).
type GCPProvider struct{}

func (p *GCPProvider) Name() string { return "gcp" }

func (p *GCPProvider) FetchCosts(ctx context.Context, account AccountConfig, r DateRange) ([]CostRecord, error) {
	saJSON := account.Credential["service_account_json"]
	bqProject := account.Credential["bq_project"]
	bqDataset := account.Credential["bq_dataset"]
	bqTable := account.Credential["bq_table"]
	if saJSON == "" || bqProject == "" || bqDataset == "" || bqTable == "" {
		return nil, fmt.Errorf("gcp: missing service_account_json/bq_project/bq_dataset/bq_table credential fields")
	}
	if account.ExternalID == "" {
		return nil, fmt.Errorf("gcp: missing billed project id (CloudAccount.ExternalID)")
	}

	client, err := bigquery.NewClient(ctx, bqProject, option.WithCredentialsJSON([]byte(saJSON)))
	if err != nil {
		return nil, fmt.Errorf("gcp: bigquery client: %w", err)
	}
	defer client.Close()

	query := client.Query(fmt.Sprintf(`
		SELECT
		  service.description AS service_name,
		  DATE(usage_start_time) AS usage_date,
		  SUM(cost) AS amount,
		  ANY_VALUE(currency) AS currency
		FROM `+"`%s.%s.%s`"+`
		WHERE project.id = @project_id
		  AND DATE(usage_start_time) BETWEEN @start_date AND @end_date
		GROUP BY service_name, usage_date
	`, bqProject, bqDataset, bqTable))

	query.Parameters = []bigquery.QueryParameter{
		{Name: "project_id", Value: account.ExternalID},
		{Name: "start_date", Value: civil.DateOf(r.Start)},
		{Name: "end_date", Value: civil.DateOf(r.End)},
	}

	it, err := query.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("gcp: query billing export: %w", err)
	}

	var records []CostRecord
	for {
		var row struct {
			ServiceName string     `bigquery:"service_name"`
			UsageDate   civil.Date `bigquery:"usage_date"`
			Amount      float64    `bigquery:"amount"`
			Currency    string     `bigquery:"currency"`
		}
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("gcp: read row: %w", err)
		}
		records = append(records, CostRecord{
			Date:        time.Date(row.UsageDate.Year, row.UsageDate.Month, row.UsageDate.Day, 0, 0, 0, 0, time.UTC),
			ServiceName: row.ServiceName,
			Amount:      row.Amount,
			Currency:    row.Currency,
		})
	}

	return records, nil
}
