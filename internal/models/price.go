package models

type Price struct {
	ID              uint       `gorm:"primarykey"`
	RegistrarID     *uint      `gorm:"index"`
	DomainID        *uint      `gorm:"index"`
	TLD             string     // empty = catch-all for registrar defaults
	InitialPerYear float64
	RenewPerYear   float64
	Transfer       float64
	PrivacyPerYear float64
	MiscPerYear    float64
}

func (p *Price) Total() float64 {
	return p.RenewPerYear + p.PrivacyPerYear + p.MiscPerYear
}
