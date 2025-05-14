package handler

import (
	"context"
	"log"

	"github.com/doniiel/event-ticketing-platform/notification-service/internal/repository"
	notificationpb "github.com/doniiel/event-ticketing-platform/proto/notification"
)

type NotificationHandler struct {
	notificationpb.UnimplementedNotificationServiceServer
	repo repository.NotificationRepository
}

func NewNotificationHandler(repo repository.NotificationRepository) *NotificationHandler {
	return &NotificationHandler{
		repo: repo,
	}
}

// @Summary Send a notification
// @Description Sends a notification to a user
// @Tags Notifications
// @Accept json
// @Produce json
// @Param userID path string true "User ID"
// @Param message body string true "Notification message"
// @Success 200 {object} map[string]string "message: Notification sent"
// @Failure 400 {object} map[string]string "error: Invalid request"
// @Router /notifications/{userID} [post]
func (h *NotificationHandler) SendNotification(ctx context.Context, req *notificationpb.SendNotificationRequest) (*notificationpb.SendNotificationResponse, error) {
	notification, err := h.repo.SaveNotification(req.UserId, req.Message)
	if err != nil {
		log.Printf("Failed to save notification: %v", err)
		return nil, err
	}

	log.Printf("Notification sent to user %s: %s", req.UserId, req.Message)
	return &notificationpb.SendNotificationResponse{
		Notification: notification,
	}, nil
}

func (h *NotificationHandler) GetNotifications(ctx context.Context, req *notificationpb.GetNotificationsRequest) (*notificationpb.GetNotificationsResponse, error) {
	notifications, err := h.repo.GetNotificationsByUserID(req.UserId)
	if err != nil {
		log.Printf("Failed to get notifications for user %s: %v", req.UserId, err)
		return nil, err
	}

	return &notificationpb.GetNotificationsResponse{
		Notifications: notifications,
	}, nil
}
