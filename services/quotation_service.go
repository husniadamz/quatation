package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"quotation/models"
	"quotation/repositories"

	"gorm.io/gorm"
)

type QuotationItemInput struct {
	Description string
	Quantity    float64
	UnitPrice   float64
}

type QuotationInput struct {
	CustomerID uint
	TaxRate    float64
	Notes      string
	ValidUntil *time.Time
	Items      []QuotationItemInput
}

type QuotationService interface {
	Create(context.Context, QuotationInput) (*models.Quotation, error)
	Get(context.Context, uint) (*models.Quotation, error)
	List(context.Context, int, int) ([]models.Quotation, int64, error)
	Update(context.Context, uint, QuotationInput) (*models.Quotation, error)
	UpdateStatus(context.Context, uint, string) (*models.Quotation, error)
	Delete(context.Context, uint) error
}

type quotationService struct {
	repo         repositories.QuotationRepository
	customerRepo repositories.CustomerRepository
}

const (
	maxQuantity = 99999999.99
	maxMoney    = 9999999999.99
)

func NewQuotationService(repo repositories.QuotationRepository, customerRepo repositories.CustomerRepository) QuotationService {
	return &quotationService{repo: repo, customerRepo: customerRepo}
}

func (s *quotationService) Create(ctx context.Context, input QuotationInput) (*models.Quotation, error) {
	if err := validateQuotationInput(input); err != nil {
		return nil, err
	}
	if _, err := s.customerRepo.Get(ctx, input.CustomerID); errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidInput
	} else if err != nil {
		return nil, err
	}

	number, err := newQuotationNumber()
	if err != nil {
		return nil, fmt.Errorf("generate quotation number: %w", err)
	}
	quotation := &models.Quotation{
		Number: number, CustomerID: input.CustomerID, Status: models.StatusDraft,
		TaxRate: roundMoney(input.TaxRate), Notes: strings.TrimSpace(input.Notes), ValidUntil: input.ValidUntil,
		Items: makeItems(input.Items),
	}
	calculateTotals(quotation)
	if err := s.repo.Create(ctx, quotation); err != nil {
		return nil, err
	}
	return s.Get(ctx, quotation.ID)
}

func (s *quotationService) Get(ctx context.Context, id uint) (*models.Quotation, error) {
	quotation, err := s.repo.Get(ctx, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return quotation, err
}

func (s *quotationService) List(ctx context.Context, page, pageSize int) ([]models.Quotation, int64, error) {
	return s.repo.List(ctx, (page-1)*pageSize, pageSize)
}

func (s *quotationService) Update(ctx context.Context, id uint, input QuotationInput) (*models.Quotation, error) {
	if err := validateQuotationInput(input); err != nil {
		return nil, err
	}
	quotation, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, err := s.customerRepo.Get(ctx, input.CustomerID); errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidInput
	} else if err != nil {
		return nil, err
	}
	quotation.CustomerID = input.CustomerID
	quotation.TaxRate = roundMoney(input.TaxRate)
	quotation.Notes = strings.TrimSpace(input.Notes)
	quotation.ValidUntil = input.ValidUntil
	quotation.Items = makeItems(input.Items)
	calculateTotals(quotation)
	if err := s.repo.Update(ctx, quotation); errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *quotationService) UpdateStatus(ctx context.Context, id uint, status string) (*models.Quotation, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case models.StatusDraft, models.StatusSent, models.StatusAccepted, models.StatusRejected, models.StatusExpired:
	default:
		return nil, ErrInvalidInput
	}
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateStatus(ctx, id, status); errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

func (s *quotationService) Delete(ctx context.Context, id uint) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	} else {
		return err
	}
}

func validateQuotationInput(input QuotationInput) error {
	if input.CustomerID == 0 || math.IsNaN(input.TaxRate) || math.IsInf(input.TaxRate, 0) ||
		input.TaxRate < 0 || input.TaxRate > 100 || len(input.Items) == 0 {
		return ErrInvalidInput
	}
	var subtotal float64
	for _, item := range input.Items {
		quantity := roundMoney(item.Quantity)
		unitPrice := roundMoney(item.UnitPrice)
		if strings.TrimSpace(item.Description) == "" || item.Quantity <= 0 || quantity <= 0 ||
			item.Quantity > maxQuantity || item.UnitPrice < 0 || item.UnitPrice > maxMoney ||
			math.IsNaN(item.Quantity) || math.IsInf(item.Quantity, 0) ||
			math.IsNaN(item.UnitPrice) || math.IsInf(item.UnitPrice, 0) ||
			quantity*unitPrice > maxMoney {
			return ErrInvalidInput
		}
		subtotal += roundMoney(quantity * unitPrice)
		if subtotal > maxMoney {
			return ErrInvalidInput
		}
	}
	tax := roundMoney(subtotal * input.TaxRate / 100)
	if roundMoney(subtotal+tax) > maxMoney {
		return ErrInvalidInput
	}
	return nil
}

func makeItems(inputs []QuotationItemInput) []models.QuotationItem {
	items := make([]models.QuotationItem, 0, len(inputs))
	for _, input := range inputs {
		quantity := roundMoney(input.Quantity)
		unitPrice := roundMoney(input.UnitPrice)
		items = append(items, models.QuotationItem{
			Description: strings.TrimSpace(input.Description),
			Quantity:    quantity,
			UnitPrice:   unitPrice,
			LineTotal:   roundMoney(quantity * unitPrice),
		})
	}
	return items
}

func calculateTotals(quotation *models.Quotation) {
	var subtotal float64
	for i := range quotation.Items {
		quotation.Items[i].LineTotal = roundMoney(quotation.Items[i].Quantity * quotation.Items[i].UnitPrice)
		subtotal += quotation.Items[i].LineTotal
	}
	quotation.Subtotal = roundMoney(subtotal)
	quotation.TaxAmount = roundMoney(quotation.Subtotal * quotation.TaxRate / 100)
	quotation.GrandTotal = roundMoney(quotation.Subtotal + quotation.TaxAmount)
}

func roundMoney(amount float64) float64 {
	return math.Round(amount*100) / 100
}

func newQuotationNumber() (string, error) {
	random := make([]byte, 4)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return "QT-" + time.Now().UTC().Format("20060102-150405") + "-" + hex.EncodeToString(random), nil
}
