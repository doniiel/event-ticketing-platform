package repository

import (
	"database/sql"
	"fmt"
	"time"

	notificationpb "github.com/doniiel/event-ticketing-platform/proto/notification"
	"github.com/google/uuid"
)

type NotificationRepository interface {
	SaveNotification(userID, message string) (*notificationpb.Notification, error)
	GetNotificationsByUserID(userID string) ([]*notificationpb.Notification, error)
}

type MySQLNotificationRepository struct {
	db *sql.DB
}

func NewNotificationRepository(db *sql.DB) NotificationRepository {
	return &MySQLNotificationRepository{
		db: db,
	}
}

func (r *MySQLNotificationRepository) SaveNotification(userID, message string) (*notificationpb.Notification, error) {
	id := uuid.New().String()
	now := time.Now().Format(time.RFC3339)

	query := `INSERT INTO notifications (id, user_id, message, sent_at) VALUES (?, ?, ?, ?)`
	_, err := r.db.Exec(query, id, userID, message, now)
	if err != nil {
		return nil, fmt.Errorf("failed to save notification: %w", err)
	}

	return &notificationpb.Notification{
		Id:      id,
		UserId:  userID,
		Message: message,
		SentAt:  now,
	}, nil
}

func (r *MySQLNotificationRepository) GetNotificationsByUserID(userID string) ([]*notificationpb.Notification, error) {
	query := `SELECT id, user_id, message, sent_at FROM notifications WHERE user_id = ? ORDER BY sent_at DESC`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query notifications: %w", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)

	var notifications []*notificationpb.Notification
	for rows.Next() {
		notification := &notificationpb.Notification{}
		err := rows.Scan(&notification.Id, &notification.UserId, &notification.Message, &notification.SentAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification row: %w", err)
		}
		notifications = append(notifications, notification)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	return notifications, nil
}
