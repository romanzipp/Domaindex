package handlers

import (
	"net/http"

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

	h.render(w, r, "notifications.html", notifications)
}
