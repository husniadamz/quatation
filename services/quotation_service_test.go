package services

import (
	"testing"

	"quotation/models"
)

func TestCalculateTotalsRoundsLineItemsBeforeTax(t *testing.T) {
	quotation := &models.Quotation{
		TaxRate: 10,
		Items: makeItems([]QuotationItemInput{
			{Quantity: 3, UnitPrice: 0.335},
			{Quantity: 2, UnitPrice: 10},
		}),
	}

	calculateTotals(quotation)

	if quotation.Items[0].UnitPrice != 0.34 || quotation.Items[0].LineTotal != 1.02 {
		t.Fatalf("first item = (%.2f, %.2f), want (0.34, 1.02)",
			quotation.Items[0].UnitPrice, quotation.Items[0].LineTotal)
	}
	if quotation.Subtotal != 21.02 || quotation.TaxAmount != 2.10 || quotation.GrandTotal != 23.12 {
		t.Fatalf("totals = (%.2f, %.2f, %.2f), want (21.02, 2.10, 23.12)",
			quotation.Subtotal, quotation.TaxAmount, quotation.GrandTotal)
	}
}

func TestMakeItemsRoundsQuantityToDatabasePrecision(t *testing.T) {
	items := makeItems([]QuotationItemInput{
		{Description: "item", Quantity: 1.234, UnitPrice: 10},
	})
	if items[0].Quantity != 1.23 || items[0].LineTotal != 12.30 {
		t.Fatalf("item = (%.2f, %.2f), want (1.23, 12.30)",
			items[0].Quantity, items[0].LineTotal)
	}
}

func TestValidateQuotationInputRejectsInvalidItemsAndTax(t *testing.T) {
	tests := []struct {
		name  string
		input QuotationInput
	}{
		{name: "missing items", input: QuotationInput{CustomerID: 1}},
		{name: "zero quantity", input: QuotationInput{CustomerID: 1, Items: []QuotationItemInput{{Description: "item", UnitPrice: 1}}}},
		{name: "tax over limit", input: QuotationInput{CustomerID: 1, TaxRate: 100.01, Items: []QuotationItemInput{{Description: "item", Quantity: 1}}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateQuotationInput(test.input); err == nil {
				t.Fatal("expected invalid input error")
			}
		})
	}
}
