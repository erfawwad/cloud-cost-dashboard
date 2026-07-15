package api

import (
	"encoding/json"
	"net/http"

	"cloudcostdash/internal/models"
	"cloudcostdash/internal/providers"
	"cloudcostdash/internal/scheduler"
)

// ---- provider credentials ---------------------------------------------------
// Credential secrets are encrypted before storage and never returned by the
// API (ProviderCredential.EncryptedPayload is tagged json:"-").

type createCredentialRequest struct {
	Provider models.ProviderKey `json:"provider"`
	Name     string             `json:"name"`
	Fields   map[string]string  `json:"fields"`
}

func (s *Server) handleCreateCredential(w http.ResponseWriter, r *http.Request) {
	var req createCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.Provider == "" {
		writeError(w, http.StatusBadRequest, "provider and name are required")
		return
	}

	payload, err := json.Marshal(req.Fields)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid fields")
		return
	}
	encrypted, err := s.Box.Encrypt(payload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encrypt credential")
		return
	}

	cred := models.ProviderCredential{
		Provider:         req.Provider,
		Name:             req.Name,
		EncryptedPayload: encrypted,
	}
	if err := s.DB.Create(&cred).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store credential")
		return
	}
	writeJSON(w, http.StatusCreated, cred)
}

func (s *Server) handleListCredentials(w http.ResponseWriter, r *http.Request) {
	creds := []models.ProviderCredential{}
	if err := s.DB.Find(&creds).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list credentials")
		return
	}
	writeJSON(w, http.StatusOK, creds)
}

func (s *Server) handleDeleteCredential(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamUint(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var inUse int64
	s.DB.Model(&models.CloudAccount{}).Where("credential_id = ?", id).Count(&inUse)
	if inUse > 0 {
		writeError(w, http.StatusConflict, "credential is still attached to a cloud account")
		return
	}
	if err := s.DB.Delete(&models.ProviderCredential{}, id).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete credential")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- cloud accounts ---------------------------------------------------------

type createCloudAccountRequest struct {
	RegionID     uint               `json:"regionId"`
	Provider     models.ProviderKey `json:"provider"`
	Name         string             `json:"name"`
	ExternalID   string             `json:"externalId"`
	CredentialID *uint              `json:"credentialId"`
}

func (s *Server) handleCreateCloudAccount(w http.ResponseWriter, r *http.Request) {
	var req createCloudAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" || req.RegionID == 0 || req.Provider == "" {
		writeError(w, http.StatusBadRequest, "regionId, provider and name are required")
		return
	}
	account := models.CloudAccount{
		RegionID:     req.RegionID,
		Provider:     req.Provider,
		Name:         req.Name,
		ExternalID:   req.ExternalID,
		CredentialID: req.CredentialID,
		Active:       true,
	}
	if err := s.DB.Create(&account).Error; err != nil {
		writeError(w, http.StatusBadRequest, "could not create cloud account")
		return
	}
	writeJSON(w, http.StatusCreated, account)
}

type updateCloudAccountRequest struct {
	Name         *string `json:"name"`
	ExternalID   *string `json:"externalId"`
	CredentialID *uint   `json:"credentialId"`
	Active       *bool   `json:"active"`
}

func (s *Server) handleUpdateCloudAccount(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamUint(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req updateCloudAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updates := map[string]interface{}{}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.ExternalID != nil {
		updates["external_id"] = *req.ExternalID
	}
	if req.CredentialID != nil {
		updates["credential_id"] = *req.CredentialID
	}
	if req.Active != nil {
		updates["active"] = *req.Active
	}

	if err := s.DB.Model(&models.CloudAccount{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update cloud account")
		return
	}

	var account models.CloudAccount
	s.DB.First(&account, id)
	writeJSON(w, http.StatusOK, account)
}

func (s *Server) handleDeleteCloudAccount(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamUint(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := s.DB.Where("cloud_account_id = ?", id).Delete(&models.CostRecord{}).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete cost history")
		return
	}
	if err := s.DB.Delete(&models.CloudAccount{}, id).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete cloud account")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleSyncNow triggers an immediate cost sync for one cloud account,
// instead of waiting for the next scheduled run. Only works for providers
// with a live adapter (AWS/Azure/GCP/OCI) — Contabo/Generic use CSV import.
func (s *Server) handleSyncNow(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamUint(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var account models.CloudAccount
	if err := s.DB.First(&account, id).Error; err != nil {
		writeError(w, http.StatusNotFound, "cloud account not found")
		return
	}

	provider, ok := providers.Registry()[account.Provider]
	if !ok {
		writeError(w, http.StatusBadRequest, "this provider has no live sync; use CSV import instead")
		return
	}

	if err := s.Scheduler.SyncAccount(r.Context(), provider, account); err != nil {
		writeError(w, http.StatusBadGateway, "sync failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "synced"})
}

// handleImportCSV bulk-loads cost records from an uploaded CSV file
// (date, service_name, amount, currency columns) into one cloud account.
// This is the primary way Contabo (no cost API) and any Generic/custom
// provider get their cost data in, but it can be used to backfill any
// account's history too.
func (s *Server) handleImportCSV(w http.ResponseWriter, r *http.Request) {
	id, err := urlParamUint(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	records, err := providers.ParseCostCSV(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := scheduler.ImportManualRecords(s.DB, id, records); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store imported costs")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"imported": len(records)})
}
