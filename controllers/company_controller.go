package controllers

import (
	"net/http"

	"quotation/services"

	"github.com/gin-gonic/gin"
)

type CompanyController struct{ service services.CompanyService }

func NewCompanyController(service services.CompanyService) *CompanyController {
	return &CompanyController{service: service}
}

type companyRequest struct {
	Name    string `json:"name" binding:"required,max=180"`
	Email   string `json:"email" binding:"omitempty,email,max=255"`
	Phone   string `json:"phone" binding:"max=60"`
	Address string `json:"address" binding:"max=2000"`
}

func (r companyRequest) input() services.CompanyInput {
	return services.CompanyInput{Name: r.Name, Email: r.Email, Phone: r.Phone, Address: r.Address}
}

func (h *CompanyController) List(c *gin.Context) {
	companies, err := h.service.List(c.Request.Context())
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": companies})
}

func (h *CompanyController) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	company, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": company})
}

func (h *CompanyController) ListMembers(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	users, err := h.service.ListMembers(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

func (h *CompanyController) Create(c *gin.Context) {
	var request companyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid company data")
		return
	}
	company, err := h.service.Create(c.Request.Context(), request.input())
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": company})
}

func (h *CompanyController) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request companyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid company data")
		return
	}
	company, err := h.service.Update(c.Request.Context(), id, request.input())
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": company})
}

func (h *CompanyController) Delete(c *gin.Context) {
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
