package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/doniiel/event-ticketing-platform/proto/notification"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMySQLNotificationRepository_SaveNotification(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock database: %v", err)
	}
	defer db.Close()

	repo := NewNotificationRepository(db)

	tests := []struct {
		name          string
		userID        string
		message       string
		mockSetup     func() string
		mockError     error
		expectedError bool
	}{
		{
			name:    "Success - Save Notification",
			userID:  "user123",
			message: "Test message",
			mockSetup: func() string {
				now := time.Now().Format(time.RFC3339)
				mock.ExpectExec("INSERT INTO notifications \\(id, user_id, message, sent_at\\) VALUES \\(\\?, \\?, \\?, \\?\\)").
					WithArgs(sqlmock.AnyArg(), "user123", "Test message", now).
					WillReturnResult(sqlmock.NewResult(1, 1))
				return now
			},
			mockError:     nil,
			expectedError: false,
		},
		{
			name:    "Failure - Database Error",
			userID:  "user123",
			message: "Test message",
			mockSetup: func() string {
				now := time.Now().Format(time.RFC3339)
				mock.ExpectExec("INSERT INTO notifications \\(id, user_id, message, sent_at\\) VALUES \\(\\?, \\?, \\?, \\?\\)").
					WithArgs(sqlmock.AnyArg(), "user123", "Test message", now).
					WillReturnError(errors.New("database error"))
				return now
			},
			mockError:     errors.New("database error"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := tt.mockSetup()
			expectedResp := &notification.Notification{
				UserId:  tt.userID,
				Message: tt.message,
				SentAt:  now,
			}

			resp, err := repo.SaveNotification(tt.userID, tt.message)

			if tt.expectedError {
				assert.Error(t, err)
				assert.EqualError(t, err, "failed to save notification: "+tt.mockError.Error())
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				if assert.NotNil(t, resp) {
					_, uuidErr := uuid.Parse(resp.Id)
					assert.NoError(t, uuidErr, "Id should be a valid UUID")
					assert.Equal(t, expectedResp.UserId, resp.UserId)
					assert.Equal(t, expectedResp.Message, resp.Message)
					assert.Equal(t, expectedResp.SentAt, resp.SentAt)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func TestMySQLNotificationRepository_GetNotificationsByUserID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock database: %v", err)
	}
	defer db.Close()

	repo := NewNotificationRepository(db)

	tests := []struct {
		name          string
		userID        string
		mockSetup     func() ([]*notification.Notification, error)
		expectedError bool
	}{
		{
			name:   "Success - Get Notifications",
			userID: "user123",
			mockSetup: func() ([]*notification.Notification, error) {
				id1, id2 := uuid.New().String(), uuid.New().String()
				now1, now2 := time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339)
				rows := sqlmock.NewRows([]string{"id", "user_id", "message", "sent_at"}).
					AddRow(id1, "user123", "Message 1", now1).
					AddRow(id2, "user123", "Message 2", now2)
				mock.ExpectQuery("SELECT id, user_id, message, sent_at FROM notifications WHERE user_id = \\? ORDER BY sent_at DESC").
					WithArgs("user123").
					WillReturnRows(rows)
				return []*notification.Notification{
					{Id: id1, UserId: "user123", Message: "Message 1", SentAt: now1},
					{Id: id2, UserId: "user123", Message: "Message 2", SentAt: now2},
				}, nil
			},
			expectedError: false,
		},
		{
			name:   "Failure - Query Error",
			userID: "user123",
			mockSetup: func() ([]*notification.Notification, error) {
				mock.ExpectQuery("SELECT id, user_id, message, sent_at FROM notifications WHERE user_id = \\? ORDER BY sent_at DESC").
					WithArgs("user123").
					WillReturnError(errors.New("query error"))
				return nil, errors.New("query error")
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			expectedResp, setupErr := tt.mockSetup()
			if setupErr != nil {
				assert.Error(t, setupErr)
			}

			resp, err := repo.GetNotificationsByUserID(tt.userID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.EqualError(t, err, "failed to query notifications: "+setupErr.Error())
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, len(expectedResp), len(resp))
				for i := range resp {
					assert.Equal(t, expectedResp[i].Id, resp[i].Id)
					assert.Equal(t, expectedResp[i].UserId, resp[i].UserId)
					assert.Equal(t, expectedResp[i].Message, resp[i].Message)
					assert.Equal(t, expectedResp[i].SentAt, resp[i].SentAt)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
