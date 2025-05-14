package model

import (
	"database/sql"
	"errors"
	"time"

	"github.com/sirupsen/logrus"
)

type Notification struct {
	ID      string    `json:"id"`
	UserID  string    `json:"user_id"`
	Message string    `json:"message"`
	SentAt  time.Time `json:"sent_at"`
}

type NotificationRepository interface {
	Create(notification *Notification) error
	GetByID(id string) (*Notification, error)
	GetByUserID(userID string) ([]*Notification, error)
}

type MySQLNotificationRepository struct {
	db     *sql.DB
	logger *logrus.Logger
}

func NewMySQLNotificationRepository(db *sql.DB, logger *logrus.Logger) *MySQLNotificationRepository {
	return &MySQLNotificationRepository{
		db:     db,
		logger: logger,
	}
}

func (r *MySQLNotificationRepository) Create(notification *Notification) error {
	if notification.ID == "" || notification.UserID == "" || notification.Message == "" {
		return errors.New("notification ID, user ID and message are required")
	}

	if notification.SentAt.IsZero() {
		notification.SentAt = time.Now()
	}

	query := `INSERT INTO notifications (id, user_id, message, sent_at) VALUES (?, ?, ?, ?)`
	_, err := r.db.Exec(query, notification.ID, notification.UserID, notification.Message, notification.SentAt)
	if err != nil {
		r.logger.WithError(err).Error("Failed to create notification")
		return err
	}

	return nil
}

func (r *MySQLNotificationRepository) GetByID(id string) (*Notification, error) {
	query := `SELECT id, user_id, message, sent_at FROM notifications WHERE id = ?`
	row := r.db.QueryRow(query, id)

	notification := &Notification{}
	err := row.Scan(&notification.ID, &notification.UserID, &notification.Message, &notification.SentAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("notification not found")
		}
		r.logger.WithError(err).Error("Failed to get notification by ID")
		return nil, err
	}

	return notification, nil
}

func (r *MySQLNotificationRepository) GetByUserID(userID string) ([]*Notification, error) {
	query := `SELECT id, user_id, message, sent_at FROM notifications WHERE user_id = ? ORDER BY sent_at DESC`
	rows, err := r.db.Query(query, userID)
	if err != nil {
		r.logger.WithError(err).Error("Failed to query notifications by user ID")
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {

		}
	}(rows)

	notifications := make([]*Notification, 0)
	for rows.Next() {
		notification := &Notification{}
		err := rows.Scan(&notification.ID, &notification.UserID, &notification.Message, &notification.SentAt)
		if err != nil {
			r.logger.WithError(err).Error("Failed to scan notification row")
			return nil, err
		}
		notifications = append(notifications, notification)
	}

	if err = rows.Err(); err != nil {
		r.logger.WithError(err).Error("Error occurred while iterating over notification rows")
		return nil, err
	}

	return notifications, nil
}
