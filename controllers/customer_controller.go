package controllers

import (
	"net/http"

	"quotation/services"

	"github.com/gin-gonic/gin"
)

type CustomerController struct{ service services.CustomerService }

func NewCustomerController(service services.CustomerService) *CustomerController {
	return &CustomerController{service: service}
}

type customerRequest struct {
	Name    string `json:"name" binding:"required,max=150"`
	Email   string `json:"email" binding:"omitempty,email,max=255"`
	Phone   string `json:"phone" binding:"max=40"`
	Address string `json:"address" binding:"max=5000"`
}

func (r customerRequest) input() services.CustomerInput {
	return services.CustomerInput{Name: r.Name, Email: r.Email, Phone: r.Phone, Address: r.Address}
}

func (h *CustomerController) Create(c *gin.Context) {
	var request customerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid customer data")
		return
	}
	customer, err := h.service.Create(c.Request.Context(), request.input())
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": customer})
}

func (h *CustomerController) List(c *gin.Context) {
	page, pageSize, ok := pagination(c)
	if !ok {
		return
	}
	customers, total, err := h.service.List(c.Request.Context(), page, pageSize)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": customers, "meta": gin.H{"page": page, "page_size": pageSize, "total": total}})
}

func (h *CustomerController) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	customer, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": customer})
}

func (h *CustomerController) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request customerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid customer data")
		return
	}
	customer, err := h.service.Update(c.Request.Context(), id, request.input())
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": customer})
}

func (h *CustomerController) Delete(c *gin.Context) {
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
