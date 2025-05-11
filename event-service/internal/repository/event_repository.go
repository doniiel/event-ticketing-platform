package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/doniiel/event-ticketing-platform/event-service/internal/model"
	"github.com/google/uuid"
)

type EventRepository struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) Create(ctx context.Context, event *model.Event) (*model.Event, error) {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	now := time.Now()
	event.CreatedAt = now
	event.UpdatedAt = now

	query := `
		INSERT INTO events (id, name, date, location, ticket_stock, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		event.ID,
		event.Name,
		event.Date,
		event.Location,
		event.TicketStock,
		event.CreatedAt,
		event.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return event, nil
}

func (r *EventRepository) GetByID(ctx context.Context, id string) (*model.Event, error) {
	query := `
		SELECT id, name, date, location, ticket_stock, created_at, updated_at
		FROM events
		WHERE id = ?
	`

	var event model.Event
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&event.ID,
		&event.Name,
		&event.Date,
		&event.Location,
		&event.TicketStock,
		&event.CreatedAt,
		&event.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("event not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	return &event, nil
}

func (r *EventRepository) Update(ctx context.Context, event *model.Event) (*model.Event, error) {
	event.UpdatedAt = time.Now()

	query := `
		UPDATE events
		SET name = ?, date = ?, location = ?, ticket_stock = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		event.Name,
		event.Date,
		event.Location,
		event.TicketStock,
		event.UpdatedAt,
		event.ID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("event not found")
	}

	return event, nil
}

func (r *EventRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM events WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("event not found")
	}

	return nil
}

func (r *EventRepository) List(ctx context.Context) ([]*model.Event, error) {
	query := `
		SELECT id, name, date, location, ticket_stock, created_at, updated_at
		FROM events
		ORDER BY date ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}
	defer rows.Close()

	var events []*model.Event
	for rows.Next() {
		var event model.Event
		if err := rows.Scan(
			&event.ID,
			&event.Name,
			&event.Date,
			&event.Location,
			&event.TicketStock,
			&event.CreatedAt,
			&event.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return events, nil
}

func (r *EventRepository) CheckAvailability(ctx context.Context, eventID string, quantity int32) (bool, error) {
	query := `SELECT ticket_stock FROM events WHERE id = ?`

	var ticketStock int32
	err := r.db.QueryRowContext(ctx, query, eventID).Scan(&ticketStock)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("event not found: %w", err)
		}
		return false, fmt.Errorf("failed to check availability: %w", err)
	}

	return ticketStock >= quantity, nil
}

func (r *EventRepository) UpdateTicketStock(ctx context.Context, eventID string, quantity int32) error {
	query := `
		UPDATE events
		SET ticket_stock = ticket_stock - ?, updated_at = ?
		WHERE id = ? AND ticket_stock >= ?
	`

	result, err := r.db.ExecContext(ctx, query, quantity, time.Now(), eventID, quantity)
	if err != nil {
		return fmt.Errorf("failed to update ticket stock: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("not enough tickets available or event not found")
	}

	return nil
}
