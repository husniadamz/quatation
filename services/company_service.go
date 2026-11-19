package services

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"quotation/models"
	"quotation/repositories"

	"gorm.io/gorm"
)

type CompanyInput struct {
	Name    string
	Email   string
	Phone   string
	Address string
}

type CompanyService interface {
	List(context.Context) ([]models.Company, error)
	Get(context.Context, uint) (*models.Company, error)
	ListMembers(context.Context, uint) ([]models.User, error)
	Create(context.Context, CompanyInput) (*models.Company, error)
	Update(context.Context, uint, CompanyInput) (*models.Company, error)
	Delete(context.Context, uint) error
}

type companyService struct {
	repo repositories.CompanyRepository
}

func NewCompanyService(repo repositories.CompanyRepository) CompanyService {
	return &companyService{repo: repo}
}

func (s *companyService) List(ctx context.Context) ([]models.Company, error) {
	return s.repo.List(ctx)
}

func (s *companyService) Get(ctx context.Context, id uint) (*models.Company, error) {
	company, err := s.repo.Get(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return company, err
}

func (s *companyService) ListMembers(ctx context.Context, id uint) ([]models.User, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}
	return s.repo.ListMembers(ctx, id)
}

func (s *companyService) Create(ctx context.Context, input CompanyInput) (*models.Company, error) {
	company, err := companyFromInput(input)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, company); err != nil {
		return nil, mapUniqueError(err)
	}
	return company, nil
}

func (s *companyService) Update(ctx context.Context, id uint, input CompanyInput) (*models.Company, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}
	company, err := companyFromInput(input)
	if err != nil {
		return nil, err
	}
	company.ID = id
	if err := s.repo.Update(ctx, company); err != nil {
		return nil, mapUniqueError(err)
	}
	return s.Get(ctx, id)
}

func (s *companyService) Delete(ctx context.Context, id uint) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); errors.Is(err, repositories.ErrCompanyInUse) {
		return ErrConflict
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	} else {
		return err
	}
}

func companyFromInput(input CompanyInput) (*models.Company, error) {
	company := &models.Company{
		Name: strings.TrimSpace(input.Name), Email: strings.ToLower(strings.TrimSpace(input.Email)),
		Phone: strings.TrimSpace(input.Phone), Address: strings.TrimSpace(input.Address),
	}
	if company.Name == "" || len(company.Name) > 180 || len(company.Email) > 255 ||
		len(company.Phone) > 60 || len(company.Address) > 2000 {
		return nil, ErrInvalidInput
	}
	if company.Email != "" {
		address, err := mail.ParseAddress(company.Email)
		if err != nil || address.Address != company.Email {
			return nil, ErrInvalidInput
		}
	}
	return company, nil
}
