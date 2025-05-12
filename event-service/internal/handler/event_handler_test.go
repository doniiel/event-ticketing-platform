package handler

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/doniiel/event-ticketing-platform/event-service/internal/model"
	eventpb "github.com/doniiel/event-ticketing-platform/proto/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockEventRepository struct {
	events map[string]*model.Event
}

func (m *mockEventRepository) Create(ctx context.Context, event *model.Event) (*model.Event, error) {
	m.events[event.ID] = event
	return event, nil
}

func (m *mockEventRepository) GetByID(ctx context.Context, id string) (*model.Event, error) {
	event, exists := m.events[id]
	if !exists {
		return nil, sql.ErrNoRows
	}
	return event, nil
}

func (m *mockEventRepository) Update(ctx context.Context, event *model.Event) (*model.Event, error) {
	if _, exists := m.events[event.ID]; !exists {
		return nil, sql.ErrNoRows
	}
	m.events[event.ID] = event
	return event, nil
}

func (m *mockEventRepository) Delete(ctx context.Context, id string) error {
	if _, exists := m.events[id]; !exists {
		return sql.ErrNoRows
	}
	delete(m.events, id)
	return nil
}

func (m *mockEventRepository) List(ctx context.Context, page, pageSize int32) ([]*model.Event, int32, error) {
	var events []*model.Event
	for _, event := range m.events {
		events = append(events, event)
	}
	return events, int32(len(events)), nil
}

func (m *mockEventRepository) CheckAvailability(ctx context.Context, eventID string, quantity int32) (bool, error) {
	event, exists := m.events[eventID]
	if !exists {
		return false, nil
	}
	return event.TicketStock >= quantity, nil
}

func (m *mockEventRepository) UpdateTicketStock(ctx context.Context, eventID string, quantity int32) error {
	event, exists := m.events[eventID]
	if !exists {
		return errors.New("event not found")
	}
	if event.TicketStock < quantity {
		return errors.New("not enough tickets")
	}
	event.TicketStock -= quantity
	return nil
}

func TestEventHandler_CreateEvent(t *testing.T) {
	repo := &mockEventRepository{events: make(map[string]*model.Event)}
	handler := NewEventHandler(repo)

	tests := []struct {
		name    string
		req     *eventpb.CreateEventRequest
		wantErr codes.Code
	}{
		{
			name: "valid request",
			req: &eventpb.CreateEventRequest{
				Name:        "Test Concert",
				Date:        "2025-06-01T19:00:00Z",
				Location:    "Test Arena",
				TicketStock: 100,
			},
			wantErr: codes.OK,
		},
		{
			name: "missing name",
			req: &eventpb.CreateEventRequest{
				Date:        "2025-06-01T19:00:00Z",
				Location:    "Test Arena",
				TicketStock: 100,
			},
			wantErr: codes.InvalidArgument,
		},
		{
			name: "invalid date",
			req: &eventpb.CreateEventRequest{
				Name:        "Test Concert",
				Date:        "invalid-date",
				Location:    "Test Arena",
				TicketStock: 100,
			},
			wantErr: codes.InvalidArgument,
		},
		{
			name: "invalid ticket stock",
			req: &eventpb.CreateEventRequest{
				Name:        "Test Concert",
				Date:        "2025-06-01T19:00:00Z",
				Location:    "Test Arena",
				TicketStock: 0,
			},
			wantErr: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := handler.CreateEvent(context.Background(), tt.req)
			if tt.wantErr == codes.OK {
				if err != nil {
					t.Errorf("CreateEvent() error = %v, want nil", err)
					return
				}
				if resp.Event.Name != tt.req.Name {
					t.Errorf("CreateEvent() name = %s, want %s", resp.Event.Name, tt.req.Name)
				}
			} else {
				if status.Code(err) != tt.wantErr {
					t.Errorf("CreateEvent() error code = %v, want %v", status.Code(err), tt.wantErr)
				}
			}
		})
	}
}

func TestEventHandler_GetEvent(t *testing.T) {
	repo := &mockEventRepository{events: make(map[string]*model.Event)}
	handler := NewEventHandler(repo)

	event, _ := model.NewEvent("Test Concert", "2025-06-01T19:00:00Z", "Test Arena", 100)
	repo.events[event.ID] = event

	tests := []struct {
		name    string
		req     *eventpb.GetEventRequest
		wantErr codes.Code
	}{
		{
			name:    "valid request",
			req:     &eventpb.GetEventRequest{Id: event.ID},
			wantErr: codes.OK,
		},
		{
			name:    "missing ID",
			req:     &eventpb.GetEventRequest{Id: ""},
			wantErr: codes.InvalidArgument,
		},
		{
			name:    "not found",
			req:     &eventpb.GetEventRequest{Id: "non-existent"},
			wantErr: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := handler.GetEvent(context.Background(), tt.req)
			if tt.wantErr == codes.OK {
				if err != nil {
					t.Errorf("GetEvent() error = %v, want nil", err)
					return
				}
				if resp.Event.Id != tt.req.Id {
					t.Errorf("GetEvent() id = %s, want %s", resp.Event.Id, tt.req.Id)
				}
			} else {
				if status.Code(err) != tt.wantErr {
					t.Errorf("GetEvent() error code = %v, want %v", status.Code(err), tt.wantErr)
				}
			}
		})
	}
}

func TestEventHandler_ListEvents(t *testing.T) {
	repo := &mockEventRepository{events: make(map[string]*model.Event)}
	handler := NewEventHandler(repo)

	event1, _ := model.NewEvent("Concert 1", "2025-06-01T19:00:00Z", "Arena 1", 100)
	event2, _ := model.NewEvent("Concert 2", "2025-06-02T19:00:00Z", "Arena 2", 200)
	repo.events[event1.ID] = event1
	repo.events[event2.ID] = event2

	tests := []struct {
		name    string
		req     *eventpb.ListEventsRequest
		wantLen int
		wantErr codes.Code
	}{
		{
			name:    "valid request",
			req:     &eventpb.ListEventsRequest{Page: 1, PageSize: 10},
			wantLen: 2,
			wantErr: codes.OK,
		},
		{
			name:    "invalid page",
			req:     &eventpb.ListEventsRequest{Page: 0, PageSize: 10},
			wantLen: 2,
			wantErr: codes.OK,
		},
		{
			name:    "invalid page size",
			req:     &eventpb.ListEventsRequest{Page: 1, PageSize: 0},
			wantLen: 2,
			wantErr: codes.OK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := handler.ListEvents(context.Background(), tt.req)
			if tt.wantErr == codes.OK {
				if err != nil {
					t.Errorf("ListEvents() error = %v, want nil", err)
					return
				}
				if len(resp.Events) != tt.wantLen {
					t.Errorf("ListEvents() events len = %d, want %d", len(resp.Events), tt.wantLen)
				}
				if int(resp.Total) != tt.wantLen {
					t.Errorf("ListEvents() total = %d, want %d", resp.Total, tt.wantLen)
				}
			} else {
				if status.Code(err) != tt.wantErr {
					t.Errorf("ListEvents() error code = %v, want %v", status.Code(err), tt.wantErr)
				}
			}
		})
	}
}

func TestEventHandler_UpdateEvent(t *testing.T) {
	repo := &mockEventRepository{events: make(map[string]*model.Event)}
	handler := NewEventHandler(repo)

	event, _ := model.NewEvent("Test Concert", "2025-06-01T19:00:00Z", "Test Arena", 100)
	repo.events[event.ID] = event

	tests := []struct {
		name    string
		req     *eventpb.UpdateEventRequest
		wantErr codes.Code
	}{
		{
			name: "valid request",
			req: &eventpb.UpdateEventRequest{
				Id:          event.ID,
				Name:        "Updated Concert",
				Date:        "2025-06-02T19:00:00Z",
				Location:    "Updated Arena",
				TicketStock: 200,
			},
			wantErr: codes.OK,
		},
		{
			name: "missing ID",
			req: &eventpb.UpdateEventRequest{
				Name:        "Updated Concert",
				Date:        "2025-06-02T19:00:00Z",
				Location:    "Updated Arena",
				TicketStock: 200,
			},
			wantErr: codes.InvalidArgument,
		},
		{
			name: "not found",
			req: &eventpb.UpdateEventRequest{
				Id:          "non-existent",
				Name:        "Updated Concert",
				Date:        "2025-06-02T19:00:00Z",
				Location:    "Updated Arena",
				TicketStock: 200,
			},
			wantErr: codes.NotFound,
		},
		{
			name: "invalid date",
			req: &eventpb.UpdateEventRequest{
				Id:          event.ID,
				Name:        "Updated Concert",
				Date:        "invalid-date",
				Location:    "Updated Arena",
				TicketStock: 200,
			},
			wantErr: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := handler.UpdateEvent(context.Background(), tt.req)
			if tt.wantErr == codes.OK {
				if err != nil {
					t.Errorf("UpdateEvent() error = %v, want nil", err)
					return
				}
				if resp.Event.Name != tt.req.Name {
					t.Errorf("UpdateEvent() name = %s, want %s", resp.Event.Name, tt.req.Name)
				}
			} else {
				if status.Code(err) != tt.wantErr {
					t.Errorf("UpdateEvent() error code = %v, want %v", status.Code(err), tt.wantErr)
				}
			}
		})
	}
}

func TestEventHandler_DeleteEvent(t *testing.T) {
	repo := &mockEventRepository{events: make(map[string]*model.Event)}
	handler := NewEventHandler(repo)

	event, _ := model.NewEvent("Test Concert", "2025-06-01T19:00:00Z", "Test Arena", 100)
	repo.events[event.ID] = event

	tests := []struct {
		name    string
		req     *eventpb.DeleteEventRequest
		wantErr codes.Code
	}{
		{
			name:    "valid request",
			req:     &eventpb.DeleteEventRequest{Id: event.ID},
			wantErr: codes.OK,
		},
		{
			name:    "missing ID",
			req:     &eventpb.DeleteEventRequest{Id: ""},
			wantErr: codes.InvalidArgument,
		},
		{
			name:    "not found",
			req:     &eventpb.DeleteEventRequest{Id: "non-existent"},
			wantErr: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handler.DeleteEvent(context.Background(), tt.req)
			if tt.wantErr == codes.OK {
				if err != nil {
					t.Errorf("DeleteEvent() error = %v, want nil", err)
					return
				}
				if _, exists := repo.events[tt.req.Id]; exists {
					t.Errorf("DeleteEvent() event still exists")
				}
			} else {
				if status.Code(err) != tt.wantErr {
					t.Errorf("DeleteEvent() error code = %v, want %v", status.Code(err), tt.wantErr)
				}
			}
		})
	}
}

func TestEventHandler_CheckAvailability(t *testing.T) {
	repo := &mockEventRepository{events: make(map[string]*model.Event)}
	handler := NewEventHandler(repo)

	event, _ := model.NewEvent("Test Concert", "2025-06-01T19:00:00Z", "Test Arena", 100)
	repo.events[event.ID] = event

	tests := []struct {
		name    string
		req     *eventpb.CheckAvailabilityRequest
		want    bool
		wantErr codes.Code
	}{
		{
			name: "valid request - available",
			req: &eventpb.CheckAvailabilityRequest{
				EventId:  event.ID,
				Quantity: 50,
			},
			want:    true,
			wantErr: codes.OK,
		},
		{
			name: "valid request - not available",
			req: &eventpb.CheckAvailabilityRequest{
				EventId:  event.ID,
				Quantity: 150,
			},
			want:    false,
			wantErr: codes.OK,
		},
		{
			name: "missing event ID",
			req: &eventpb.CheckAvailabilityRequest{
				EventId:  "",
				Quantity: 50,
			},
			wantErr: codes.InvalidArgument,
		},
		{
			name: "invalid quantity",
			req: &eventpb.CheckAvailabilityRequest{
				EventId:  event.ID,
				Quantity: 0,
			},
			wantErr: codes.InvalidArgument,
		},
		{
			name: "event not found",
			req: &eventpb.CheckAvailabilityRequest{
				EventId:  "non-existent",
				Quantity: 50,
			},
			want:    false,
			wantErr: codes.OK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := handler.CheckAvailability(context.Background(), tt.req)
			if tt.wantErr == codes.OK {
				if err != nil {
					t.Errorf("CheckAvailability() error = %v, want nil", err)
					return
				}
				if resp.Available != tt.want {
					t.Errorf("CheckAvailability() available = %v, want %v", resp.Available, tt.want)
				}
			} else {
				if status.Code(err) != tt.wantErr {
					t.Errorf("CheckAvailability() error code = %v, want %v", status.Code(err), tt.wantErr)
				}
			}
		})
	}
}

// Benchmarking tests
func BenchmarkEventHandler_CreateEvent(b *testing.B) {
	repo := &mockEventRepository{events: make(map[string]*model.Event)}
	handler := NewEventHandler(repo)
	req := &eventpb.CreateEventRequest{
		Name:        "Test Concert",
		Date:        "2025-06-01T19:00:00Z",
		Location:    "Test Arena",
		TicketStock: 100,
	}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := handler.CreateEvent(ctx, req)
		if err != nil {
			b.Fatalf("CreateEvent() error = %v", err)
		}
	}
}

func BenchmarkEventHandler_GetEvent(b *testing.B) {
	repo := &mockEventRepository{events: make(map[string]*model.Event)}
	handler := NewEventHandler(repo)
	event, _ := model.NewEvent("Test Concert", "2025-06-01T19:00:00Z", "Test Arena", 100)
	repo.events[event.ID] = event
	req := &eventpb.GetEventRequest{Id: event.ID}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := handler.GetEvent(ctx, req)
		if err != nil {
			b.Fatalf("GetEvent() error = %v", err)
		}
	}
}
