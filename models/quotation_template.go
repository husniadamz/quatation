package models

import "time"

type QuotationTemplateSettings struct {
	CompanyEmail   string `json:"company_email"`
	CompanyPhone   string `json:"company_phone"`
	CompanyAddress string `json:"company_address"`
	HeaderTitle    string `json:"header_title"`
	HeaderText     string `json:"header_text"`
	FooterText     string `json:"footer_text"`
	PrimaryColor   string `json:"primary_color"`
	AccentColor    string `json:"accent_color"`
	FontFamily     string `json:"font_family"`
	Layout         string `json:"layout"`
	Terms          string `json:"terms"`
	PaymentDetails string `json:"payment_details"`
	SignatureLabel string `json:"signature_label"`
}

type QuotationTemplate struct {
	ID               uint                      `json:"id" gorm:"primaryKey"`
	CompanyID        uint                      `json:"company_id" gorm:"not null;default:0;index"`
	CompanyName      string                    `json:"company_name" gorm:"size:180;not null;uniqueIndex:idx_template_company_name"`
	Name             string                    `json:"name" gorm:"size:120;not null;uniqueIndex:idx_template_company_name"`
	Settings         QuotationTemplateSettings `json:"settings" gorm:"type:json;serializer:json;not null"`
	LogoPath         string                    `json:"-" gorm:"size:255"`
	LogoContentType  string                    `json:"logo_content_type,omitempty" gorm:"size:40"`
	LogoOriginalName string                    `json:"logo_original_name,omitempty" gorm:"size:255"`
	LogoURL          string                    `json:"logo_url,omitempty" gorm:"-"`
	IsActive         bool                      `json:"is_active" gorm:"not null;default:true;index"`
	IsDefault        bool                      `json:"is_default" gorm:"not null;default:false"`
	CreatedAt        time.Time                 `json:"created_at"`
	UpdatedAt        time.Time                 `json:"updated_at"`
}

type QuotationTemplateSnapshot struct {
	ID              uint                      `json:"id"`
	CompanyName     string                    `json:"company_name"`
	Name            string                    `json:"name"`
	Settings        QuotationTemplateSettings `json:"settings"`
	LogoPath        string                    `json:"-"`
	LogoURL         string                    `json:"logo_url,omitempty"`
	LogoContentType string                    `json:"logo_content_type,omitempty"`
}
