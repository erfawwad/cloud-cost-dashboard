package providers

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// ParseCostCSV parses a manually uploaded cost file, used for Contabo (which
// has no cost API) and any Generic/custom provider added without writing a
// dedicated adapter.
//
// Expected columns (header row required, any column order):
//   date, service_name, amount, currency
// date must be YYYY-MM-DD. currency is optional and defaults to USD.
func ParseCostCSV(r io.Reader) ([]CostRecord, error) {
	reader := csv.NewReader(r)
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("csv: %w", err)
	}
	if len(rows) < 1 {
		return nil, fmt.Errorf("csv: empty file")
	}

	colIndex := map[string]int{}
	for i, col := range rows[0] {
		colIndex[strings.ToLower(strings.TrimSpace(col))] = i
	}
	dateIdx, hasDate := colIndex["date"]
	serviceIdx, hasService := colIndex["service_name"]
	amountIdx, hasAmount := colIndex["amount"]
	if !hasDate || !hasService || !hasAmount {
		return nil, fmt.Errorf("csv: header must include date, service_name, amount columns")
	}
	currencyIdx, hasCurrency := colIndex["currency"]

	var records []CostRecord
	for lineNo, row := range rows[1:] {
		if len(row) <= dateIdx || len(row) <= serviceIdx || len(row) <= amountIdx {
			return nil, fmt.Errorf("csv: row %d: not enough columns", lineNo+2)
		}
		date, err := time.Parse("2006-01-02", strings.TrimSpace(row[dateIdx]))
		if err != nil {
			return nil, fmt.Errorf("csv: row %d: invalid date %q (want YYYY-MM-DD)", lineNo+2, row[dateIdx])
		}
		amount, err := strconv.ParseFloat(strings.TrimSpace(row[amountIdx]), 64)
		if err != nil {
			return nil, fmt.Errorf("csv: row %d: invalid amount %q", lineNo+2, row[amountIdx])
		}
		currency := "USD"
		if hasCurrency && len(row) > currencyIdx && strings.TrimSpace(row[currencyIdx]) != "" {
			currency = strings.TrimSpace(row[currencyIdx])
		}
		records = append(records, CostRecord{
			Date:        date,
			ServiceName: strings.TrimSpace(row[serviceIdx]),
			Amount:      amount,
			Currency:    currency,
		})
	}

	return records, nil
}
