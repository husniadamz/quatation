package repositories

import (
	"context"
	"errors"

	"quotation/models"

	"gorm.io/gorm"
)

var ErrTemplateInactive = errors.New("template is inactive")

type QuotationTemplateRepository interface {
	EnsureStarterTemplate(context.Context) error
	Create(context.Context, *models.QuotationTemplate) error
	Get(context.Context, uint) (*models.QuotationTemplate, error)
	GetActive(context.Context, uint) (*models.QuotationTemplate, error)
	List(context.Context, uint, bool) ([]models.QuotationTemplate, error)
	Update(context.Context, *models.QuotationTemplate) error
	SetActive(context.Context, uint, bool) error
	SetDefault(context.Context, uint) error
	UpdateLogo(context.Context, uint, string, string, string) error
	Delete(context.Context, uint) error
}

type quotationTemplateRepository struct{ db *gorm.DB }

func NewQuotationTemplateRepository(db *gorm.DB) QuotationTemplateRepository {
	return &quotationTemplateRepository{db: db}
}

func (r *quotationTemplateRepository) EnsureStarterTemplate(ctx context.Context) error {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.QuotationTemplate{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	template := models.QuotationTemplate{
		CompanyName: "Your Company",
		Name:        "Standard Quotation",
		Settings: models.QuotationTemplateSettings{
			HeaderTitle:  "Quotation",
			HeaderText:   "We are pleased to provide the following quotation.",
			FooterText:   "Thank you for your business.",
			PrimaryColor: "#264A73",
			AccentColor:  "#D6A94B",
			FontFamily:   "Arial",
			Layout:       "portrait",
		},
		IsActive:  true,
		IsDefault: true,
	}
	return r.db.WithContext(ctx).Create(&template).Error
}

func (r *quotationTemplateRepository) Create(ctx context.Context, template *models.QuotationTemplate) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if template.IsDefault {
			if err := tx.Model(&models.QuotationTemplate{}).
				Where("company_name = ?", template.CompanyName).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(template).Error
	})
}

func (r *quotationTemplateRepository) Get(ctx context.Context, id uint) (*models.QuotationTemplate, error) {
	var template models.QuotationTemplate
	if err := r.db.WithContext(ctx).First(&template, id).Error; err != nil {
		return nil, err
	}
	return &template, nil
}

func (r *quotationTemplateRepository) GetActive(ctx context.Context, id uint) (*models.QuotationTemplate, error) {
	template, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !template.IsActive {
		return nil, ErrTemplateInactive
	}
	return template, nil
}

func (r *quotationTemplateRepository) List(ctx context.Context, companyID uint, activeOnly bool) ([]models.QuotationTemplate, error) {
	var templates []models.QuotationTemplate
	query := r.db.WithContext(ctx).Model(&models.QuotationTemplate{})
	if companyID != 0 {
		query = query.Where("company_id = ?", companyID)
	}
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}
	err := query.Order("company_name ASC").Order("is_default DESC").Order("name ASC").Find(&templates).Error
	return templates, err
}

func (r *quotationTemplateRepository) Update(ctx context.Context, template *models.QuotationTemplate) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if template.IsDefault {
			if err := tx.Model(&models.QuotationTemplate{}).
				Where("company_name = ? AND id <> ?", template.CompanyName, template.ID).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		result := tx.Model(&models.QuotationTemplate{}).Where("id = ?", template.ID).Updates(map[string]interface{}{
			"company_name": template.CompanyName,
			"name":         template.Name,
			"settings":     template.Settings,
			"is_active":    template.IsActive,
			"is_default":   template.IsDefault,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *quotationTemplateRepository) SetActive(ctx context.Context, id uint, active bool) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var template models.QuotationTemplate
		if err := tx.First(&template, id).Error; err != nil {
			return err
		}
		updates := map[string]interface{}{"is_active": active}
		if !active {
			updates["is_default"] = false
		}
		return tx.Model(&template).Updates(updates).Error
	})
}

func (r *quotationTemplateRepository) SetDefault(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var template models.QuotationTemplate
		if err := tx.First(&template, id).Error; err != nil {
			return err
		}
		if !template.IsActive {
			return ErrTemplateInactive
		}
		if err := tx.Model(&models.QuotationTemplate{}).
			Where("company_name = ?", template.CompanyName).
			Update("is_default", false).Error; err != nil {
			return err
		}
		return tx.Model(&template).Update("is_default", true).Error
	})
}

func (r *quotationTemplateRepository) UpdateLogo(ctx context.Context, id uint, filePath, contentType, originalName string) error {
	result := r.db.WithContext(ctx).Model(&models.QuotationTemplate{}).Where("id = ?", id).Updates(map[string]interface{}{
		"logo_path": filePath, "logo_content_type": contentType, "logo_original_name": originalName,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *quotationTemplateRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.QuotationTemplate{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
