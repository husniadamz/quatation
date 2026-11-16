package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"quotation/models"
	"quotation/repositories"
)

var ErrTemplateInactive = errors.New("template is inactive")
var ErrInvalidLogo = errors.New("invalid logo image")

const MaxTemplateLogoSize = 2 << 20

var templateColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
var logoAssetPattern = regexp.MustCompile(`^[a-f0-9]{32}\.(png|jpg)$`)

type QuotationTemplateInput struct {
	CompanyName string
	Name        string
	Settings    models.QuotationTemplateSettings
	IsActive    bool
	IsDefault   bool
}

type QuotationTemplateService interface {
	List(context.Context, string, bool) ([]models.QuotationTemplate, error)
	Get(context.Context, uint) (*models.QuotationTemplate, error)
	GetActive(context.Context, uint) (*models.QuotationTemplate, error)
	Create(context.Context, QuotationTemplateInput) (*models.QuotationTemplate, error)
	Update(context.Context, uint, QuotationTemplateInput) (*models.QuotationTemplate, error)
	SetActive(context.Context, uint, bool) (*models.QuotationTemplate, error)
	SetDefault(context.Context, uint) (*models.QuotationTemplate, error)
	UploadLogo(context.Context, uint, string, []byte) (*models.QuotationTemplate, error)
	ReadLogoAsset(context.Context, string) ([]byte, string, error)
	Delete(context.Context, uint) error
}

type quotationTemplateService struct {
	repo    repositories.QuotationTemplateRepository
	logoDir string
}

func NewQuotationTemplateService(repo repositories.QuotationTemplateRepository, logoDir string) QuotationTemplateService {
	return &quotationTemplateService{repo: repo, logoDir: logoDir}
}

func (s *quotationTemplateService) List(ctx context.Context, companyName string, activeOnly bool) ([]models.QuotationTemplate, error) {
	templates, err := s.repo.List(ctx, companyName, activeOnly)
	if err != nil {
		return nil, err
	}
	for i := range templates {
		templates[i].LogoURL = templateLogoURL(templates[i])
	}
	return templates, nil
}

func (s *quotationTemplateService) Get(ctx context.Context, id uint) (*models.QuotationTemplate, error) {
	template, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, mapTemplateError(err)
	}
	template.LogoURL = templateLogoURL(*template)
	return template, nil
}

func (s *quotationTemplateService) GetActive(ctx context.Context, id uint) (*models.QuotationTemplate, error) {
	template, err := s.repo.GetActive(ctx, id)
	if err != nil {
		return nil, mapTemplateError(err)
	}
	template.LogoURL = templateLogoURL(*template)
	return template, nil
}

func (s *quotationTemplateService) Create(ctx context.Context, input QuotationTemplateInput) (*models.QuotationTemplate, error) {
	template, err := templateFromInput(input)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, template); err != nil {
		return nil, mapTemplateError(err)
	}
	return s.Get(ctx, template.ID)
}

func (s *quotationTemplateService) Update(ctx context.Context, id uint, input QuotationTemplateInput) (*models.QuotationTemplate, error) {
	template, err := templateFromInput(input)
	if err != nil {
		return nil, err
	}
	existing, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if !template.IsActive && template.IsDefault {
		return nil, ErrInvalidInput
	}
	template.ID = id
	template.LogoPath = existing.LogoPath
	template.LogoContentType = existing.LogoContentType
	template.LogoOriginalName = existing.LogoOriginalName
	if err := s.repo.Update(ctx, template); err != nil {
		return nil, mapTemplateError(err)
	}
	return s.Get(ctx, id)
}

func (s *quotationTemplateService) SetActive(ctx context.Context, id uint, active bool) (*models.QuotationTemplate, error) {
	if err := s.repo.SetActive(ctx, id, active); err != nil {
		return nil, mapTemplateError(err)
	}
	return s.Get(ctx, id)
}

func (s *quotationTemplateService) SetDefault(ctx context.Context, id uint) (*models.QuotationTemplate, error) {
	if err := s.repo.SetDefault(ctx, id); err != nil {
		return nil, mapTemplateError(err)
	}
	return s.Get(ctx, id)
}

func (s *quotationTemplateService) UploadLogo(ctx context.Context, id uint, originalName string, data []byte) (*models.QuotationTemplate, error) {
	if len(data) == 0 || len(data) > MaxTemplateLogoSize {
		return nil, ErrInvalidLogo
	}
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}
	contentType := http.DetectContentType(data)
	imageConfig, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || imageConfig.Width < 1 || imageConfig.Height < 1 ||
		imageConfig.Width > 4000 || imageConfig.Height > 4000 || int64(imageConfig.Width)*int64(imageConfig.Height) > 8_000_000 {
		return nil, ErrInvalidLogo
	}
	extension, ok := map[string]string{"png": ".png", "jpeg": ".jpg"}[format]
	expectedType, supported := map[string]string{"png": "image/png", "jpeg": "image/jpeg"}[format]
	if !ok || !supported || contentType != expectedType {
		return nil, ErrInvalidLogo
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		return nil, fmt.Errorf("generate logo filename: %w", err)
	}
	if err := os.MkdirAll(s.logoDir, 0750); err != nil {
		return nil, fmt.Errorf("create template logo directory: %w", err)
	}
	filename := hex.EncodeToString(random) + extension
	fullPath := filepath.Join(s.logoDir, filename)
	if err := os.WriteFile(fullPath, data, 0600); err != nil {
		return nil, fmt.Errorf("save template logo: %w", err)
	}
	if err := s.repo.UpdateLogo(ctx, id, fullPath, expectedType, filepath.Base(filepath.Clean(originalName))); err != nil {
		if removeErr := os.Remove(fullPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			log.Printf("remove unassociated template logo %s: %v", filename, removeErr)
		}
		return nil, mapTemplateError(err)
	}
	return s.Get(ctx, id)
}

func (s *quotationTemplateService) ReadLogoAsset(ctx context.Context, name string) ([]byte, string, error) {
	if filepath.Base(name) != name || !logoAssetPattern.MatchString(name) {
		return nil, "", ErrNotFound
	}
	data, err := os.ReadFile(filepath.Join(s.logoDir, name))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, "", ErrNotFound
		}
		return nil, "", fmt.Errorf("read template logo: %w", err)
	}
	contentType := "image/png"
	if strings.HasSuffix(name, ".jpg") {
		contentType = "image/jpeg"
	}
	return data, contentType, nil
}

func (s *quotationTemplateService) Delete(ctx context.Context, id uint) error {
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return mapTemplateError(err)
	}
	return nil
}

func templateFromInput(input QuotationTemplateInput) (*models.QuotationTemplate, error) {
	input.CompanyName = strings.TrimSpace(input.CompanyName)
	input.Name = strings.TrimSpace(input.Name)
	if input.CompanyName == "" || len(input.CompanyName) > 180 ||
		input.Name == "" || len(input.Name) > 120 {
		return nil, ErrInvalidInput
	}
	settings := input.Settings
	settings.CompanyEmail = strings.TrimSpace(settings.CompanyEmail)
	settings.CompanyPhone = strings.TrimSpace(settings.CompanyPhone)
	settings.CompanyAddress = strings.TrimSpace(settings.CompanyAddress)
	settings.HeaderTitle = strings.TrimSpace(settings.HeaderTitle)
	settings.HeaderText = strings.TrimSpace(settings.HeaderText)
	settings.FooterText = strings.TrimSpace(settings.FooterText)
	settings.Terms = strings.TrimSpace(settings.Terms)
	settings.PaymentDetails = strings.TrimSpace(settings.PaymentDetails)
	settings.SignatureLabel = strings.TrimSpace(settings.SignatureLabel)
	settings.PrimaryColor = strings.ToUpper(strings.TrimSpace(settings.PrimaryColor))
	settings.AccentColor = strings.ToUpper(strings.TrimSpace(settings.AccentColor))
	settings.FontFamily = strings.TrimSpace(settings.FontFamily)
	settings.Layout = strings.ToLower(strings.TrimSpace(settings.Layout))

	if settings.HeaderTitle == "" {
		settings.HeaderTitle = "Quotation"
	}
	if settings.PrimaryColor == "" {
		settings.PrimaryColor = "#264A73"
	}
	if settings.AccentColor == "" {
		settings.AccentColor = "#D6A94B"
	}
	if settings.FontFamily == "" {
		settings.FontFamily = "Arial"
	}
	if settings.Layout == "" {
		settings.Layout = "portrait"
	}
	if !templateColorPattern.MatchString(settings.PrimaryColor) ||
		!templateColorPattern.MatchString(settings.AccentColor) ||
		!containsString([]string{"Arial", "Helvetica", "Times", "Courier"}, settings.FontFamily) ||
		!containsString([]string{"portrait", "landscape"}, settings.Layout) ||
		len(settings.CompanyEmail) > 255 || len(settings.CompanyPhone) > 60 ||
		len(settings.CompanyAddress) > 2000 || len(settings.HeaderTitle) > 120 ||
		len(settings.HeaderText) > 2000 || len(settings.FooterText) > 2000 ||
		len(settings.Terms) > 10000 || len(settings.PaymentDetails) > 4000 ||
		len(settings.SignatureLabel) > 120 ||
		(settings.CompanyEmail != "" && !validEmail(settings.CompanyEmail)) {
		return nil, ErrInvalidInput
	}
	if input.IsDefault && !input.IsActive {
		return nil, ErrInvalidInput
	}
	return &models.QuotationTemplate{
		CompanyName: input.CompanyName,
		Name:        input.Name,
		Settings:    settings,
		IsActive:    input.IsActive,
		IsDefault:   input.IsDefault,
	}, nil
}

func templateLogoURL(template models.QuotationTemplate) string {
	if template.LogoPath == "" {
		return ""
	}
	return fmt.Sprintf("/api/v1/template-logos/%s", filepath.Base(template.LogoPath))
}

func mapTemplateError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrTemplateInactive) || errors.Is(err, repositories.ErrTemplateInactive) {
		return ErrTemplateInactive
	}
	if errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if errors.Is(err, os.ErrNotExist) {
		return ErrNotFound
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "duplicate") || strings.Contains(message, "unique constraint") {
		return ErrConflict
	}
	return err
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
