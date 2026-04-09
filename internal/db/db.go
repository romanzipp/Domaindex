package db

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/romanzipp/domain-manager/internal/config"
	"github.com/romanzipp/domain-manager/internal/models"
	"github.com/romanzipp/domain-manager/internal/seeds"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfg.DBDriver {
	case "postgres":
		dialector = postgres.Open(cfg.DBDSN)
	default:
		if err := ensureDir(cfg.DBDSN); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
		dialector = sqlite.Open(cfg.DBDSN)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	if err := seeds.ForAllUsers(db); err != nil {
		return nil, fmt.Errorf("seed: %w", err)
	}

	return db, nil
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.Registrar{},
		&models.Price{},
		&models.Domain{},
		&models.Notification{},
	)
}

func ensureDir(dsn string) error {
	dir := filepath.Dir(dsn)
	if dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0755)
}
