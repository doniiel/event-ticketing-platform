package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/doniiel/event-ticketing-platform/event-service/internal/model"
)

type EventRepository interface {
	Create(ctx context.Context, event *model.Event) (*model.Event, error)
	GetByID(ctx context.Context, id string) (*model.Event, error)
	Update(ctx context.Context, event *model.Event) (*model.Event, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, page, pageSize int32) ([]*model.Event, int32, error)
	CheckAvailability(ctx context.Context, eventID string, quantity int32) (bool, error)
	UpdateTicketStock(ctx context.Context, eventID string, quantity int32) error
}

type EventRepositoryImpl struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) EventRepository {
	return &EventRepositoryImpl{db: db}
}

func (r *EventRepositoryImpl) Create(ctx context.Context, event *model.Event) (*model.Event, error) {
	query := `
		INSERT INTO events (id, name, date, location, ticket_stock)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		event.ID,
		event.Name,
		event.Date,
		event.Location,
		event.TicketStock,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	createdEvent, err := r.GetByID(ctx, event.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve created event: %w", err)
	}

	return createdEvent, nil
}

func (r *EventRepositoryImpl) GetByID(ctx context.Context, id string) (*model.Event, error) {
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

func (r *EventRepositoryImpl) Update(ctx context.Context, event *model.Event) (*model.Event, error) {
	query := `
		UPDATE events
		SET name = ?, date = ?, location = ?, ticket_stock = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		event.Name,
		event.Date,
		event.Location,
		event.TicketStock,
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

	updatedEvent, err := r.GetByID(ctx, event.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve updated event: %w", err)
	}

	return updatedEvent, nil
}

func (r *EventRepositoryImpl) Delete(ctx context.Context, id string) error {
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

func (r *EventRepositoryImpl) List(ctx context.Context, page, pageSize int32) ([]*model.Event, int32, error) {
	var total int32
	countQuery := `SELECT COUNT(*) FROM events`
	err := r.db.QueryRowContext(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count events: %w", err)
	}

	query := `
		SELECT id, name, date, location, ticket_stock, created_at, updated_at
		FROM events
		ORDER BY date ASC
		LIMIT ? OFFSET ?
	`

	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list events: %w", err)
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
			return nil, 0, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return events, total, nil
}

func (r *EventRepositoryImpl) CheckAvailability(ctx context.Context, eventID string, quantity int32) (bool, error) {
	query := `SELECT ticket_stock FROM events WHERE id = ?`

	if quantity <= 0 {
		return false, fmt.Errorf("invalid quantity: must be greater than 0")
	}

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

func (r *EventRepositoryImpl) UpdateTicketStock(ctx context.Context, eventID string, quantity int32) error {
	query := `
		UPDATE events
		SET ticket_stock = ticket_stock - ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND ticket_stock >= ?
	`

	if quantity <= 0 {
		return fmt.Errorf("invalid quantity: must be greater than 0")
	}

	result, err := r.db.ExecContext(ctx, query, quantity, eventID, quantity)
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
