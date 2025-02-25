package models

import (
	"github.com/google/uuid"
	"time"
)

type Like struct {
	ID        uint      `gorm:"primary_key" json:"-"`
	ImageID   uuid.UUID `gorm:"type:uuid;not null" json:"image_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}
