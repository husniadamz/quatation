package models

import "time"

type Customer struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:150;not null"`
	Email     string    `json:"email" gorm:"size:255"`
	Phone     string    `json:"phone" gorm:"size:40"`
	Address   string    `json:"address" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
