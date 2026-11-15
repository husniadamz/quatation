package models

import "time"

const (
	StatusDraft    = "draft"
	StatusSent     = "sent"
	StatusAccepted = "accepted"
	StatusRejected = "rejected"
	StatusExpired  = "expired"
)

type Quotation struct {
	ID         uint            `json:"id" gorm:"primaryKey"`
	Number     string          `json:"number" gorm:"size:40;uniqueIndex;not null"`
	CustomerID uint            `json:"customer_id" gorm:"not null;index"`
	Customer   Customer        `json:"customer,omitempty" gorm:"foreignKey:CustomerID"`
	Items      []QuotationItem `json:"items,omitempty" gorm:"foreignKey:QuotationID;constraint:OnDelete:CASCADE"`
	Status     string          `json:"status" gorm:"size:20;not null;default:draft"`
	Subtotal   float64         `json:"subtotal" gorm:"type:decimal(12,2);not null"`
	TaxRate    float64         `json:"tax_rate" gorm:"type:decimal(5,2);not null;default:0"`
	TaxAmount  float64         `json:"tax_amount" gorm:"type:decimal(12,2);not null"`
	GrandTotal float64         `json:"grand_total" gorm:"type:decimal(12,2);not null"`
	Notes      string          `json:"notes" gorm:"type:text"`
	ValidUntil *time.Time      `json:"valid_until,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type QuotationItem struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	QuotationID uint      `json:"quotation_id" gorm:"not null;index"`
	Description string    `json:"description" gorm:"size:255;not null"`
	Quantity    float64   `json:"quantity" gorm:"type:decimal(10,2);not null"`
	UnitPrice   float64   `json:"unit_price" gorm:"type:decimal(12,2);not null"`
	LineTotal   float64   `json:"line_total" gorm:"type:decimal(12,2);not null"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
