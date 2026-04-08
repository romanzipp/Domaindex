package models

import "time"

const (
	NotificationTypeExpiry7d        = "expiry_7d"
	NotificationTypeExpiry30d       = "expiry_30d"
	NotificationTypeExpiry24h       = "expiry_24h"
	NotificationTypeWhoisChanged    = "whois_changed"
)

type Notification struct {
	ID        uint       `gorm:"primarykey"`
	UserID    uint       `gorm:"not null;index"`
	User      User       `gorm:"foreignKey:UserID"`
	DomainID  uint       `gorm:"not null;index"`
	Domain    Domain     `gorm:"foreignKey:DomainID"`
	Type      string     `gorm:"not null"`
	Message   string     `gorm:"type:text"`
	SentAt    *time.Time
	ReadAt    *time.Time
	CreatedAt time.Time
}
