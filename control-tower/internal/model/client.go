package model

import (
	"time"

	"github.com/google/uuid"
)

type ClientItem struct {
	PrimaryKey   string     `json:"primaryKey"`
	SortKey      string     `json:"sortKey"`
	Phone        string     `json:"phone" validate:"omitempty,e164"`
	Email        string     `json:"email" validate:"omitempty,email"`
	FirstName    string     `json:"firstName" validate:"omitempty,min=2"`
	LastName     string     `json:"lastName" validate:"omitempty"`
	Description  string     `json:"description"`
	OptIn        bool       `json:"optIn"`
	LastSeen     *time.Time `json:"lastSeen"`
	CreatedAt    time.Time  `json:"createdAt"`
	LastUpdateAt time.Time  `json:"lastUpdateAt"`
}

func NewClientItem(creator, phone, email, firstName, lastName, description string, lastSeen *time.Time) *ClientItem {
	return &ClientItem{
		PrimaryKey:  creator,
		SortKey:     uuid.New().String(),
		Phone:       phone,
		Email:       email,
		FirstName:   firstName,
		LastName:    lastName,
		Description: description,
		CreatedAt:   time.Now().UTC(),
		OptIn:       true,
		LastSeen:    lastSeen,
	}
}
