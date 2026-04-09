package models

import (
	"encoding/json"
	"time"
)

type Domain struct {
	ID             uint       `gorm:"primarykey"`
	UserID         uint       `gorm:"not null;index"`
	User           User       `gorm:"foreignKey:UserID"`
	RegistrarID    *uint      `gorm:"index"`
	Registrar      *Registrar `gorm:"foreignKey:RegistrarID"`
	Name           string     `gorm:"not null"`
	TLD            string     `gorm:"not null"`
	AutoRenewed    bool       `gorm:"default:false"`
	Wishlisted     bool       `gorm:"default:false"`
	WhoisRaw       string     `gorm:"type:text"`
	WhoisFetchedAt *time.Time
	CreatedDate    *time.Time
	UpdatedDate    *time.Time
	ExpirationDate *time.Time
	RegistrarName  string
	NameServersRaw string `gorm:"column:name_servers;type:text"`
	DomainStatus   string `gorm:"column:domain_status;type:text"`
	DNSSec         bool
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

func (d *Domain) DaysUntilExpiry() *int {
	if d.ExpirationDate == nil {
		return nil
	}
	days := int(time.Until(*d.ExpirationDate).Hours() / 24)
	return &days
}
