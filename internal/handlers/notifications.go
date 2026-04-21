package handlers

import (
	"net/http"
	"time"

	"github.com/romanzipp/domaindex/internal/middleware"
	"github.com/romanzipp/domaindex/internal/models"
	"github.com/romanzipp/domaindex/internal/services"
)

type NotificationsHandler struct {
	*Base
	notifSvc *services.NotificationService
}

func NewNotificationsHandler(base *Base, notifSvc *services.NotificationService) *NotificationsHandler {
	return &NotificationsHandler{Base: base, notifSvc: notifSvc}
}

func (h *NotificationsHandler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	var notifications []models.Notification
	h.db.Preload("Domain").
		Where("user_id = ?", user.ID).
		Order("created_at desc").
		Limit(200).
		Find(&notifications)

	// Render first so the user sees unread highlighted on this visit.
	// After the response is written, mark them read so F5 shows them as read.
	h.render(w, r, "notifications.html", notifications)

	now := time.Now()
	h.db.Model(&models.Notification{}).
		Where("user_id = ? AND read_at IS NULL", user.ID).
		Update("read_at", &now)
}

func (h *NotificationsHandler) SendTest(w http.ResponseWriter, r *http.Request) {
	user := middleware.UserFromContext(r.Context())

	var domain models.Domain
	if err := h.db.Where("user_id = ?", user.ID).First(&domain).Error; err != nil {
		h.flashError(w, r, "No domains found to send a test notification for")
		http.Redirect(w, r, "/notifications", http.StatusSeeOther)
		return
	}

	if err := h.notifSvc.Send(user.ID, domain.ID, models.NotificationTypeTest, "Test notification from Domaindex"); err != nil {
		h.flashError(w, r, "Failed to send test notification: "+err.Error())
	} else {
		h.flashSuccess(w, r, "Test notification sent")
	}

	http.Redirect(w, r, "/notifications", http.StatusSeeOther)
}
