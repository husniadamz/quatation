package services

import (
	"context"
	"errors"
	"strings"

	"quotation/models"
	"quotation/repositories"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("resource not found")
var ErrInvalidInput = errors.New("invalid input")

type CustomerInput struct {
	Name    string
	Email   string
	Phone   string
	Address string
}

type CustomerService interface {
	Create(context.Context, CustomerInput) (*models.Customer, error)
	Get(context.Context, uint) (*models.Customer, error)
	List(context.Context, int, int) ([]models.Customer, int64, error)
	Update(context.Context, uint, CustomerInput) (*models.Customer, error)
	Delete(context.Context, uint) error
}

type customerService struct {
	repo repositories.CustomerRepository
}

func NewCustomerService(repo repositories.CustomerRepository) CustomerService {
	return &customerService{repo: repo}
}

func (s *customerService) Create(ctx context.Context, input CustomerInput) (*models.Customer, error) {
	customer := customerFromInput(input)
	if err := s.repo.Create(ctx, customer); err != nil {
		return nil, err
	}
	return customer, nil
}

func (s *customerService) Get(ctx context.Context, id uint) (*models.Customer, error) {
	customer, err := s.repo.Get(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return customer, err
}

func (s *customerService) List(ctx context.Context, page, pageSize int) ([]models.Customer, int64, error) {
	return s.repo.List(ctx, (page-1)*pageSize, pageSize)
}

func (s *customerService) Update(ctx context.Context, id uint, input CustomerInput) (*models.Customer, error) {
	_, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	updated := customerFromInput(input)
	updated.ID = id
	if err := s.repo.Update(ctx, updated); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *customerService) Delete(ctx context.Context, id uint) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	} else {
		return err
	}
}

func customerFromInput(input CustomerInput) *models.Customer {
	return &models.Customer{
		Name: strings.TrimSpace(input.Name), Email: strings.TrimSpace(input.Email),
		Phone: strings.TrimSpace(input.Phone), Address: strings.TrimSpace(input.Address),
	}
}
