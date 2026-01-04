package domain

import (
	"time"

	"github.com/google/uuid"
)

type DraftMode string

const (
	DraftModeProPlay  DraftMode = "pro_play"
	DraftModeFearless DraftMode = "fearless"
)

type RoomStatus string

const (
	RoomStatusWaiting    RoomStatus = "waiting"
	RoomStatusInProgress RoomStatus = "in_progress"
	RoomStatusCompleted  RoomStatus = "completed"
)

type Room struct {
	ID                   uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ShortCode            string     `json:"shortCode" gorm:"uniqueIndex;not null"`
	CreatedBy            uuid.UUID  `json:"createdBy" gorm:"type:uuid;not null"`
	DraftMode            DraftMode  `json:"draftMode" gorm:"not null;default:'pro_play'"`
	TimerDurationSeconds int        `json:"timerDurationSeconds" gorm:"not null;default:30"`
	Status               RoomStatus `json:"status" gorm:"not null;default:'waiting'"`
	BlueSideUserID       *uuid.UUID `json:"blueSideUserId" gorm:"type:uuid"`
	RedSideUserID        *uuid.UUID `json:"redSideUserId" gorm:"type:uuid"`
	SeriesID             *uuid.UUID `json:"seriesId" gorm:"type:uuid"`
	GameNumber           int        `json:"gameNumber" gorm:"default:1"`
	CreatedAt            time.Time  `json:"createdAt"`
	StartedAt            *time.Time `json:"startedAt"`
	CompletedAt          *time.Time `json:"completedAt"`

	// Relations
	Creator      *User `json:"creator,omitempty" gorm:"foreignKey:CreatedBy"`
	BlueSideUser *User `json:"blueSideUser,omitempty" gorm:"foreignKey:BlueSideUserID"`
	RedSideUser  *User `json:"redSideUser,omitempty" gorm:"foreignKey:RedSideUserID"`
}

type Side string

const (
	SideBlue      Side = "blue"
	SideRed       Side = "red"
	SideSpectator Side = "spectator"
)
