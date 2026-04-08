package handlers

import (
	"net/http"
	"time"

	"github.com/romanzipp/domain-manager/internal/middleware"
	"github.com/romanzipp/domain-manager/internal/models"
)

type NotificationsHandler struct {
	*Base
}

func NewNotificationsHandler(base *Base) *NotificationsHandler {
	return &NotificationsHandler{Base: base}
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
