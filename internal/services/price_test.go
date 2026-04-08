package services_test

import (
	"testing"

	"github.com/romanzipp/domain-manager/internal/models"
	"github.com/romanzipp/domain-manager/internal/services"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Registrar{}, &models.Price{}, &models.Domain{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestComputedPrice_DomainOverride(t *testing.T) {
	db := setupTestDB(t)
	svc := services.NewPriceService(db)

	user := models.User{Username: "test", Email: "test@test.com", PasswordHash: "x"}
	db.Create(&user)

	reg := models.Registrar{UserID: user.ID, Name: "TestReg", Currency: "USD"}
	db.Create(&reg)

	domain := models.Domain{UserID: user.ID, RegistrarID: &reg.ID, Name: "example.com", TLD: "com"}
	db.Create(&domain)

	// Registrar catch-all price
	catchAll := models.Price{RegistrarID: &reg.ID, TLD: "", RenewPerYear: 10.00}
	db.Create(&catchAll)

	// Domain-specific override
	override := models.Price{DomainID: &domain.ID, RenewPerYear: 5.00}
	db.Create(&override)

	price := svc.ComputedPrice(&domain)
	if price == nil {
		t.Fatal("expected price, got nil")
	}
	if price.RenewPerYear != 5.00 {
		t.Errorf("expected domain override 5.00, got %.2f", price.RenewPerYear)
	}
}

func TestComputedPrice_RegistrarTLD(t *testing.T) {
	db := setupTestDB(t)
	svc := services.NewPriceService(db)

	user := models.User{Username: "test2", Email: "test2@test.com", PasswordHash: "x"}
	db.Create(&user)

	reg := models.Registrar{UserID: user.ID, Name: "TestReg2", Currency: "USD"}
	db.Create(&reg)

	domain := models.Domain{UserID: user.ID, RegistrarID: &reg.ID, Name: "example.com", TLD: "com"}
	db.Create(&domain)

	catchAll := models.Price{RegistrarID: &reg.ID, TLD: "", RenewPerYear: 10.00}
	db.Create(&catchAll)

	tldPrice := models.Price{RegistrarID: &reg.ID, TLD: "com", RenewPerYear: 12.00}
	db.Create(&tldPrice)

	price := svc.ComputedPrice(&domain)
	if price == nil {
		t.Fatal("expected price, got nil")
	}
	if price.RenewPerYear != 12.00 {
		t.Errorf("expected TLD price 12.00, got %.2f", price.RenewPerYear)
	}
}

func TestComputedPrice_CatchAll(t *testing.T) {
	db := setupTestDB(t)
	svc := services.NewPriceService(db)

	user := models.User{Username: "test3", Email: "test3@test.com", PasswordHash: "x"}
	db.Create(&user)

	reg := models.Registrar{UserID: user.ID, Name: "TestReg3", Currency: "USD"}
	db.Create(&reg)

	domain := models.Domain{UserID: user.ID, RegistrarID: &reg.ID, Name: "example.org", TLD: "org"}
	db.Create(&domain)

	catchAll := models.Price{RegistrarID: &reg.ID, TLD: "", RenewPerYear: 10.00}
	db.Create(&catchAll)

	price := svc.ComputedPrice(&domain)
	if price == nil {
		t.Fatal("expected price, got nil")
	}
	if price.RenewPerYear != 10.00 {
		t.Errorf("expected catch-all 10.00, got %.2f", price.RenewPerYear)
	}
}

func TestComputedPrice_NoRegistrar(t *testing.T) {
	db := setupTestDB(t)
	svc := services.NewPriceService(db)

	user := models.User{Username: "test4", Email: "test4@test.com", PasswordHash: "x"}
	db.Create(&user)

	domain := models.Domain{UserID: user.ID, Name: "example.com", TLD: "com"}
	db.Create(&domain)

	price := svc.ComputedPrice(&domain)
	if price != nil {
		t.Errorf("expected nil for domain with no registrar, got %+v", price)
	}
}
