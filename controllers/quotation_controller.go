package controllers

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"quotation/models"
	"quotation/services"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
)

type QuotationController struct{ service services.QuotationService }

func NewQuotationController(service services.QuotationService) *QuotationController {
	return &QuotationController{service: service}
}

type quotationItemRequest struct {
	Description string   `json:"description" binding:"required,max=255"`
	Quantity    float64  `json:"quantity" binding:"required,gt=0"`
	UnitPrice   *float64 `json:"unit_price" binding:"required,gte=0"`
}

type quotationRequest struct {
	CustomerID uint                   `json:"customer_id" binding:"required"`
	TaxRate    float64                `json:"tax_rate" binding:"gte=0,lte=100"`
	Notes      string                 `json:"notes" binding:"max=5000"`
	ValidUntil *time.Time             `json:"valid_until"`
	Items      []quotationItemRequest `json:"items" binding:"required,min=1,dive"`
}

type statusRequest struct {
	Status string `json:"status" binding:"required"`
}

func (r quotationRequest) input() services.QuotationInput {
	items := make([]services.QuotationItemInput, 0, len(r.Items))
	for _, item := range r.Items {
		items = append(items, services.QuotationItemInput{
			Description: item.Description, Quantity: item.Quantity, UnitPrice: *item.UnitPrice,
		})
	}
	return services.QuotationInput{
		CustomerID: r.CustomerID, TaxRate: r.TaxRate, Notes: r.Notes,
		ValidUntil: r.ValidUntil, Items: items,
	}
}

func (h *QuotationController) Create(c *gin.Context) {
	var request quotationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid quotation data")
		return
	}
	quotation, err := h.service.Create(c.Request.Context(), request.input())
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": quotation})
}

func (h *QuotationController) List(c *gin.Context) {
	page, pageSize, ok := pagination(c)
	if !ok {
		return
	}
	quotations, total, err := h.service.List(c.Request.Context(), c.Query("search"), c.Query("status"), page, pageSize)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": quotations, "meta": gin.H{"page": page, "page_size": pageSize, "total": total}})
}

func (h *QuotationController) Dashboard(c *gin.Context) {
	stats, err := h.service.Stats(c.Request.Context())
	if err != nil {
		handleServiceError(c, err)
		return
	}
	recent, _, err := h.service.List(c.Request.Context(), "", "", 1, 5)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"stats": stats, "recent_quotations": recent}})
}

func (h *QuotationController) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	quotation, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": quotation})
}

func (h *QuotationController) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request quotationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid quotation data")
		return
	}
	quotation, err := h.service.Update(c.Request.Context(), id, request.input())
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": quotation})
}

func (h *QuotationController) UpdateStatus(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	user, ok := currentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "authentication required")
		return
	}
	var request statusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid status")
		return
	}
	if user.Role == models.RoleManager &&
		request.Status != models.StatusAccepted && request.Status != models.StatusRejected {
		writeError(c, http.StatusForbidden, "managers may only approve or reject quotations")
		return
	}
	if user.Role == models.RoleManager {
		quotation, err := h.service.Get(c.Request.Context(), id)
		if err != nil {
			handleServiceError(c, err)
			return
		}
		if quotation.Status != models.StatusDraft && quotation.Status != models.StatusSent {
			writeError(c, http.StatusConflict, "only pending quotations can be approved or rejected")
			return
		}
	}
	quotation, err := h.service.UpdateStatus(c.Request.Context(), id, request.Status)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": quotation})
}

func (h *QuotationController) Delete(c *gin.Context) {
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

func (h *QuotationController) PDF(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	quotation, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	data, err := quotationPDF(quotation)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.pdf"`, quotation.Number))
	c.Data(http.StatusOK, "application/pdf", data)
}

func quotationPDF(quotation *models.Quotation) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 18)
	pdf.Cell(0, 10, "Quotation")
	pdf.Ln(12)
	pdf.SetFont("Arial", "", 11)
	pdf.Cell(0, 7, "Number: "+quotation.Number)
	pdf.Ln(7)
	pdf.Cell(0, 7, "Status: "+quotation.Status)
	pdf.Ln(7)
	pdf.Cell(0, 7, "Customer: "+quotation.Customer.Name)
	pdf.Ln(12)

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(85, 8, "Description", "1", 0, "L", false, 0, "")
	pdf.CellFormat(25, 8, "Quantity", "1", 0, "R", false, 0, "")
	pdf.CellFormat(35, 8, "Unit price", "1", 0, "R", false, 0, "")
	pdf.CellFormat(35, 8, "Total", "1", 1, "R", false, 0, "")
	pdf.SetFont("Arial", "", 10)
	for _, item := range quotation.Items {
		pdf.CellFormat(85, 8, item.Description, "1", 0, "L", false, 0, "")
		pdf.CellFormat(25, 8, fmt.Sprintf("%.2f", item.Quantity), "1", 0, "R", false, 0, "")
		pdf.CellFormat(35, 8, fmt.Sprintf("%.2f", item.UnitPrice), "1", 0, "R", false, 0, "")
		pdf.CellFormat(35, 8, fmt.Sprintf("%.2f", item.LineTotal), "1", 1, "R", false, 0, "")
	}
	pdf.Ln(4)
	pdf.CellFormat(145, 7, "Subtotal", "", 0, "R", false, 0, "")
	pdf.CellFormat(35, 7, fmt.Sprintf("%.2f", quotation.Subtotal), "", 1, "R", false, 0, "")
	pdf.CellFormat(145, 7, fmt.Sprintf("Tax (%.2f%%)", quotation.TaxRate), "", 0, "R", false, 0, "")
	pdf.CellFormat(35, 7, fmt.Sprintf("%.2f", quotation.TaxAmount), "", 1, "R", false, 0, "")
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(145, 8, "Grand total", "", 0, "R", false, 0, "")
	pdf.CellFormat(35, 8, fmt.Sprintf("%.2f", quotation.GrandTotal), "", 1, "R", false, 0, "")
	if quotation.Notes != "" {
		pdf.Ln(6)
		pdf.SetFont("Arial", "B", 10)
		pdf.Cell(0, 7, "Notes")
		pdf.Ln(7)
		pdf.SetFont("Arial", "", 10)
		pdf.MultiCell(0, 6, quotation.Notes, "", "L", false)
	}

	var buffer bytes.Buffer
	if err := pdf.Output(&buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
