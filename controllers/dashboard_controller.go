package controllers

import (
	"net/http"

	"quotation/services"

	"github.com/gin-gonic/gin"
)

type DashboardController struct {
	quotations services.QuotationService
	customers  services.CustomerService
}

func NewDashboardController(quotations services.QuotationService, customers services.CustomerService) *DashboardController {
	return &DashboardController{quotations: quotations, customers: customers}
}

func (h *DashboardController) Get(c *gin.Context) {
	stats, err := h.quotations.Stats(c.Request.Context())
	if err != nil {
		handleServiceError(c, err)
		return
	}
	recent, _, err := h.quotations.List(c.Request.Context(), "", "", 1, 5)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	_, customerCount, err := h.customers.List(c.Request.Context(), "", 1, 1)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	stats["customers"] = customerCount
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"stats": stats, "recent_quotations": recent}})
}
