package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/doniiel/event-ticketing-platform/event-service/internal/model"
	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("mysql", "root:password@tcp(localhost:3306)/?parseTime=true")
	if err != nil {
		t.Fatalf("failed to connect to MySQL: %v", err)
	}

	_, _ = db.Exec("DROP DATABASE IF EXISTS events_test")
	_, _ = db.Exec("CREATE DATABASE events_test")
	_, _ = db.Exec("USE events_test")

	schema := `
	CREATE TABLE events (
		id VARCHAR(36) PRIMARY KEY,
		name VARCHAR(255),
		date DATETIME,
		location VARCHAR(255),
		ticket_stock INT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	);
	`
	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	return db
}

func seedEvent(t *testing.T, repo EventRepository, ticketStock int32) *model.Event {
	id := uuid.New().String()
	event := &model.Event{
		ID:          id,
		Name:        "Test Event",
		Date:        time.Now().AddDate(0, 1, 0),
		Location:    "Test Location",
		TicketStock: ticketStock,
	}

	created, err := repo.Create(context.Background(), event)
	assert.NoError(t, err)
	return created
}

func TestEventRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewEventRepository(db)

	event, err := model.NewEvent("Test Concert", "2025-06-01T19:00:00Z", "Test Arena", 100)
	assert.NoError(t, err)

	created, err := repo.Create(context.Background(), event)
	assert.NoError(t, err)
	assert.Equal(t, event.ID, created.ID)
	assert.Equal(t, event.Name, created.Name)
	assert.Equal(t, event.TicketStock, created.TicketStock)

	var dbEvent model.Event
	err = db.QueryRow("SELECT id, name, ticket_stock FROM events WHERE id = ?", event.ID).
		Scan(&dbEvent.ID, &dbEvent.Name, &dbEvent.TicketStock)
	assert.NoError(t, err)
	assert.Equal(t, event.ID, dbEvent.ID)
	assert.Equal(t, event.Name, dbEvent.Name)
	assert.Equal(t, event.TicketStock, dbEvent.TicketStock)
}

func TestEventRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewEventRepository(db)

	_, err := repo.GetByID(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event not found")

	event, err := model.NewEvent("Test Concert", "2025-06-01T19:00:00Z", "Test Arena", 100)
	assert.NoError(t, err)
	_, err = repo.Create(context.Background(), event)
	assert.NoError(t, err)

	found, err := repo.GetByID(context.Background(), event.ID)
	assert.NoError(t, err)
	assert.Equal(t, event.ID, found.ID)
	assert.Equal(t, event.Name, found.Name)
	assert.Equal(t, event.TicketStock, found.TicketStock)
}

func TestEventRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewEventRepository(db)

	event, err := model.NewEvent("Test Concert", "2025-06-01T19:00:00Z", "Test Arena", 100)
	assert.NoError(t, err)
	_, err = repo.Create(context.Background(), event)
	assert.NoError(t, err)

	updatedEvent := *event
	updatedEvent.Name = "Updated Concert"
	updatedEvent.TicketStock = 200

	result, err := repo.Update(context.Background(), &updatedEvent)
	assert.NoError(t, err)
	assert.Equal(t, updatedEvent.Name, result.Name)
	assert.Equal(t, updatedEvent.TicketStock, result.TicketStock)

	var dbEvent model.Event
	err = db.QueryRow("SELECT name, ticket_stock FROM events WHERE id = ?", event.ID).
		Scan(&dbEvent.Name, &dbEvent.TicketStock)
	assert.NoError(t, err)
	assert.Equal(t, updatedEvent.Name, dbEvent.Name)
	assert.Equal(t, updatedEvent.TicketStock, dbEvent.TicketStock)
}

func TestEventRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewEventRepository(db)

	err := repo.Delete(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event not found")

	event, err := model.NewEvent("Test Concert", "2025-06-01T19:00:00Z", "Test Arena", 100)
	assert.NoError(t, err)
	_, err = repo.Create(context.Background(), event)
	assert.NoError(t, err)

	err = repo.Delete(context.Background(), event.ID)
	assert.NoError(t, err)

	_, err = repo.GetByID(context.Background(), event.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event not found")
}

func TestEventRepository_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewEventRepository(db)

	events := make([]*model.Event, 5)
	for i := 0; i < 5; i++ {
		event, err := model.NewEvent(
			"Test Concert "+string(rune('A'+i)),
			time.Now().AddDate(0, 0, i).Format(time.RFC3339),
			"Venue "+string(rune('A'+i)),
			int32(100+i*10),
		)
		assert.NoError(t, err)
		_, err = repo.Create(context.Background(), event)
		assert.NoError(t, err)
		events[i] = event
	}

	result, total, err := repo.List(context.Background(), 1, 2)
	assert.NoError(t, err)
	assert.Equal(t, int32(5), total)
	assert.Len(t, result, 2)
	assert.Equal(t, events[0].ID, result[0].ID)
	assert.Equal(t, events[1].ID, result[1].ID)

	result, total, err = repo.List(context.Background(), 2, 2)
	assert.NoError(t, err)
	assert.Equal(t, int32(5), total)
	assert.Len(t, result, 2)
	assert.Equal(t, events[2].ID, result[0].ID)
	assert.Equal(t, events[3].ID, result[1].ID)

	result, total, err = repo.List(context.Background(), 3, 2)
	assert.NoError(t, err)
	assert.Equal(t, int32(5), total)
	assert.Len(t, result, 1)
	assert.Equal(t, events[4].ID, result[0].ID)
}

func TestEventRepository_CheckAvailability(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewEventRepository(db)

	event, err := model.NewEvent("Test Concert", "2025-06-01T19:00:00Z", "Test Arena", 100)
	assert.NoError(t, err)
	_, err = repo.Create(context.Background(), event)
	assert.NoError(t, err)

	tests := []struct {
		name      string
		quantity  int32
		want      bool
		wantError bool
	}{
		{"Available", 50, true, false},
		{"Exact", 100, true, false},
		{"NotAvailable", 101, false, false},
		{"InvalidQuantity", -1, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			available, err := repo.CheckAvailability(context.Background(), event.ID, tt.quantity)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, available)
			}
		})
	}

	_, err = repo.CheckAvailability(context.Background(), uuid.NewString(), 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "event not found")
}

func TestEventRepository_UpdateTicketStock(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	repo := NewEventRepository(db)

	t.Run("Valid", func(t *testing.T) {
		event := seedEvent(t, repo, 100)
		err := repo.UpdateTicketStock(context.Background(), event.ID, 10)
		assert.NoError(t, err)

		updated, _ := repo.GetByID(context.Background(), event.ID)
		assert.Equal(t, int32(90), updated.TicketStock)
	})

	t.Run("Exact", func(t *testing.T) {
		event := seedEvent(t, repo, 90)
		err := repo.UpdateTicketStock(context.Background(), event.ID, 90)
		assert.NoError(t, err)

		updated, _ := repo.GetByID(context.Background(), event.ID)
		assert.Equal(t, int32(0), updated.TicketStock)
	})

	t.Run("NotAvailable", func(t *testing.T) {
		event := seedEvent(t, repo, 5)
		err := repo.UpdateTicketStock(context.Background(), event.ID, 10)
		assert.Error(t, err)
	})

	t.Run("InvalidQuantity", func(t *testing.T) {
		event := seedEvent(t, repo, 100)
		err := repo.UpdateTicketStock(context.Background(), event.ID, 0)
		assert.Error(t, err)
	})
}
