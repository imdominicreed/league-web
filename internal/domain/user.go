package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	PasswordHash string    `json:"-" gorm:"not null"`
	DisplayName  string    `json:"displayName" gorm:"uniqueIndex;not null"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type UserSession struct {
	ID               uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID           uuid.UUID `json:"userId" gorm:"type:uuid;not null"`
	RefreshTokenHash string    `json:"-" gorm:"not null"`
	ExpiresAt        time.Time `json:"expiresAt" gorm:"not null"`
	CreatedAt        time.Time `json:"createdAt"`
}
