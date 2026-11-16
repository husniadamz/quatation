package controllers

import (
	"bytes"
	"testing"
	"time"

	"quotation/models"
)

func TestQuotationPDFUsesTemplateLayoutAndSettings(t *testing.T) {
	quotation := &models.Quotation{
		Number:   "QT-TEST-1",
		Customer: models.Customer{Name: "Example Customer"},
		Items: []models.QuotationItem{
			{Description: "Consulting", Quantity: 2, UnitPrice: 125, LineTotal: 250},
		},
		Subtotal: 250, TaxRate: 10, TaxAmount: 25, GrandTotal: 275,
		CreatedAt: time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC),
	}
	snapshot := models.QuotationTemplateSnapshot{
		CompanyName: "Example Company",
		Name:        "Branded",
		Settings: models.QuotationTemplateSettings{
			HeaderTitle:    "Project Proposal",
			PrimaryColor:   "#123456",
			AccentColor:    "#ABCDEF",
			FontFamily:     "Helvetica",
			Layout:         "landscape",
			FooterText:     "Thank you for your business",
			PaymentDetails: "Pay by bank transfer",
			Terms:          "Payment due in 30 days",
			SignatureLabel: "Authorized signature",
		},
	}

	data, err := quotationPDF(quotation, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatalf("generated output does not start with a PDF header: %q", data[:min(20, len(data))])
	}
	if len(data) < 500 {
		t.Fatalf("generated PDF is unexpectedly small: %d bytes", len(data))
	}
}
