package handler

import (
	"context"
	"time"

	"github.com/doniiel/event-ticketing-platform/event-service/internal/model"
	"github.com/doniiel/event-ticketing-platform/event-service/internal/repository"
	eventpb "github.com/doniiel/event-ticketing-platform/proto/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type EventHandler struct {
	eventpb.UnimplementedEventServiceServer
	repo repository.EventRepository
}

func NewEventHandler(repo repository.EventRepository) *EventHandler {
	return &EventHandler{repo: repo}
}

func (h *EventHandler) GetEvent(ctx context.Context, req *eventpb.GetEventRequest) (*eventpb.GetEventResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "event ID is required")
	}

	event, err := h.repo.GetByID(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to find event: %v", err)
	}

	return &eventpb.GetEventResponse{Event: event.ToProto()}, nil
}

func (h *EventHandler) ListEvents(ctx context.Context, req *eventpb.ListEventsRequest) (*eventpb.ListEventsResponse, error) {
	page := req.Page
	if page <= 0 {
		page = 1
	}

	pageSize := req.PageSize
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}

	events, total, err := h.repo.List(ctx, page, pageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list events: %v", err)
	}

	var protoEvents []*eventpb.Event
	for _, event := range events {
		protoEvents = append(protoEvents, event.ToProto())
	}

	return &eventpb.ListEventsResponse{
		Events: protoEvents,
		Total:  total,
	}, nil
}

func (h *EventHandler) CreateEvent(ctx context.Context, req *eventpb.CreateEventRequest) (*eventpb.CreateEventResponse, error) {
	if req.Name == "" || req.Location == "" || req.Date == "" {
		return nil, status.Error(codes.InvalidArgument, "name, location, and date are required")
	}

	if req.TicketStock <= 0 {
		return nil, status.Error(codes.InvalidArgument, "ticket stock must be greater than 0")
	}

	event, err := model.NewEvent(req.Name, req.Date, req.Location, req.TicketStock)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid date format: %v", err)
	}

	createdEvent, err := h.repo.Create(ctx, event)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create event: %v", err)
	}

	return &eventpb.CreateEventResponse{Event: createdEvent.ToProto()}, nil
}

func (h *EventHandler) UpdateEvent(ctx context.Context, req *eventpb.UpdateEventRequest) (*eventpb.UpdateEventResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "event ID is required")
	}

	existingEvent, err := h.repo.GetByID(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "event not found: %v", err)
	}

	if req.Name != "" {
		existingEvent.Name = req.Name
	}

	if req.Location != "" {
		existingEvent.Location = req.Location
	}

	if req.Date != "" {
		newDate, err := time.Parse(time.RFC3339, req.Date)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid date format: %v", err)
		}
		existingEvent.Date = newDate
	}

	if req.TicketStock > 0 {
		existingEvent.TicketStock = req.TicketStock
	}

	updatedEvent, err := h.repo.Update(ctx, existingEvent)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update event: %v", err)
	}

	return &eventpb.UpdateEventResponse{Event: updatedEvent.ToProto()}, nil
}

func (h *EventHandler) DeleteEvent(ctx context.Context, req *eventpb.DeleteEventRequest) (*eventpb.DeleteEventResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "event ID is required")
	}

	if err := h.repo.Delete(ctx, req.Id); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete event: %v", err)
	}

	return &eventpb.DeleteEventResponse{}, nil
}

func (h *EventHandler) CheckAvailability(ctx context.Context, req *eventpb.CheckAvailabilityRequest) (*eventpb.CheckAvailabilityResponse, error) {
	if req.EventId == "" {
		return nil, status.Error(codes.InvalidArgument, "event ID is required")
	}

	if req.Quantity <= 0 {
		return nil, status.Error(codes.InvalidArgument, "quantity must be greater than 0")
	}

	available, err := h.repo.CheckAvailability(ctx, req.EventId, req.Quantity)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check availability: %v", err)
	}

	return &eventpb.CheckAvailabilityResponse{Available: available}, nil
}
