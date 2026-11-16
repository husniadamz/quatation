package controllers

import (
	"io"
	"net/http"
	"strings"

	"quotation/models"
	"quotation/services"

	"github.com/gin-gonic/gin"
)

type QuotationTemplateController struct {
	service services.QuotationTemplateService
}

func NewQuotationTemplateController(service services.QuotationTemplateService) *QuotationTemplateController {
	return &QuotationTemplateController{service: service}
}

type quotationTemplateRequest struct {
	CompanyName string                           `json:"company_name" binding:"required,max=180"`
	Name        string                           `json:"name" binding:"required,max=120"`
	Settings    models.QuotationTemplateSettings `json:"settings"`
	IsActive    *bool                            `json:"is_active"`
	IsDefault   bool                             `json:"is_default"`
}

type templateActiveRequest struct {
	IsActive *bool `json:"is_active" binding:"required"`
}

func (request quotationTemplateRequest) input() services.QuotationTemplateInput {
	active := true
	if request.IsActive != nil {
		active = *request.IsActive
	}
	return services.QuotationTemplateInput{
		CompanyName: request.CompanyName,
		Name:        request.Name,
		Settings:    request.Settings,
		IsActive:    active,
		IsDefault:   request.IsDefault,
	}
}

func (h *QuotationTemplateController) List(c *gin.Context) {
	activeOnly := currentRole(c) != models.RoleAdmin || c.Query("active_only") == "true"
	templates, err := h.service.List(c.Request.Context(), c.Query("company_name"), activeOnly)
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
	template, err := h.service.Create(c.Request.Context(), request.input())
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
	template, err := h.service.Update(c.Request.Context(), id, request.input())
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
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		handleServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *QuotationTemplateController) getByRole(c *gin.Context, id uint) (*models.QuotationTemplate, error) {
	if currentRole(c) == models.RoleAdmin {
		return h.service.Get(c.Request.Context(), id)
	}
	return h.service.GetActive(c.Request.Context(), id)
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
