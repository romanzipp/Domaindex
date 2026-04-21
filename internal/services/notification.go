package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/romanzipp/domaindex/internal/models"
	"gorm.io/gorm"
)

type NotificationService struct {
	db         *gorm.DB
	appriseURL string
	appriseKey string
	appriseTag string
}

func NewNotificationService(db *gorm.DB, appriseURL, appriseKey, appriseTag string) *NotificationService {
	return &NotificationService{db: db, appriseURL: appriseURL, appriseKey: appriseKey, appriseTag: appriseTag}
}

func (s *NotificationService) Send(userID, domainID uint, notifType, message string) error {
	if s.alreadySent(domainID, notifType) {
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

	if s.appriseURL != "" {
		if err := s.sendToApprise(message); err != nil {
			log.Printf("apprise send failed: %v", err)
		}
	}

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

func (s *NotificationService) sendToApprise(message string) error {
	payload := map[string]string{
		"title": "Domain Manager",
		"body":  message,
		"type":  "info",
	}
	if s.appriseTag != "" {
		payload["tag"] = s.appriseTag
	}

	body, _ := json.Marshal(payload)

	url := s.appriseURL + "/notify/"
	if s.appriseKey != "" {
		url = s.appriseURL + "/notify/" + s.appriseKey
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("apprise returned status %d", resp.StatusCode)
	}

	return nil
}
