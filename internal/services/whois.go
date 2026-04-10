package services

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/likexian/whois"
	whoisparser "github.com/likexian/whois-parser"
	"github.com/romanzipp/domaindex/internal/models"
	"gorm.io/gorm"
)

type WhoisService struct {
	db *gorm.DB
}

func NewWhoisService(db *gorm.DB) *WhoisService {
	return &WhoisService{db: db}
}

type WhoisResult struct {
	Raw              string
	CreatedDate      *time.Time
	UpdatedDate      *time.Time
	ExpirationDate   *time.Time
	RegistrarName    string
	RegistrarIanaID  string
	RegistrarURL     string
	NameServers      []string
	Statuses         []string
	DNSSec           bool
}

func (s *WhoisService) Fetch(domainName string) (*WhoisResult, error) {
	raw, err := whois.Whois(domainName)
	if err != nil {
		return nil, err
	}

	result := &WhoisResult{Raw: raw}

	parsed, err := whoisparser.Parse(raw)
	if err == nil && parsed.Domain != nil {
		result.CreatedDate = parsed.Domain.CreatedDateInTime
		result.UpdatedDate = parsed.Domain.UpdatedDateInTime
		result.ExpirationDate = parsed.Domain.ExpirationDateInTime
		result.NameServers = parsed.Domain.NameServers
		result.Statuses = parsed.Domain.Status
		result.DNSSec = parsed.Domain.DNSSec
	}

	if parsed.Registrar != nil {
		result.RegistrarName = parsed.Registrar.Name
		result.RegistrarIanaID = parsed.Registrar.ID
		result.RegistrarURL = parsed.Registrar.ReferralURL
	}

	return result, nil
}

// ResolveRegistrar finds an existing registrar by IANA ID, or creates one if none exists.
// Returns nil if no IANA ID is present in the WHOIS result.
func (s *WhoisService) ResolveRegistrar(result *WhoisResult, userID uint) *uint {
	if result.RegistrarIanaID == "" {
		return nil
	}
	var reg models.Registrar
	if s.db.Where("user_id = ? AND iana_id = ?", userID, result.RegistrarIanaID).First(&reg).Error == nil {
		return &reg.ID
	}
	reg = models.Registrar{
		UserID:   userID,
		Name:     result.RegistrarName,
		IanaID:   result.RegistrarIanaID,
		URL:      result.RegistrarURL,
		Currency: "USD",
	}
	if s.db.Create(&reg).Error != nil {
		return nil
	}
	return &reg.ID
}

func (s *WhoisService) UpdateDomain(domain *models.Domain) (changed bool, result *WhoisResult, err error) {
	result, err = s.Fetch(domain.Name)
	if err != nil {
		return false, nil, err
	}

	changed = domain.WhoisRaw != "" && domain.WhoisRaw != result.Raw

	nsJSON, _ := json.Marshal(result.NameServers)
	statusJSON, _ := json.Marshal(result.Statuses)

	now := time.Now()
	domain.WhoisRaw = result.Raw
	domain.WhoisFetchedAt = &now
	domain.CreatedDate = result.CreatedDate
	domain.UpdatedDate = result.UpdatedDate
	if result.ExpirationDate != nil {
		domain.ExpirationDate = result.ExpirationDate
	}
	domain.NameServersRaw = string(nsJSON)
	domain.DomainStatus = string(statusJSON)
	domain.DNSSec = result.DNSSec

	if result.RegistrarIanaID != "" && domain.RegistrarID != nil {
		s.db.Model(&models.Registrar{}).
			Where("id = ? AND (iana_id IS NULL OR iana_id = '')", *domain.RegistrarID).
			Update("iana_id", result.RegistrarIanaID)
	}

	return changed, result, nil
}

func ExtractTLD(domainName string) string {
	parts := strings.Split(domainName, ".")
	if len(parts) < 2 {
		return domainName
	}
	return strings.Join(parts[1:], ".")
}
