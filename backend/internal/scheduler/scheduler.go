// Package scheduler runs the periodic cost sync: for every active
// CloudAccount with a live provider (AWS/Azure/GCP/OCI), fetch the latest
// costs and upsert them into the database.
package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"cloudcostdash/internal/crypto"
	"cloudcostdash/internal/models"
	"cloudcostdash/internal/providers"
)

// syncWindowDays controls how far back each sync re-fetches costs. Cloud
// providers often revise the last few days of billing data as it finalizes,
// so re-pulling a rolling window (rather than just "since last sync") keeps
// already-stored numbers accurate. The unique index on CostRecord makes this
// an idempotent upsert, not a duplicate insert.
const syncWindowDays = 14

type Scheduler struct {
	db   *gorm.DB
	box  *crypto.Box
	reg  map[models.ProviderKey]providers.CostProvider
	cron *cron.Cron
}

func New(db *gorm.DB, box *crypto.Box) *Scheduler {
	return &Scheduler{
		db:  db,
		box: box,
		reg: providers.Registry(),
	}
}

// Start schedules SyncAll to run every intervalMinutes, and kicks off one
// run immediately in the background so data isn't empty on first boot.
func (s *Scheduler) Start(intervalMinutes int) {
	s.cron = cron.New()
	spec := fmt.Sprintf("@every %dm", intervalMinutes)
	if _, err := s.cron.AddFunc(spec, func() {
		s.SyncAll(context.Background())
	}); err != nil {
		log.Printf("scheduler: failed to schedule sync: %v", err)
		return
	}
	s.cron.Start()

	go s.SyncAll(context.Background())
}

func (s *Scheduler) Stop() {
	if s.cron != nil {
		s.cron.Stop()
	}
}

// SyncAll fetches and stores costs for every active cloud account that has a
// live provider adapter (Contabo/Generic accounts are skipped — they're
// updated via CSV import instead).
func (s *Scheduler) SyncAll(ctx context.Context) {
	var accounts []models.CloudAccount
	if err := s.db.Where("active = ?", true).Find(&accounts).Error; err != nil {
		log.Printf("scheduler: list accounts: %v", err)
		return
	}

	for _, account := range accounts {
		provider, ok := s.reg[account.Provider]
		if !ok {
			continue // manual-import-only provider, e.g. contabo/generic
		}
		if err := s.SyncAccount(ctx, provider, account); err != nil {
			log.Printf("scheduler: sync account %d (%s): %v", account.ID, account.Name, err)
			now := time.Now()
			s.db.Model(&models.CloudAccount{}).Where("id = ?", account.ID).Updates(map[string]interface{}{
				"last_sync_at":  now,
				"last_sync_err": err.Error(),
			})
			continue
		}
	}
}

func (s *Scheduler) SyncAccount(ctx context.Context, provider providers.CostProvider, account models.CloudAccount) error {
	if account.CredentialID == nil {
		return fmt.Errorf("no credential attached")
	}

	var cred models.ProviderCredential
	if err := s.db.First(&cred, *account.CredentialID).Error; err != nil {
		return fmt.Errorf("load credential: %w", err)
	}

	plaintext, err := s.box.Decrypt(cred.EncryptedPayload)
	if err != nil {
		return fmt.Errorf("decrypt credential: %w", err)
	}

	var fields map[string]string
	if err := json.Unmarshal(plaintext, &fields); err != nil {
		return fmt.Errorf("parse credential payload: %w", err)
	}

	end := time.Now().UTC().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -syncWindowDays)

	records, err := provider.FetchCosts(ctx, providers.AccountConfig{
		ExternalID: account.ExternalID,
		Credential: fields,
	}, providers.DateRange{Start: start, End: end})
	if err != nil {
		return err
	}

	if err := upsertCostRecords(s.db, account.ID, records); err != nil {
		return fmt.Errorf("store cost records: %w", err)
	}

	now := time.Now()
	return s.db.Model(&models.CloudAccount{}).Where("id = ?", account.ID).Updates(map[string]interface{}{
		"last_sync_at":  now,
		"last_sync_err": "",
	}).Error
}

func upsertCostRecords(db *gorm.DB, cloudAccountID uint, records []providers.CostRecord) error {
	if len(records) == 0 {
		return nil
	}
	rows := make([]models.CostRecord, 0, len(records))
	for _, r := range records {
		rows = append(rows, models.CostRecord{
			CloudAccountID: cloudAccountID,
			Date:           r.Date,
			ServiceName:    r.ServiceName,
			Amount:         r.Amount,
			Currency:       r.Currency,
		})
	}

	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "cloud_account_id"}, {Name: "date"}, {Name: "service_name"}},
		DoUpdates: clause.AssignmentColumns([]string{"amount", "currency"}),
	}).Create(&rows).Error
}

// ImportManualRecords stores CSV-imported costs (Contabo/Generic) the same
// idempotent way scheduled syncs do.
func ImportManualRecords(db *gorm.DB, cloudAccountID uint, records []providers.CostRecord) error {
	return upsertCostRecords(db, cloudAccountID, records)
}
