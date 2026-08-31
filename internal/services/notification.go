package services

import (
	"log"
	"time"

	"github.com/containrrr/shoutrrr"
	"github.com/containrrr/shoutrrr/pkg/router"
	"github.com/containrrr/shoutrrr/pkg/types"
	"github.com/romanzipp/domaindex/internal/models"
	"gorm.io/gorm"
)

type NotificationService struct {
	db     *gorm.DB
	sender *router.ServiceRouter
}

func NewNotificationService(db *gorm.DB, urls []string) *NotificationService {
	s := &NotificationService{db: db}

	if len(urls) > 0 {
		sender, err := shoutrrr.CreateSender(urls...)
		if err != nil {
			log.Printf("notification urls invalid: %v", err)
		} else {
			s.sender = sender
		}
	}

	return s
}

func (s *NotificationService) Send(userID, domainID uint, notifType, message string) error {
	if notifType != models.NotificationTypeTest && s.alreadySent(domainID, notifType) {
		return nil
	}

	n := &models.Notification{
		UserID:   userID,
		DomainID: domainID,
		Type:     notifType,
		Message:  message,
	}

	if err := s.db.Create(n).Error; err != nil {
		return err
	}

	now := time.Now()
	if err := s.db.Model(n).Update("sent_at", &now).Error; err != nil {
		return err
	}

	s.notify(message)

	return nil
}

func (s *NotificationService) alreadySent(domainID uint, notifType string) bool {
	var count int64
	today := time.Now().Truncate(24 * time.Hour)
	s.db.Model(&models.Notification{}).
		Where("domain_id = ? AND type = ? AND created_at >= ?", domainID, notifType, today).
		Count(&count)
	return count > 0
}

func (s *NotificationService) notify(message string) {
	if s.sender == nil {
		return
	}

	for _, err := range s.sender.Send(message, &types.Params{"title": "Domaindex"}) {
		if err != nil {
			log.Printf("notification send failed: %v", err)
		}
	}
}
