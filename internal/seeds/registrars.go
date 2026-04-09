package seeds

import (
	"fmt"

	"github.com/romanzipp/domain-manager/internal/models"
	"gorm.io/gorm"
)

type registrarSeed struct {
	Name   string
	IanaID string
	URL    string
}

var defaultRegistrars = []registrarSeed{
	{Name: "GoDaddy", IanaID: "146", URL: "https://www.godaddy.com"},
	{Name: "Namecheap", IanaID: "1068", URL: "https://www.namecheap.com"},
	{Name: "Network Solutions", IanaID: "2", URL: "https://www.networksolutions.com"},
	{Name: "Register.com", IanaID: "9", URL: "https://www.register.com"},
	{Name: "Tucows Domains", IanaID: "69", URL: "https://opensrs.com"},
	{Name: "Gandi", IanaID: "81", URL: "https://www.gandi.net"},
	{Name: "IONOS", IanaID: "83", URL: "https://www.ionos.com"},
	{Name: "Enom", IanaID: "48", URL: "https://www.enom.com"},
	{Name: "Name.com", IanaID: "625", URL: "https://www.name.com"},
	{Name: "Dynadot", IanaID: "472", URL: "https://www.dynadot.com"},
	{Name: "Hover", IanaID: "955", URL: "https://www.hover.com"},
	{Name: "OVH", IanaID: "433", URL: "https://www.ovhcloud.com"},
	{Name: "Squarespace Domains", IanaID: "895", URL: "https://domains.squarespace.com"},
	{Name: "Cloudflare", IanaID: "1910", URL: "https://www.cloudflare.com/products/registrar/"},
	{Name: "Porkbun", IanaID: "1861", URL: "https://porkbun.com"},
	{Name: "NameSilo", IanaID: "1479", URL: "https://www.namesilo.com"},
	{Name: "Hostinger", IanaID: "1636", URL: "https://www.hostinger.com"},
	{Name: "Hetzner", IanaID: "1373", URL: "https://www.hetzner.com"},
	{Name: "INWX", IanaID: "1408", URL: "https://www.inwx.com"},
	{Name: "DreamHost", IanaID: "431", URL: "https://www.dreamhost.com"},
	{Name: "Domain.com", IanaID: "886", URL: "https://www.domain.com"},
	{Name: "Bluehost", IanaID: "", URL: "https://www.bluehost.com"},
	{Name: "Spaceship", IanaID: "3920", URL: "https://www.spaceship.com"},
	{Name: "Sav.com", IanaID: "3764", URL: "https://www.sav.com"},
	{Name: "Hostgator", IanaID: "", URL: "https://www.hostgator.com"},
}

func seedRegistrars(db *gorm.DB, userID uint) error {
	key := fmt.Sprintf("registrars_%d", userID)
	if hasRun(db, key) {
		return nil
	}

	for _, r := range defaultRegistrars {
		db.Create(&models.Registrar{
			UserID: userID,
			Name:   r.Name,
			IanaID: r.IanaID,
			URL:    r.URL,
		})
	}

	markRun(db, key)
	return nil
}
