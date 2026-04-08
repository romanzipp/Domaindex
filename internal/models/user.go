package models

import "time"

type User struct {
	ID              uint      `gorm:"primarykey"`
	Username        string    `gorm:"uniqueIndex;not null"`
	Email           string    `gorm:"uniqueIndex;not null"`
	PasswordHash    string    `gorm:"not null"`
	DefaultCurrency string    `gorm:"default:'USD'"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
