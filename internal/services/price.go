package services

import (
	"github.com/romanzipp/domain-manager/internal/models"
	"gorm.io/gorm"
)

type PriceService struct {
	db *gorm.DB
}

func NewPriceService(db *gorm.DB) *PriceService {
	return &PriceService{db: db}
}

// ComputedPrice returns the effective price for a domain:
// domain override > registrar TLD-specific > registrar catch-all > nil
func (s *PriceService) ComputedPrice(domain *models.Domain) *models.Price {
	var price models.Price

	// Domain-level override
	if err := s.db.Where("domain_id = ?", domain.ID).First(&price).Error; err == nil {
		return &price
	}

	if domain.RegistrarID == nil {
		return nil
	}

	// Registrar TLD-specific price
	if err := s.db.Where("registrar_id = ? AND tld = ?", *domain.RegistrarID, domain.TLD).First(&price).Error; err == nil {
		return &price
	}

	// Registrar catch-all
	if err := s.db.Where("registrar_id = ? AND tld = ''", *domain.RegistrarID).First(&price).Error; err == nil {
		return &price
	}

	return nil
}
