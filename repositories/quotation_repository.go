package repositories

import (
	"context"

	"quotation/models"

	"gorm.io/gorm"
)

type QuotationRepository interface {
	Create(context.Context, *models.Quotation) error
	Get(context.Context, uint) (*models.Quotation, error)
	List(context.Context, string, string, int, int) ([]models.Quotation, int64, error)
	Stats(context.Context) (map[string]int64, error)
	Update(context.Context, *models.Quotation) error
	UpdateStatus(context.Context, uint, string) error
	Delete(context.Context, uint) error
}

type quotationRepository struct{ db *gorm.DB }

func NewQuotationRepository(db *gorm.DB) QuotationRepository {
	return &quotationRepository{db: db}
}

func (r *quotationRepository) Create(ctx context.Context, quotation *models.Quotation) error {
	return r.db.WithContext(ctx).Create(quotation).Error
}

func (r *quotationRepository) Get(ctx context.Context, id uint) (*models.Quotation, error) {
	var quotation models.Quotation
	err := r.db.WithContext(ctx).Preload("Customer").Preload("Items").First(&quotation, id).Error
	if err != nil {
		return nil, err
	}
	return &quotation, nil
}

func (r *quotationRepository) List(ctx context.Context, search, status string, offset, limit int) ([]models.Quotation, int64, error) {
	var quotations []models.Quotation
	var total int64
	query := r.db.WithContext(ctx).Model(&models.Quotation{})
	if search != "" {
		pattern := "%" + search + "%"
		query = query.Joins("LEFT JOIN customers ON customers.id = quotations.customer_id").
			Where("quotations.number LIKE ? OR customers.name LIKE ?", pattern, pattern)
	}
	if status != "" {
		query = query.Where("quotations.status = ?", status)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Preload("Customer").Preload("Items").Order("quotations.id DESC").Offset(offset).Limit(limit).Find(&quotations).Error
	return quotations, total, err
}

func (r *quotationRepository) Stats(ctx context.Context) (map[string]int64, error) {
	stats := map[string]int64{"total": 0, "pending": 0, "approved": 0, "rejected": 0}
	query := r.db.WithContext(ctx).Model(&models.Quotation{})
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	stats["total"] = total
	for status, target := range map[string]string{
		"approved": "accepted", "rejected": "rejected",
	} {
		var count int64
		if err := r.db.WithContext(ctx).Model(&models.Quotation{}).Where("status = ?", target).Count(&count).Error; err != nil {
			return nil, err
		}
		stats[status] = count
	}
	var pending int64
	if err := r.db.WithContext(ctx).Model(&models.Quotation{}).
		Where("status IN ?", []string{models.StatusDraft, models.StatusSent}).
		Count(&pending).Error; err != nil {
		return nil, err
	}
	stats["pending"] = pending
	return stats, nil
}

func (r *quotationRepository) Update(ctx context.Context, quotation *models.Quotation) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.Quotation{}).Where("id = ?", quotation.ID).Updates(map[string]interface{}{
			"customer_id": quotation.CustomerID, "subtotal": quotation.Subtotal, "tax_rate": quotation.TaxRate,
			"tax_amount": quotation.TaxAmount, "grand_total": quotation.GrandTotal,
			"notes": quotation.Notes, "valid_until": quotation.ValidUntil,
			"template_id": quotation.TemplateID, "template_snapshot": quotation.TemplateSnapshot,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		if err := tx.Where("quotation_id = ?", quotation.ID).Delete(&models.QuotationItem{}).Error; err != nil {
			return err
		}
		for i := range quotation.Items {
			quotation.Items[i].ID = 0
			quotation.Items[i].QuotationID = quotation.ID
			if err := tx.Create(&quotation.Items[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *quotationRepository) UpdateStatus(ctx context.Context, id uint, status string) error {
	result := r.db.WithContext(ctx).Model(&models.Quotation{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *quotationRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.Quotation{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
