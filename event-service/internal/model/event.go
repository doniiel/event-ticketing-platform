package model

type Event struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Date        string `json:"date"`
	Location    string `json:"location"`
	TicketStock int32  `json:"ticket_stock"`
}
