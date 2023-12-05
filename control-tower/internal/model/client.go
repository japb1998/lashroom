package model

import (
	"time"

	"github.com/google/uuid"
)

type ClientItem struct {
	PrimaryKey   string    `json:"primaryKey"`
	SortKey      string    `json:"sortKey"`
	Phone        string    `json:"phone"`
	Email        string    `json:"email"`
	FirstName    string    `json:"firstName"`
	LastName     string    `json:"lastName"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"createdAt"`
	LastUpdateAt time.Time `json:"lastUpdateAt"`
	OptIn        bool      `json:"optIn"`
}

func NewClientItem(creator, phone, email, firstName, lastName, description string) *ClientItem {
	return &ClientItem{
		PrimaryKey:  creator,
		SortKey:     uuid.New().String(),
		Phone:       phone,
		Email:       email,
		FirstName:   firstName,
		LastName:    lastName,
		Description: description,
		CreatedAt:   time.Now(),
		OptIn:       true,
	}
}
