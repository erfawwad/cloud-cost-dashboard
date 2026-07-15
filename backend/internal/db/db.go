package db

import (
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite" // pure-Go sqlite driver, no cgo/C compiler needed
	"gorm.io/gorm"

	"cloudcostdash/internal/models"
)

func Open(path string) (*gorm.DB, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	dbConn, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := dbConn.AutoMigrate(
		&models.User{},
		&models.Organization{},
		&models.Product{},
		&models.Project{},
		&models.Environment{},
		&models.Region{},
		&models.ProviderCredential{},
		&models.CloudAccount{},
		&models.CostRecord{},
	); err != nil {
		return nil, err
	}

	return dbConn, nil
}
