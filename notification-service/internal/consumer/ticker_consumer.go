package consumer

import (
	"fmt"
	"google.golang.org/grpc/credentials/insecure"
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
	client            ticketpb.TicketServiceClient
	conn              *grpc.ClientConn
}

func NewTicketConsumer(ticketServiceAddr string, notificationRepo repository.NotificationRepository) *TicketConsumer {
	return &TicketConsumer{
		ticketServiceAddr: ticketServiceAddr,
		notificationRepo:  notificationRepo,
		stopCh:            make(chan struct{}),
	}
}

func (c *TicketConsumer) Start() error {
	conn, err := grpc.Dial(
		c.ticketServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to ticket service: %w", err)
	}

	c.conn = conn
	c.client = ticketpb.NewTicketServiceClient(conn)

	log.Printf("Starting ticket event consumer, connecting to %s", c.ticketServiceAddr)
	go c.simulateTicketEventListener()

	return nil
}

func (c *TicketConsumer) Stop() {
	close(c.stopCh)
	if c.conn != nil {
		err := c.conn.Close()
		if err != nil {
			return
		}
	}
}

func (c *TicketConsumer) simulateTicketEventListener() {
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
