package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"quotation/models"
	"quotation/repositories"
)

type templateSnapshotRepository struct{ template models.QuotationTemplate }

func (r templateSnapshotRepository) Create(context.Context, *models.QuotationTemplate) error {
	return nil
}
func (r templateSnapshotRepository) Get(context.Context, uint) (*models.QuotationTemplate, error) {
	return &r.template, nil
}
func (r templateSnapshotRepository) GetActive(context.Context, uint) (*models.QuotationTemplate, error) {
	if !r.template.IsActive {
		return nil, repositories.ErrTemplateInactive
	}
	return &r.template, nil
}
func (r templateSnapshotRepository) List(context.Context, string, bool) ([]models.QuotationTemplate, error) {
	return nil, nil
}
func (r templateSnapshotRepository) Update(context.Context, *models.QuotationTemplate) error {
	return nil
}
func (r templateSnapshotRepository) SetActive(context.Context, uint, bool) error { return nil }
func (r templateSnapshotRepository) SetDefault(context.Context, uint) error      { return nil }
func (r templateSnapshotRepository) UpdateLogo(context.Context, uint, string, string, string) error {
	return nil
}
func (r templateSnapshotRepository) Delete(context.Context, uint) error { return nil }

func TestQuotationTemplateSnapshotCopiesSettingsAndImmutableLogoURL(t *testing.T) {
	repo := templateSnapshotRepository{template: models.QuotationTemplate{
		ID: 42, CompanyName: "Example Company", Name: "Branded", IsActive: true,
		LogoPath:        `data/template-logos/0123456789abcdef0123456789abcdef.png`,
		LogoContentType: "image/png",
		Settings: models.QuotationTemplateSettings{
			HeaderTitle: "Proposal", PrimaryColor: "#123456", Terms: "Original terms",
		},
	}}
	service := quotationService{templateRepo: repo}
	templateID := uint(42)
	id, encoded, err := service.templateSnapshot(context.Background(), &templateID)
	if err != nil {
		t.Fatal(err)
	}
	if id == nil || *id != 42 {
		t.Fatalf("template id = %v, want 42", id)
	}
	if !strings.Contains(encoded, `"/api/v1/template-logos/0123456789abcdef0123456789abcdef.png"`) ||
		strings.Contains(encoded, `data/template-logos`) {
		t.Fatalf("snapshot did not preserve a safe, immutable logo URL: %s", encoded)
	}
	if !strings.Contains(encoded, `"terms":"Original terms"`) ||
		!strings.Contains(encoded, `"primary_color":"#123456"`) {
		t.Fatalf("snapshot omitted template settings: %s", encoded)
	}
}

func TestQuotationTemplateSnapshotRejectsInactiveTemplate(t *testing.T) {
	repo := templateSnapshotRepository{template: models.QuotationTemplate{ID: 2, IsActive: false}}
	service := quotationService{templateRepo: repo}
	id := uint(2)
	if _, _, err := service.templateSnapshot(context.Background(), &id); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("templateSnapshot error = %v, want ErrInvalidInput", err)
	}
}
