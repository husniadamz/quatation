package models

import "time"

type Company struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:180;not null;uniqueIndex"`
	Email     string    `json:"email" gorm:"size:255"`
	Phone     string    `json:"phone" gorm:"size:60"`
	Address   string    `json:"address" gorm:"size:2000"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
