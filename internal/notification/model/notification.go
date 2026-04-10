package model

import "time"

type Notification struct {
	ID        string
	EventType string
	Recipient string
	Channel   string
	Subject   string
	Body      string
	CreatedAt time.Time
}
