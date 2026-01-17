package domain

import (
	"time"

	"gorm.io/datatypes"
)

type Champion struct {
	ID           string         `json:"id" gorm:"primaryKey"`          // e.g., "Aatrox"
	Key          string         `json:"key" gorm:"not null"`           // e.g., "266"
	Name         string         `json:"name" gorm:"not null"`          // Display name
	Title        string         `json:"title"`                         // e.g., "the Darkin Blade"
	ImageURL     string         `json:"imageUrl" gorm:"not null"`      // Full URL to champion image
	Tags         datatypes.JSON `json:"tags" gorm:"type:jsonb"`        // ["Fighter", "Tank"]
	Lanes        datatypes.JSON `json:"lanes" gorm:"type:jsonb"`       // ["mid", "top"] - lanes with >1% playrate, ordered by playrate
	LastSyncedAt time.Time      `json:"lastSyncedAt"`
}

type ChampionTag string

const (
	TagFighter   ChampionTag = "Fighter"
	TagTank      ChampionTag = "Tank"
	TagMage      ChampionTag = "Mage"
	TagAssassin  ChampionTag = "Assassin"
	TagSupport   ChampionTag = "Support"
	TagMarksman  ChampionTag = "Marksman"
)
