package api

import (
	"fmt"
	"net/http"
	"time"

	appauth "cloudcostdash/internal/auth"
	"cloudcostdash/internal/models"
)

func parseDateRange(r *http.Request) (time.Time, time.Time, error) {
	end := time.Now().UTC().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -30)

	if v := r.URL.Query().Get("start"); v != "" {
		parsed, err := time.Parse("2006-01-02", v)
		if err != nil {
			return start, end, fmt.Errorf("invalid start date (want YYYY-MM-DD)")
		}
		start = parsed
	}
	if v := r.URL.Query().Get("end"); v != "" {
		parsed, err := time.Parse("2006-01-02", v)
		if err != nil {
			return start, end, fmt.Errorf("invalid end date (want YYYY-MM-DD)")
		}
		end = parsed
	}
	return start, end, nil
}

type costPoint struct {
	Label  string  `json:"label"`
	Amount float64 `json:"amount"`
}

// handleCostTimeseries returns cost broken down by day or by service, scoped
// to any node in the hierarchy (org/product/project/environment/region/account).
// Viewers are always restricted to their assigned scope, regardless of query params.
func (s *Server) handleCostTimeseries(w http.ResponseWriter, r *http.Request) {
	start, end, err := parseDateRange(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	scopeType := r.URL.Query().Get("scopeType")
	scopeIDStr := r.URL.Query().Get("scopeId")
	groupBy := r.URL.Query().Get("groupBy")
	if groupBy == "" {
		groupBy = "day"
	}

	claims, _ := appauth.FromContext(r.Context())
	if claims != nil && claims.Role == models.RoleViewer && claims.ScopeType != models.ScopeNone {
		scopeType = string(claims.ScopeType)
		if claims.ScopeID != nil {
			scopeIDStr = fmt.Sprintf("%d", *claims.ScopeID)
		}
	}

	var accountIDs []uint
	if scopeType != "" && scopeIDStr != "" {
		scopeID, err := parseUintParam(scopeIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid scopeId")
			return
		}
		accountIDs, err = s.resolveCloudAccountIDs(scopeType, scopeID)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if len(accountIDs) == 0 {
			writeJSON(w, http.StatusOK, []costPoint{})
			return
		}
	}

	query := s.DB.Model(&models.CostRecord{}).Where("date >= ? AND date <= ?", start, end)
	if accountIDs != nil {
		query = query.Where("cloud_account_id IN ?", accountIDs)
	}

	points := []costPoint{}
	if groupBy == "service" {
		err = query.Select("service_name as label, sum(amount) as amount").
			Group("service_name").Order("amount desc").Scan(&points).Error
	} else {
		err = query.Select("strftime('%Y-%m-%d', date) as label, sum(amount) as amount").
			Group("label").Order("label asc").Scan(&points).Error
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query costs")
		return
	}

	writeJSON(w, http.StatusOK, points)
}

// resolveCloudAccountIDs expands a hierarchy node (any level) down to the set
// of CloudAccount ids beneath it.
func (s *Server) resolveCloudAccountIDs(scopeType string, scopeID uint) ([]uint, error) {
	var ids []uint
	var err error

	switch scopeType {
	case "account":
		return []uint{scopeID}, nil
	case "region":
		err = s.DB.Model(&models.CloudAccount{}).Where("region_id = ?", scopeID).Pluck("id", &ids).Error
	case "environment":
		err = s.DB.Model(&models.CloudAccount{}).
			Where("region_id IN (SELECT id FROM regions WHERE environment_id = ?)", scopeID).
			Pluck("id", &ids).Error
	case "project":
		err = s.DB.Model(&models.CloudAccount{}).
			Where("region_id IN (SELECT id FROM regions WHERE environment_id IN (SELECT id FROM environments WHERE project_id = ?))", scopeID).
			Pluck("id", &ids).Error
	case "product":
		err = s.DB.Model(&models.CloudAccount{}).
			Where(`region_id IN (
				SELECT id FROM regions WHERE environment_id IN (
					SELECT id FROM environments WHERE project_id IN (
						SELECT id FROM projects WHERE product_id = ?
					)
				)
			)`, scopeID).
			Pluck("id", &ids).Error
	case "org":
		err = s.DB.Model(&models.CloudAccount{}).
			Where(`region_id IN (
				SELECT id FROM regions WHERE environment_id IN (
					SELECT id FROM environments WHERE project_id IN (
						SELECT id FROM projects WHERE product_id IN (
							SELECT id FROM products WHERE organization_id = ?
						)
					)
				)
			)`, scopeID).
			Pluck("id", &ids).Error
	default:
		return nil, fmt.Errorf("invalid scopeType: %s", scopeType)
	}

	return ids, err
}

func parseUintParam(s string) (uint, error) {
	var v uint
	_, err := fmt.Sscanf(s, "%d", &v)
	return v, err
}
