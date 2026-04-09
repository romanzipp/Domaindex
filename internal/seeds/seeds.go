package seeds

import (
	"time"

	"gorm.io/gorm"
)

type SeedRun struct {
	ID    string    `gorm:"primarykey"`
	RanAt time.Time `gorm:"autoCreateTime"`
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(&SeedRun{})
}

func hasRun(db *gorm.DB, id string) bool {
	var count int64
	db.Model(&SeedRun{}).Where("id = ?", id).Count(&count)
	return count > 0
}

func markRun(db *gorm.DB, id string) {
	db.Create(&SeedRun{ID: id, RanAt: time.Now()})
}

// ForUser runs all seeds for a given user. Safe to call multiple times.
func ForUser(db *gorm.DB, userID uint) error {
	if err := migrate(db); err != nil {
		return err
	}
	if err := seedRegistrars(db, userID); err != nil {
		return err
	}
	return seedRegistrarPrices(db, userID)
}

// ForAllUsers runs all seeds for every user in the database.
func ForAllUsers(db *gorm.DB) error {
	if err := migrate(db); err != nil {
		return err
	}

	rows, err := db.Raw("SELECT id FROM users").Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var userID uint
		if err := rows.Scan(&userID); err != nil {
			return err
		}
		if err := seedRegistrars(db, userID); err != nil {
			return err
		}
	}

	return nil
}
