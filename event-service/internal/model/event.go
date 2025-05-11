package model

import (
	"time"

	eventpb "github.com/doniiel/event-ticketing-platform/proto/event"
)

type Event struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Date        time.Time `json:"date"`
	Location    string    `json:"location"`
	TicketStock int32     `json:"ticket_stock"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (e *Event) ToProto() *eventpb.Event {
	return &eventpb.Event{
		Id:          e.ID,
		Name:        e.Name,
		Date:        e.Date.Format(time.RFC3339),
		Location:    e.Location,
		TicketStock: e.TicketStock,
	}
}

func EventFromProto(e *eventpb.Event) (*Event, error) {
	date, err := time.Parse(time.RFC3339, e.Date)
	if err != nil {
		return nil, err
	}

	return &Event{
		ID:          e.Id,
		Name:        e.Name,
		Date:        date,
		Location:    e.Location,
		TicketStock: e.TicketStock,
	}, nil
}
