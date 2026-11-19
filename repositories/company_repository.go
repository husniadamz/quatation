package repositories

import (
	"context"
	"errors"

	"quotation/models"

	"gorm.io/gorm"
)

var ErrCompanyInUse = errors.New("company is assigned to users")

type CompanyRepository interface {
	List(context.Context) ([]models.Company, error)
	Get(context.Context, uint) (*models.Company, error)
	ListMembers(context.Context, uint) ([]models.User, error)
	Create(context.Context, *models.Company) error
	Update(context.Context, *models.Company) error
	Delete(context.Context, uint) error
}

type companyRepository struct{ db *gorm.DB }

func NewCompanyRepository(db *gorm.DB) CompanyRepository {
	return &companyRepository{db: db}
}

func (r *companyRepository) List(ctx context.Context) ([]models.Company, error) {
	var companies []models.Company
	err := r.db.WithContext(ctx).Order("name ASC").Find(&companies).Error
	return companies, err
}

func (r *companyRepository) Get(ctx context.Context, id uint) (*models.Company, error) {
	var company models.Company
	if err := r.db.WithContext(ctx).First(&company, id).Error; err != nil {
		return nil, err
	}
	return &company, nil
}

func (r *companyRepository) ListMembers(ctx context.Context, companyID uint) ([]models.User, error) {
	var users []models.User
	err := r.db.WithContext(ctx).Where("company_id = ?", companyID).Order("name ASC").Find(&users).Error
	return users, err
}

func (r *companyRepository) Create(ctx context.Context, company *models.Company) error {
	return r.db.WithContext(ctx).Create(company).Error
}

func (r *companyRepository) Update(ctx context.Context, company *models.Company) error {
	return r.db.WithContext(ctx).Model(company).Updates(map[string]interface{}{
		"name": company.Name, "email": company.Email, "phone": company.Phone, "address": company.Address,
	}).Error
}

func (r *companyRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.User{}).Where("company_id = ?", id).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrCompanyInUse
		}
		result := tx.Delete(&models.Company{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}
