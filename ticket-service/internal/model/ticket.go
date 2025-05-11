package model

import (
	"time"

	ticketpb "github.com/doniiel/event-ticketing-platform/proto/ticket"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TicketStatus string

const (
	TicketStatusReserved  TicketStatus = "RESERVED"
	TicketStatusConfirmed TicketStatus = "CONFIRMED"
	TicketStatusCancelled TicketStatus = "CANCELLED"
	TicketStatusUsed      TicketStatus = "USED"
)

type Ticket struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	EventID   string             `bson:"event_id" json:"event_id"`
	UserID    string             `bson:"user_id" json:"user_id"`
	Status    TicketStatus       `bson:"status" json:"status"`
	Quantity  int32              `bson:"quantity" json:"quantity"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

func (t *Ticket) ToProto() *ticketpb.Ticket {
	return &ticketpb.Ticket{
		Id:      t.ID.Hex(),
		EventId: t.EventID,
		UserId:  t.UserID,
		Status:  string(t.Status),
	}
}

func NewTicket(eventID, userID string, quantity int32) *Ticket {
	now := time.Now()
	return &Ticket{
		ID:        primitive.NewObjectID(),
		EventID:   eventID,
		UserID:    userID,
		Status:    TicketStatusReserved,
		Quantity:  quantity,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
