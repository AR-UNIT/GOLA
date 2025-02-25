package models

type Tag struct {
	ID    uint   `gorm:"primary_key" json:"id"`
	Name  string `gorm:"unique;not null" json:"name"`
	Count int    `json:"count"`
}
