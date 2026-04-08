package models

import "time"

type Registrar struct {
	ID        uint      `gorm:"primarykey"`
	UserID    uint      `gorm:"not null;index"`
	User      User      `gorm:"foreignKey:UserID"`
	Name      string    `gorm:"not null"`
	URL       string
	Notes     string
	Currency  string    `gorm:"default:'USD'"`
	Prices    []Price   `gorm:"foreignKey:RegistrarID"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
