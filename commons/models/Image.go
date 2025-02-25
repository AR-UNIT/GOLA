package models

import (
	"github.com/google/uuid"
	"time"
)

type Image struct {
	ID          uuid.UUID  `gorm:"type:uuid;primary_key;" json:"id"`
	UserID      uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	FilePath    string     `json:"-"`
	PublicURL   string     `json:"url"`
	IsPrivate   bool       `json:"is_private"`
	Tags        []Tag      `gorm:"many2many:image_tags;" json:"tags"`
	Comments    []Comment  `json:"comments,omitempty"`
	Likes       []Like     `json:"-"`
	LikeCount   int        `json:"like_count"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}
