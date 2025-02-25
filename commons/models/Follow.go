package models

import (
	"github.com/google/uuid"
	"time"
)

type Follow struct {
	ID          uint      `gorm:"primary_key" json:"-"`
	FollowerID  uuid.UUID `gorm:"type:uuid;not null" json:"follower_id"`
	FollowingID uuid.UUID `gorm:"type:uuid;not null" json:"following_id"`
	CreatedAt   time.Time `json:"created_at"`
}
