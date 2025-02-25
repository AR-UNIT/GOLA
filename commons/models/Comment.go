package models

import (
	"github.com/google/uuid"
	"time"
)

type Comment struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;" json:"id"`
	ImageID   uuid.UUID `gorm:"type:uuid;not null" json:"image_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
