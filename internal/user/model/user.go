package model

import "time"

type User struct {
	ID           string
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	AuthToken    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
