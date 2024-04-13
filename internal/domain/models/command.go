package models

import "time"

type Command struct {
	ID        int
	Script    string
	Status    string
	PID       *int
	Output    string
	CreatedAt time.Time
	UpdatedAt time.Time
}
