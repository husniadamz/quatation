package services

import (
	"errors"
	"testing"

	"quotation/models"
)

func TestTemplateFromInputNormalizesDefaultsAndSettings(t *testing.T) {
	template, err := templateFromInput(QuotationTemplateInput{
		CompanyName: "  Example Company ",
		Name:        "  Standard ",
		IsActive:    true,
		Settings: models.QuotationTemplateSettings{
			CompanyEmail: " sales@example.com ", Layout: "LANDSCAPE",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if template.CompanyName != "Example Company" || template.Name != "Standard" {
		t.Fatalf("template identity was not normalized: %#v", template)
	}
	if template.Settings.Layout != "landscape" || template.Settings.FontFamily != "Arial" ||
		template.Settings.HeaderTitle != "Quotation" || template.Settings.PrimaryColor != "#264A73" {
		t.Fatalf("unexpected normalized settings: %#v", template.Settings)
	}
	if template.Settings.CompanyEmail != "sales@example.com" {
		t.Fatalf("company email = %q", template.Settings.CompanyEmail)
	}
}

func TestTemplateFromInputRejectsUnsafeOrInvalidSettings(t *testing.T) {
	tests := []struct {
		name  string
		input QuotationTemplateInput
	}{
		{
			name: "invalid color",
			input: QuotationTemplateInput{CompanyName: "Example", Name: "Standard", IsActive: true,
				Settings: models.QuotationTemplateSettings{PrimaryColor: "red"}},
		},
		{
			name: "unsupported font",
			input: QuotationTemplateInput{CompanyName: "Example", Name: "Standard", IsActive: true,
				Settings: models.QuotationTemplateSettings{FontFamily: "Comic Sans"}},
		},
		{
			name: "invalid layout",
			input: QuotationTemplateInput{CompanyName: "Example", Name: "Standard", IsActive: true,
				Settings: models.QuotationTemplateSettings{Layout: "freeform"}},
		},
		{
			name:  "inactive default",
			input: QuotationTemplateInput{CompanyName: "Example", Name: "Standard", IsDefault: true},
		},
		{
			name: "invalid company email",
			input: QuotationTemplateInput{CompanyName: "Example", Name: "Standard", IsActive: true,
				Settings: models.QuotationTemplateSettings{CompanyEmail: "not-an-email"}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := templateFromInput(test.input); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("templateFromInput error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestTemplateLogoURLUsesAnImmutableAssetName(t *testing.T) {
	template := models.QuotationTemplate{
		ID: 9, LogoPath: `data/template-logos/0123456789abcdef0123456789abcdef.png`,
	}
	if got, want := templateLogoURL(template), "/api/v1/template-logos/0123456789abcdef0123456789abcdef.png"; got != want {
		t.Fatalf("templateLogoURL = %q, want %q", got, want)
	}
}
