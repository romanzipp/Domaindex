package models

import (
	"encoding/json"
	"time"
)

type Domain struct {
	ID             uint       `gorm:"primarykey"`
	UserID         uint       `gorm:"not null;index;uniqueIndex:idx_user_domain"`
	User           User       `gorm:"foreignKey:UserID"`
	RegistrarID    *uint      `gorm:"index"`
	Registrar      *Registrar `gorm:"foreignKey:RegistrarID"`
	Name           string     `gorm:"not null;uniqueIndex:idx_user_domain"`
	TLD            string     `gorm:"not null"`
	AutoRenewed     bool       `gorm:"default:false"`
	Wishlisted      bool       `gorm:"default:false"`
	TechInfoEnabled bool       `gorm:"default:true"`
	WhoisRaw        string     `gorm:"type:text"`
	WhoisFetchedAt *time.Time
	CreatedDate    *time.Time
	UpdatedDate    *time.Time
	ExpirationDate *time.Time
	NameServersRaw string `gorm:"column:name_servers;type:text"`
	DomainStatus   string `gorm:"column:domain_status;type:text"`
	DNSSec         bool

	// Technical info (DNS, ASN, SSL)
	TechARecords    string     `gorm:"column:tech_a_records;type:text"`
	TechAAAARecords string     `gorm:"column:tech_aaaa_records;type:text"`
	TechASN         string     `gorm:"column:tech_asn"`
	TechASNOrg      string     `gorm:"column:tech_asn_org"`
	TechCountry     string     `gorm:"column:tech_country"`
	TechSSLEnabled  bool       `gorm:"column:tech_ssl_enabled"`
	TechSSLExpiry   *time.Time `gorm:"column:tech_ssl_expiry"`
	TechSSLIssuer   string     `gorm:"column:tech_ssl_issuer"`
	TechFetchedAt   *time.Time `gorm:"column:tech_fetched_at"`

	Tags           []Tag          `gorm:"many2many:domain_tags;"`
	PriceOverride  *Price         `gorm:"foreignKey:DomainID"`
	Notifications  []Notification `gorm:"foreignKey:DomainID"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (d *Domain) NameServers() []string {
	if d.NameServersRaw == "" {
		return nil
	}
	var ns []string
	_ = json.Unmarshal([]byte(d.NameServersRaw), &ns)
	return ns
}

func (d *Domain) Statuses() []string {
	if d.DomainStatus == "" {
		return nil
	}
	var s []string
	_ = json.Unmarshal([]byte(d.DomainStatus), &s)
	return s
}

func (d *Domain) TechAAddresses() []string {
	if d.TechARecords == "" {
		return nil
	}
	var addrs []string
	_ = json.Unmarshal([]byte(d.TechARecords), &addrs)
	return addrs
}

func (d *Domain) TechAAAAAddresses() []string {
	if d.TechAAAARecords == "" {
		return nil
	}
	var addrs []string
	_ = json.Unmarshal([]byte(d.TechAAAARecords), &addrs)
	return addrs
}

func (d *Domain) DaysUntilExpiry() *int {
	if d.ExpirationDate == nil {
		return nil
	}
	days := int(time.Until(*d.ExpirationDate).Hours() / 24)
	return &days
}
