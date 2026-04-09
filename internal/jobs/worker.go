package jobs

import (
	"fmt"
	"log"
	"time"

	"github.com/romanzipp/domaindex/internal/models"
	"github.com/romanzipp/domaindex/internal/services"
	"gorm.io/gorm"
)

type Worker struct {
	db           *gorm.DB
	whois        *services.WhoisService
	notification *services.NotificationService
	interval     time.Duration
	trigger      chan struct{}
}

func NewWorker(db *gorm.DB, whois *services.WhoisService, notification *services.NotificationService, interval time.Duration) *Worker {
	return &Worker{
		db:           db,
		whois:        whois,
		notification: notification,
		interval:     interval,
		trigger:      make(chan struct{}, 1),
	}
}

func (w *Worker) Start() {
	go w.run()
}

// RunNow triggers an immediate worker cycle. Non-blocking: if a run is already queued, this is a no-op.
func (w *Worker) RunNow() {
	select {
	case w.trigger <- struct{}{}:
	default:
	}
}

func (w *Worker) run() {
	w.tick()
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			w.tick()
		case <-w.trigger:
			w.tick()
		}
	}
}

func (w *Worker) tick() {
	if err := w.refreshWhois(); err != nil {
		log.Printf("whois refresh error: %v", err)
	}
	if err := w.checkExpiries(); err != nil {
		log.Printf("expiry check error: %v", err)
	}
}

func (w *Worker) refreshWhois() error {
	var domains []models.Domain
	if err := w.db.Find(&domains).Error; err != nil {
		return err
	}

	for i := range domains {
		d := &domains[i]
		changed, result, err := w.whois.UpdateDomain(d)
		if err != nil {
			log.Printf("whois fetch failed for %s: %v", d.Name, err)
			continue
		}
		if d.RegistrarID == nil && result != nil {
			d.RegistrarID = w.whois.ResolveRegistrar(result, d.UserID)
		}
		if err := w.db.Save(d).Error; err != nil {
			log.Printf("save domain %s: %v", d.Name, err)
			continue
		}
		if changed {
			msg := fmt.Sprintf("WHOIS data changed for %s", d.Name)
			if err := w.notification.Send(d.UserID, d.ID, models.NotificationTypeWhoisChanged, msg); err != nil {
				log.Printf("send notification for %s: %v", d.Name, err)
			}
		}
	}

	return nil
}

func (w *Worker) checkExpiries() error {
	var domains []models.Domain
	if err := w.db.Where("expiration_date IS NOT NULL").Find(&domains).Error; err != nil {
		return err
	}

	for i := range domains {
		d := &domains[i]
		if d.ExpirationDate == nil {
			continue
		}

		days := d.DaysUntilExpiry()
		if days == nil {
			continue
		}

		if d.Wishlisted {
			w.checkWishlistExpiry(d, *days)
		} else {
			w.checkDomainExpiry(d, *days)
		}
	}

	return nil
}

func (w *Worker) checkDomainExpiry(d *models.Domain, days int) {
	if days <= 7 {
		msg := fmt.Sprintf("Domain %s expires in %d days", d.Name, days)
		if err := w.notification.Send(d.UserID, d.ID, models.NotificationTypeExpiry7d, msg); err != nil {
			log.Printf("send expiry notification for %s: %v", d.Name, err)
		}
	}
}

func (w *Worker) checkWishlistExpiry(d *models.Domain, days int) {
	var notifType, msg string

	switch {
	case days <= 1:
		notifType = models.NotificationTypeExpiry24h
		msg = fmt.Sprintf("Wishlisted domain %s expires in less than 24 hours", d.Name)
	case days <= 7:
		notifType = models.NotificationTypeExpiry7d
		msg = fmt.Sprintf("Wishlisted domain %s expires in %d days", d.Name, days)
	case days <= 30:
		notifType = models.NotificationTypeExpiry30d
		msg = fmt.Sprintf("Wishlisted domain %s expires in %d days", d.Name, days)
	default:
		return
	}

	if err := w.notification.Send(d.UserID, d.ID, notifType, msg); err != nil {
		log.Printf("send wishlist notification for %s: %v", d.Name, err)
	}
}
