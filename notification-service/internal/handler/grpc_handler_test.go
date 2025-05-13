package handler

import (
	"context"
	"errors"
	"testing"

	notificationpb "github.com/doniiel/event-ticketing-platform/proto/notification"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockNotificationRepository struct {
	mock.Mock
}

func (m *MockNotificationRepository) SaveNotification(userID, message string) (*notificationpb.Notification, error) {
	args := m.Called(userID, message)
	return args.Get(0).(*notificationpb.Notification), args.Error(1)
}

func (m *MockNotificationRepository) GetNotificationsByUserID(userID string) ([]*notificationpb.Notification, error) {
	args := m.Called(userID)
	return args.Get(0).([]*notificationpb.Notification), args.Error(1)
}

func TestNotificationHandler_SendNotification(t *testing.T) {
	mockRepo := new(MockNotificationRepository)
	handler := NewNotificationHandler(mockRepo)

	tests := []struct {
		name          string
		userID        string
		message       string
		mockResponse  *notificationpb.Notification
		mockError     error
		expectedError bool
	}{
		{
			name:    "Success - Send Notification",
			userID:  "user123",
			message: "Test message",
			mockResponse: &notificationpb.Notification{
				UserId:  "user123",
				Message: "Test message",
			},
			mockError:     nil,
			expectedError: false,
		},
		{
			name:          "Failure - Save Notification Error",
			userID:        "user123",
			message:       "Test message",
			mockResponse:  nil,
			mockError:     errors.New("failed to save notification"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.On("SaveNotification", tt.userID, tt.message).Return(tt.mockResponse, tt.mockError).Once()

			resp, err := handler.SendNotification(context.Background(), &notificationpb.SendNotificationRequest{
				UserId:  tt.userID,
				Message: tt.message,
			})

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, resp)
				assert.EqualError(t, err, tt.mockError.Error())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.mockResponse, resp.Notification)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestNotificationHandler_GetNotifications(t *testing.T) {
	mockRepo := new(MockNotificationRepository)
	handler := NewNotificationHandler(mockRepo)

	tests := []struct {
		name          string
		userID        string
		mockResponse  []*notificationpb.Notification
		mockError     error
		expectedError bool
	}{
		{
			name:   "Success - Get Notifications",
			userID: "user123",
			mockResponse: []*notificationpb.Notification{
				{UserId: "user123", Message: "Message 1"},
				{UserId: "user123", Message: "Message 2"},
			},
			mockError:     nil,
			expectedError: false,
		},
		{
			name:          "Failure - Get Notifications Error",
			userID:        "user123",
			mockResponse:  nil,
			mockError:     errors.New("failed to retrieve notifications"),
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo.On("GetNotificationsByUserID", tt.userID).Return(tt.mockResponse, tt.mockError).Once()

			resp, err := handler.GetNotifications(context.Background(), &notificationpb.GetNotificationsRequest{
				UserId: tt.userID,
			})

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, resp)
				assert.EqualError(t, err, tt.mockError.Error())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tt.mockResponse, resp.Notifications)
			}

			mockRepo.AssertExpectations(t)
		})
	}
}
