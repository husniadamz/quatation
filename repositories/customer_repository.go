package repositories

import (
	"context"

	"quotation/models"

	"gorm.io/gorm"
)

type CustomerRepository interface {
	Create(context.Context, *models.Customer) error
	Get(context.Context, uint) (*models.Customer, error)
	List(context.Context, int, int) ([]models.Customer, int64, error)
	Update(context.Context, *models.Customer) error
	Delete(context.Context, uint) error
}

type customerRepository struct{ db *gorm.DB }

func NewCustomerRepository(db *gorm.DB) CustomerRepository {
	return &customerRepository{db: db}
}

func (r *customerRepository) Create(ctx context.Context, customer *models.Customer) error {
	return r.db.WithContext(ctx).Create(customer).Error
}

func (r *customerRepository) Get(ctx context.Context, id uint) (*models.Customer, error) {
	var customer models.Customer
	if err := r.db.WithContext(ctx).First(&customer, id).Error; err != nil {
		return nil, err
	}
	return &customer, nil
}

func (r *customerRepository) List(ctx context.Context, offset, limit int) ([]models.Customer, int64, error) {
	var customers []models.Customer
	var total int64
	query := r.db.WithContext(ctx).Model(&models.Customer{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&customers).Error
	return customers, total, err
}

func (r *customerRepository) Update(ctx context.Context, customer *models.Customer) error {
	return r.db.WithContext(ctx).Model(customer).Updates(map[string]interface{}{
		"name": customer.Name, "email": customer.Email, "phone": customer.Phone, "address": customer.Address,
	}).Error
}

func (r *customerRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.Customer{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
