package domain

import "time"

type Item struct {
	ID        string
	UserID    string
	Type      string
	Title     string
	Content   string
	Version   int
	Deleted   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
