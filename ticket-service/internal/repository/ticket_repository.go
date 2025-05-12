package repository

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/doniiel/event-ticketing-platform/ticket-service/internal/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type TicketRepository struct {
	db         *mongo.Database
	collection *mongo.Collection
}

func NewTicketRepository(db *mongo.Database) *TicketRepository {
	collection := db.Collection("tickets")

	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "user_id", Value: 1},
			{Key: "event_id", Value: 1},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		log.Printf("Error creating index: %v", err)
	}

	return &TicketRepository{
		db:         db,
		collection: collection,
	}
}

func (r *TicketRepository) Create(ctx context.Context, ticket *model.Ticket) (*model.Ticket, error) {
	result, err := r.collection.InsertOne(ctx, ticket)
	if err != nil {
		return nil, err
	}

	if oid, ok := result.InsertedID.(primitive.ObjectID); ok {
		ticket.ID = oid
	}

	return ticket, nil
}

func (r *TicketRepository) GetByID(ctx context.Context, id string) (*model.Ticket, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid ID format")
	}

	var ticket model.Ticket
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&ticket)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("ticket not found")
		}
		return nil, err
	}

	return &ticket, nil
}

func (r *TicketRepository) GetByUserID(ctx context.Context, userID string) ([]*model.Ticket, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {

		}
	}(cursor, ctx)

	var tickets []*model.Ticket
	if err := cursor.All(ctx, &tickets); err != nil {
		return nil, err
	}

	return tickets, nil
}

func (r *TicketRepository) UpdateStatus(ctx context.Context, id string, status model.TicketStatus) (*model.Ticket, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, errors.New("invalid ID format")
	}

	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var ticket model.Ticket

	err = r.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objectID},
		update,
		opts,
	).Decode(&ticket)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, errors.New("ticket not found")
		}
		return nil, err
	}

	return &ticket, nil
}

func (r *TicketRepository) GetActiveTicketsForEvent(ctx context.Context, eventID string) ([]*model.Ticket, error) {
	cursor, err := r.collection.Find(ctx, bson.M{
		"event_id": eventID,
		"status": bson.M{
			"$in": []string{
				string(model.TicketStatusReserved),
				string(model.TicketStatusConfirmed),
			},
		},
	})

	if err != nil {
		return nil, err
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {

		}
	}(cursor, ctx)

	var tickets []*model.Ticket
	if err := cursor.All(ctx, &tickets); err != nil {
		return nil, err
	}

	return tickets, nil
}
