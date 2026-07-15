package providers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

// AWSProvider reads daily per-service cost from AWS Cost Explorer.
//
// Required IAM permissions on the credential used: ce:GetCostAndUsage
// (read-only). Cost Explorer's API endpoint is only available in us-east-1
// regardless of which region the account's resources actually run in.
//
// Expected AccountConfig.Credential fields:
//   access_key_id, secret_access_key
type AWSProvider struct{}

func (p *AWSProvider) Name() string { return "aws" }

func (p *AWSProvider) FetchCosts(ctx context.Context, account AccountConfig, r DateRange) ([]CostRecord, error) {
	accessKey := account.Credential["access_key_id"]
	secretKey := account.Credential["secret_access_key"]
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("aws: missing access_key_id/secret_access_key credential fields")
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("aws: load config: %w", err)
	}
	client := costexplorer.NewFromConfig(cfg)

	var records []CostRecord
	var pageToken *string

	for {
		out, err := client.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
			TimePeriod: &types.DateInterval{
				Start: aws.String(r.Start.Format("2006-01-02")),
				End:   aws.String(r.End.Format("2006-01-02")),
			},
			Granularity: types.GranularityDaily,
			Metrics:     []string{"UnblendedCost"},
			GroupBy: []types.GroupDefinition{
				{Type: types.GroupDefinitionTypeDimension, Key: aws.String("SERVICE")},
			},
			NextPageToken: pageToken,
		})
		if err != nil {
			return nil, fmt.Errorf("aws: GetCostAndUsage: %w", err)
		}

		for _, byTime := range out.ResultsByTime {
			date, err := time.Parse("2006-01-02", aws.ToString(byTime.TimePeriod.Start))
			if err != nil {
				continue
			}
			for _, group := range byTime.Groups {
				if len(group.Keys) == 0 {
					continue
				}
				metric, ok := group.Metrics["UnblendedCost"]
				if !ok {
					continue
				}
				amount, err := strconv.ParseFloat(aws.ToString(metric.Amount), 64)
				if err != nil {
					continue
				}
				records = append(records, CostRecord{
					Date:        date,
					ServiceName: group.Keys[0],
					Amount:      amount,
					Currency:    aws.ToString(metric.Unit),
				})
			}
		}

		if out.NextPageToken == nil {
			break
		}
		pageToken = out.NextPageToken
	}

	return records, nil
}
