package repository

import (
	"database/sql"
	"errors"
	"github.com/doniiel/event-ticketing-platform/event-service/internal/model"
	"github.com/google/uuid"
)

type EventRepository struct{ db *sql.DB }

func NewEventRepository(dsn string) (*EventRepository, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	return &EventRepository{db: db}, nil
}

func (r *EventRepository) CreateEvent(event *model.Event) error {
	event.ID = uuid.New().String()
	_, err := r.db.Exec(
		"INSERT INTO events (id, name, date, location, ticket_stock) VALUES (?, ?, ?, ?, ?)",
		event.ID,
		event.Name,
		event.Date,
		event.Location,
		event.TicketStock,
	)
	return err
}

func (r *EventRepository) GetEvent(id string) (*model.Event, error) {
	event := &model.Event{}
	err := r.db.QueryRow("SELECT id, name, date, location, ticket_stock FROM events WHERE id = ?", id).
		Scan(&event.ID, &event.Name, &event.Date, &event.Location, &event.TicketStock)
	if err == sql.ErrNoRows {
		return nil, errors.New("event not found")
	}
	if err != nil {
		return nil, err
	}
	return event, nil
}

func (r *EventRepository) UpdateEvent(event *model.Event) error {
	result, err := r.db.Exec(
		"UPDATE events SET name = ?, date = ?, location = ?, ticket_stock = ? WHERE id = ?",
		event.Name,
		event.Date,
		event.Location,
		event.TicketStock,
		event.ID,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("event not found")
	}
	return nil
}

func (r *EventRepository) DeleteEvent(id string) error {
	result, err := r.db.Exec(
		"DELETE FROM events WHERE id = ?", id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("event not found")
	}
	return nil
}

func (r *EventRepository) CheckAvailability(eventID string, quantity int32) (bool, error) {
	var ticketStock int32
	err := r.db.QueryRow(
		"SELECT ticket_stock FROM events WHERE id = ?", eventID).Scan(&ticketStock)
	if err == sql.ErrNoRows {
		return false, errors.New("event not found")
	}
	if err != nil {
		return false, err
	}
	return quantity <= ticketStock, nil
}
