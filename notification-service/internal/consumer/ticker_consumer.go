package consumer

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/doniiel/event-ticketing-platform/notification-service/internal/repository"
	ticketpb "github.com/doniiel/event-ticketing-platform/proto/ticket"
	"google.golang.org/grpc"
)

type TicketConsumer struct {
	ticketServiceAddr string
	notificationRepo  repository.NotificationRepository
	stopCh            chan struct{}
}

func NewTicketConsumer(ticketServiceAddr string, notificationRepo repository.NotificationRepository) *TicketConsumer {
	return &TicketConsumer{
		ticketServiceAddr: ticketServiceAddr,
		notificationRepo:  notificationRepo,
		stopCh:            make(chan struct{}),
	}
}

func (c *TicketConsumer) Start() error {
	conn, err := grpc.Dial(c.ticketServiceAddr, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("failed to connect to ticket service: %w", err)
	}
	defer conn.Close()

	client := ticketpb.NewTicketServiceClient(conn)

	log.Printf("Starting ticket event consumer, connecting to %s", c.ticketServiceAddr)
	go c.simulateTicketEventListener(client)

	return nil
}

func (c *TicketConsumer) Stop() {
	close(c.stopCh)
}

func (c *TicketConsumer) simulateTicketEventListener(client ticketpb.TicketServiceClient) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Println("Checking for new ticket events...")
		case <-c.stopCh:
			log.Println("Ticket consumer stopped")
			return
		}
	}
}

func (c *TicketConsumer) processTicketPurchase(userID, eventID string) {
	message := fmt.Sprintf("Your ticket for event %s has been confirmed!", eventID)

	_, err := c.notificationRepo.SaveNotification(userID, message)
	if err != nil {
		log.Printf("Failed to send purchase confirmation: %v", err)
		return
	}

	log.Printf("Purchase confirmation sent to user %s for event %s", userID, eventID)
}
