package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"cloudcostdash/internal/api"
	"cloudcostdash/internal/auth"
	"cloudcostdash/internal/config"
	"cloudcostdash/internal/crypto"
	"cloudcostdash/internal/db"
	"cloudcostdash/internal/models"
	"cloudcostdash/internal/scheduler"
)

func main() {
	// Optional: only present when running the binary directly (`go run`,
	// or a bare `./server`) outside Docker. Ignored if absent — real env
	// vars (set by docker-compose's env_file, systemd, etc.) always win.
	_ = godotenv.Load()

	cfg := config.Load()

	gormDB, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}

	box, err := crypto.New(cfg.EncryptionKey)
	if err != nil {
		log.Fatalf("failed to init encryption: %v", err)
	}

	bootstrapAdmin(gormDB)

	sched := scheduler.New(gormDB, box)
	sched.Start(cfg.SyncIntervalMi)
	defer sched.Stop()

	server := &api.Server{
		DB:        gormDB,
		Box:       box,
		JWTSecret: cfg.JWTSecret,
		Scheduler: sched,
	}

	log.Printf("cloud cost dashboard listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, server.Router()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// bootstrapAdmin creates the first admin user from ADMIN_EMAIL/ADMIN_PASSWORD
// env vars if no users exist yet, so there's always a way to log in on first run.
func bootstrapAdmin(gormDB *gorm.DB) {
	var count int64
	gormDB.Model(&models.User{}).Count(&count)
	if count > 0 {
		return
	}

	email := os.Getenv("ADMIN_EMAIL")
	password := os.Getenv("ADMIN_PASSWORD")
	if email == "" {
		email = "admin@example.com"
	}
	if password == "" {
		password = "changeme123"
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		log.Fatalf("failed to hash bootstrap admin password: %v", err)
	}

	admin := models.User{Email: email, PasswordHash: hash, Role: models.RoleAdmin, ScopeType: models.ScopeNone}
	if err := gormDB.Create(&admin).Error; err != nil {
		log.Fatalf("failed to create bootstrap admin: %v", err)
	}

	log.Printf("created initial admin user %s (set ADMIN_EMAIL/ADMIN_PASSWORD env vars to control this) — log in and change the password", email)
}
