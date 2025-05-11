package handler

import (
	"context"
	"log"

	eventpb "github.com/doniiel/event-ticketing-platform/proto/event"
	notificationpb "github.com/doniiel/event-ticketing-platform/proto/notification"
	ticketpb "github.com/doniiel/event-ticketing-platform/proto/ticket"
	"github.com/doniiel/event-ticketing-platform/ticket-service/internal/model"
	"github.com/doniiel/event-ticketing-platform/ticket-service/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TicketHandler struct {
	ticketpb.UnimplementedTicketServiceServer
	repo               *repository.TicketRepository
	eventClient        eventpb.EventServiceClient
	notificationClient notificationpb.NotificationServiceClient
}

func NewTicketHandler(
	repo *repository.TicketRepository,
	eventConn *grpc.ClientConn,
	notifConn *grpc.ClientConn,
) *TicketHandler {
	return &TicketHandler{
		repo:               repo,
		eventClient:        eventpb.NewEventServiceClient(eventConn),
		notificationClient: notificationpb.NewNotificationServiceClient(notifConn),
	}
}

func (h *TicketHandler) PurchaseTicket(ctx context.Context, req *ticketpb.PurchaseTicketRequest) (*ticketpb.PurchaseTicketResponse, error) {
	if req.EventId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "event ID and user ID are required")
	}

	if req.Quantity <= 0 {
		return nil, status.Error(codes.InvalidArgument, "quantity must be greater than 0")
	}

	availRes, err := h.eventClient.CheckAvailability(ctx, &eventpb.CheckAvailabilityRequest{
		EventId:  req.EventId,
		Quantity: req.Quantity,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to check ticket availability: %v", err)
	}

	if !availRes.Available {
		return nil, status.Error(codes.ResourceExhausted, "not enough tickets available")
	}

	ticket := model.NewTicket(req.EventId, req.UserId, req.Quantity)
	createdTicket, err := h.repo.Create(ctx, ticket)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create ticket: %v", err)
	}

	eventRes, err := h.eventClient.GetEvent(ctx, &eventpb.GetEventRequest{
		Id: req.EventId,
	})
	if err != nil {
		log.Printf("Failed to get event details: %v", err)
	} else {
		go func() {
			message := "Thank you for purchasing tickets to " + eventRes.Event.Name
			h.notificationClient.SendNotification(context.Background(), &notificationpb.SendNotificationRequest{
				UserId:  req.UserId,
				Message: message,
			})
		}()
	}

	return &ticketpb.PurchaseTicketResponse{
		Ticket: createdTicket.ToProto(),
	}, nil
}

func (h *TicketHandler) GetTicket(ctx context.Context, req *ticketpb.GetTicketRequest) (*ticketpb.GetTicketResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "ticket ID is required")
	}

	if _, err := primitive.ObjectIDFromHex(req.Id); err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid ticket ID format")
	}

	ticket, err := h.repo.GetByID(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to find ticket: %v", err)
	}

	return &ticketpb.GetTicketResponse{
		Ticket: ticket.ToProto(),
	}, nil
}
