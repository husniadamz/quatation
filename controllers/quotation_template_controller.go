package controllers

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"quotation/models"
	"quotation/services"

	"github.com/gin-gonic/gin"
)

type QuotationTemplateController struct {
	service   services.QuotationTemplateService
	companies services.CompanyService
}

func NewQuotationTemplateController(service services.QuotationTemplateService, companies services.CompanyService) *QuotationTemplateController {
	return &QuotationTemplateController{service: service, companies: companies}
}

type quotationTemplateRequest struct {
	CompanyID   uint                             `json:"company_id" binding:"required"`
	CompanyName string                           `json:"company_name"`
	Name        string                           `json:"name" binding:"required,max=120"`
	Settings    models.QuotationTemplateSettings `json:"settings"`
	IsActive    *bool                            `json:"is_active"`
	IsDefault   bool                             `json:"is_default"`
}

type templateActiveRequest struct {
	IsActive *bool `json:"is_active" binding:"required"`
}

func (request quotationTemplateRequest) input(company models.Company) services.QuotationTemplateInput {
	active := true
	if request.IsActive != nil {
		active = *request.IsActive
	}
	return services.QuotationTemplateInput{
		CompanyID:   company.ID,
		CompanyName: company.Name,
		Name:        request.Name,
		Settings:    request.Settings,
		IsActive:    active,
		IsDefault:   request.IsDefault,
	}
}

func (h *QuotationTemplateController) List(c *gin.Context) {
	activeOnly := currentRole(c) != models.RoleAdmin || c.Query("active_only") == "true"
	var companyID uint
	if currentRole(c) != models.RoleAdmin {
		user, ok := currentUser(c)
		if !ok || user.CompanyID == nil {
			writeError(c, http.StatusForbidden, "your account is not assigned to a company")
			return
		}
		companyID = *user.CompanyID
	} else if value := c.Query("company_id"); value != "" {
		parsed, err := strconv.ParseUint(value, 10, 32)
		if err != nil || parsed == 0 {
			writeError(c, http.StatusBadRequest, "invalid company ID")
			return
		}
		companyID = uint(parsed)
	}
	templates, err := h.service.List(c.Request.Context(), companyID, activeOnly)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": templates})
}

func (h *QuotationTemplateController) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	template, err := h.getByRole(c, id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": template})
}

func (h *QuotationTemplateController) Create(c *gin.Context) {
	var request quotationTemplateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid quotation template")
		return
	}
	company, ok := h.requestCompany(c, request.CompanyID)
	if !ok {
		return
	}
	template, err := h.service.Create(c.Request.Context(), request.input(company))
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": template})
}

func (h *QuotationTemplateController) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request quotationTemplateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid quotation template")
		return
	}
	if !h.authorizeTemplate(c, id) {
		return
	}
	company, ok := h.requestCompany(c, request.CompanyID)
	if !ok {
		return
	}
	template, err := h.service.Update(c.Request.Context(), id, request.input(company))
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": template})
}

func (h *QuotationTemplateController) SetActive(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request templateActiveRequest
	if err := c.ShouldBindJSON(&request); err != nil || request.IsActive == nil {
		writeError(c, http.StatusBadRequest, "is_active is required")
		return
	}
	if !h.authorizeTemplate(c, id) {
		return
	}
	template, err := h.service.SetActive(c.Request.Context(), id, *request.IsActive)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": template})
}

func (h *QuotationTemplateController) SetDefault(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.authorizeTemplate(c, id) {
		return
	}
	template, err := h.service.SetDefault(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": template})
}

func (h *QuotationTemplateController) UploadLogo(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.authorizeTemplate(c, id) {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, services.MaxTemplateLogoSize+(1<<20))
	file, header, err := c.Request.FormFile("logo")
	if err != nil {
		writeError(c, http.StatusBadRequest, "select a PNG or JPEG logo")
		return
	}
	defer file.Close()
	if header.Size < 1 || header.Size > services.MaxTemplateLogoSize {
		writeError(c, http.StatusBadRequest, "logo must be no larger than 2 MB")
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, services.MaxTemplateLogoSize+1))
	if err != nil {
		handleServiceError(c, err)
		return
	}
	template, err := h.service.UploadLogo(c.Request.Context(), id, header.Filename, data)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": template})
}

func (h *QuotationTemplateController) LogoAsset(c *gin.Context) {
	data, contentType, err := h.service.ReadLogoAsset(c.Request.Context(), c.Param("filename"))
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private, max-age=3600")
	c.Data(http.StatusOK, contentType, data)
}

func (h *QuotationTemplateController) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.authorizeTemplate(c, id) {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		handleServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *QuotationTemplateController) getByRole(c *gin.Context, id uint) (*models.QuotationTemplate, error) {
	template, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		return nil, err
	}
	if !h.templateBelongsToUser(c, template) {
		return nil, services.ErrNotFound
	}
	if currentRole(c) != models.RoleAdmin && !template.IsActive {
		return nil, services.ErrTemplateInactive
	}
	return template, nil
}

func (h *QuotationTemplateController) requestCompany(c *gin.Context, companyID uint) (models.Company, bool) {
	user, ok := currentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "authentication required")
		return models.Company{}, false
	}
	if currentRole(c) != models.RoleAdmin && (user.CompanyID == nil || *user.CompanyID != companyID) {
		writeError(c, http.StatusForbidden, "templates can only be assigned to your company")
		return models.Company{}, false
	}
	company, err := h.companies.Get(c.Request.Context(), companyID)
	if err != nil {
		handleServiceError(c, err)
		return models.Company{}, false
	}
	return *company, true
}

func (h *QuotationTemplateController) authorizeTemplate(c *gin.Context, id uint) bool {
	template, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return false
	}
	if !h.templateBelongsToUser(c, template) {
		writeError(c, http.StatusNotFound, "resource not found")
		return false
	}
	return true
}

func (h *QuotationTemplateController) templateBelongsToUser(c *gin.Context, template *models.QuotationTemplate) bool {
	if currentRole(c) == models.RoleAdmin {
		return true
	}
	user, ok := currentUser(c)
	return ok && user.CompanyID != nil && template.CompanyID == *user.CompanyID
}

func currentRole(c *gin.Context) string {
	user, ok := currentUser(c)
	if !ok {
		return ""
	}
	if user.Role == models.RoleUser {
		return models.RoleSalesExecutive
	}
	return strings.ToLower(user.Role)
}
