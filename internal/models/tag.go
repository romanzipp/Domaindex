package models

import "time"

var TagColors = []string{
	"gray", "red", "orange", "yellow", "green",
	"teal", "blue", "indigo", "purple", "pink",
}

type Tag struct {
	ID        uint      `gorm:"primarykey"`
	UserID    uint      `gorm:"not null;index"`
	Name      string    `gorm:"not null"`
	Color     string    `gorm:"not null;default:'gray'"`
	Domains   []Domain  `gorm:"many2many:domain_tags;"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
