package services_test

import (
	"testing"

	"github.com/romanzipp/domaindex/internal/models"
	"github.com/romanzipp/domaindex/internal/services"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupNotifDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_busy_timeout=5000"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.AutoMigrate(&models.User{}, &models.Registrar{}, &models.Domain{}, &models.Notification{})
	return db
}

func TestNotification_NoDuplicatePerDay(t *testing.T) {
	db := setupNotifDB(t)
	svc := services.NewNotificationService(db, "", "")

	user := models.User{Username: "notiftest", Email: "n@test.com", PasswordHash: "x"}
	db.Create(&user)
	domain := models.Domain{UserID: user.ID, Name: "expiring.com", TLD: "com"}
	db.Create(&domain)

	err := svc.Send(user.ID, domain.ID, models.NotificationTypeExpiry7d, "expires soon")
	if err != nil {
		t.Fatalf("first send: %v", err)
	}

	err = svc.Send(user.ID, domain.ID, models.NotificationTypeExpiry7d, "expires soon")
	if err != nil {
		t.Fatalf("second send: %v", err)
	}

	var count int64
	db.Model(&models.Notification{}).Where("domain_id = ? AND type = ?", domain.ID, models.NotificationTypeExpiry7d).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 notification (dedup), got %d", count)
	}
}

func TestNotification_DifferentTypesAllowed(t *testing.T) {
	db := setupNotifDB(t)
	svc := services.NewNotificationService(db, "", "")

	user := models.User{Username: "notiftest2", Email: "n2@test.com", PasswordHash: "x"}
	db.Create(&user)
	domain := models.Domain{UserID: user.ID, Name: "expiring2.com", TLD: "com"}
	db.Create(&domain)

	svc.Send(user.ID, domain.ID, models.NotificationTypeExpiry7d, "7d")
	svc.Send(user.ID, domain.ID, models.NotificationTypeWhoisChanged, "changed")

	var count int64
	db.Model(&models.Notification{}).Where("domain_id = ?", domain.ID).Count(&count)
	if count != 2 {
		t.Errorf("expected 2 notifications for different types, got %d", count)
	}
}
