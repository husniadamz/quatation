package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"quotation/models"
	"quotation/services"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
)

var pdfColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

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
	TemplateID *uint                  `json:"template_id"`
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
		CustomerID: r.CustomerID, TemplateID: r.TemplateID, TaxRate: r.TaxRate, Notes: r.Notes,
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
	var snapshot models.QuotationTemplateSnapshot
	if len(quotation.TemplateSnapshot) > 0 {
		if err := json.Unmarshal(quotation.TemplateSnapshot, &snapshot); err != nil {
			handleServiceError(c, fmt.Errorf("decode quotation template snapshot: %w", err))
			return
		}
	}
	data, err := quotationPDF(quotation, snapshot)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.pdf"`, quotation.Number))
	c.Data(http.StatusOK, "application/pdf", data)
}

func quotationPDF(quotation *models.Quotation, snapshot models.QuotationTemplateSnapshot) ([]byte, error) {
	settings := snapshot.Settings
	orientation := "P"
	if settings.Layout == "landscape" {
		orientation = "L"
	}
	font := settings.FontFamily
	if font == "" {
		font = "Arial"
	}
	pdf := gofpdf.New(orientation, "mm", "A4", "")
	pdf.SetMargins(15, 16, 15)
	pdf.SetAutoPageBreak(true, 22)
	primary := parsePDFColor(settings.PrimaryColor, [3]int{38, 74, 115})
	accent := parsePDFColor(settings.AccentColor, [3]int{214, 169, 75})
	pdf.SetFooterFunc(func() {
		if settings.FooterText == "" {
			return
		}
		pdf.SetY(-15)
		pdf.SetFont(font, "", 8)
		pdf.SetTextColor(110, 118, 130)
		pdf.CellFormat(0, 7, settings.FooterText, "", 0, "C", false, 0, "")
	})
	pdf.AddPage()
	if snapshot.LogoPath != "" {
		if _, err := os.Stat(snapshot.LogoPath); err != nil {
			return nil, fmt.Errorf("read quotation template logo: %w", err)
		} else {
			imageType := "PNG"
			if snapshot.LogoContentType == "image/jpeg" {
				imageType = "JPG"
			}
			pdf.ImageOptions(snapshot.LogoPath, 15, 15, 35, 0, false,
				gofpdf.ImageOptions{ImageType: imageType, ReadDpi: true}, 0, "")
			pdf.SetXY(55, 15)
		}
	}
	pdf.SetDrawColor(primary[0], primary[1], primary[2])
	pdf.SetTextColor(primary[0], primary[1], primary[2])
	pdf.SetFont(font, "B", 18)
	title := settings.HeaderTitle
	if title == "" {
		title = "Quotation"
	}
	pdf.Cell(0, 10, title)
	pdf.Ln(12)
	if snapshot.CompanyName != "" {
		pdf.SetFont(font, "B", 11)
		pdf.Cell(0, 7, snapshot.CompanyName)
		pdf.Ln(6)
	}
	pdf.SetFont(font, "", 9)
	for _, line := range []string{settings.CompanyAddress, settings.CompanyEmail, settings.CompanyPhone} {
		if strings.TrimSpace(line) != "" {
			pdf.Cell(0, 5, line)
			pdf.Ln(5)
		}
	}
	if settings.HeaderText != "" {
		pdf.Ln(2)
		pdf.MultiCell(0, 5, settings.HeaderText, "", "L", false)
	}
	pdf.Ln(3)
	pdf.SetDrawColor(accent[0], accent[1], accent[2])
	pdf.SetLineWidth(1)
	pageWidth, _ := pdf.GetPageSize()
	pdf.Line(15, pdf.GetY(), pageWidth-15, pdf.GetY())
	pdf.SetLineWidth(.2)
	pdf.Ln(6)
	pdf.SetTextColor(38, 45, 57)
	pdf.SetFont(font, "", 10)
	pdf.Cell(0, 7, "Quotation: "+quotation.Number)
	pdf.Ln(7)
	pdf.Cell(0, 7, "Date: "+quotation.CreatedAt.Format("02 Jan 2006"))
	pdf.Ln(7)
	if quotation.ValidUntil != nil {
		pdf.Cell(0, 7, "Valid until: "+quotation.ValidUntil.Format("02 Jan 2006"))
		pdf.Ln(7)
	}
	pdf.Cell(0, 7, "Customer: "+quotation.Customer.Name)
	pdf.Ln(11)

	widths := [4]float64{80, 25, 35, 40}
	if orientation == "L" {
		widths = [4]float64{120, 35, 50, 55}
	}
	pdf.SetFillColor(primary[0], primary[1], primary[2])
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont(font, "B", 9)
	for i, heading := range []string{"Description", "Quantity", "Unit price", "Total"} {
		align := "R"
		if i == 0 {
			align = "L"
		}
		pdf.CellFormat(widths[i], 9, heading, "", 0, align, true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetTextColor(38, 45, 57)
	pdf.SetFont(font, "", 9)
	for _, item := range quotation.Items {
		if pdf.GetY() > 265 {
			pdf.AddPage()
		}
		pdf.SetDrawColor(225, 230, 238)
		pdf.CellFormat(widths[0], 8, item.Description, "B", 0, "L", false, 0, "")
		pdf.CellFormat(widths[1], 8, fmt.Sprintf("%.2f", item.Quantity), "B", 0, "R", false, 0, "")
		pdf.CellFormat(widths[2], 8, fmt.Sprintf("%.2f", item.UnitPrice), "B", 0, "R", false, 0, "")
		pdf.CellFormat(widths[3], 8, fmt.Sprintf("%.2f", item.LineTotal), "B", 1, "R", false, 0, "")
	}
	pdf.Ln(4)
	labelWidth := widths[0] + widths[1] + widths[2]
	pdf.SetTextColor(38, 45, 57)
	pdf.SetFont(font, "", 9)
	pdf.CellFormat(labelWidth, 7, "Subtotal", "", 0, "R", false, 0, "")
	pdf.CellFormat(widths[3], 7, fmt.Sprintf("%.2f", quotation.Subtotal), "", 1, "R", false, 0, "")
	pdf.CellFormat(labelWidth, 7, fmt.Sprintf("Tax (%.2f%%)", quotation.TaxRate), "", 0, "R", false, 0, "")
	pdf.CellFormat(widths[3], 7, fmt.Sprintf("%.2f", quotation.TaxAmount), "", 1, "R", false, 0, "")
	pdf.SetFillColor(primary[0], primary[1], primary[2])
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont(font, "B", 10)
	pdf.CellFormat(labelWidth, 9, "Grand total", "", 0, "R", true, 0, "")
	pdf.CellFormat(widths[3], 9, fmt.Sprintf("%.2f", quotation.GrandTotal), "", 1, "R", true, 0, "")
	pdf.SetTextColor(38, 45, 57)
	pdf.SetFont(font, "", 9)
	for _, section := range []struct{ title, body string }{
		{"Notes", quotation.Notes},
		{"Payment details", settings.PaymentDetails},
		{"Terms and conditions", settings.Terms},
	} {
		if section.body == "" {
			continue
		}
		pdf.Ln(6)
		pdf.SetFont(font, "B", 10)
		pdf.Cell(0, 7, section.title)
		pdf.Ln(7)
		pdf.SetFont(font, "", 9)
		pdf.MultiCell(0, 5, section.body, "", "L", false)
	}
	if settings.SignatureLabel != "" {
		pdf.Ln(12)
		pdf.Cell(85, 7, settings.SignatureLabel)
		pdf.Ln(15)
		pdf.SetDrawColor(120, 130, 145)
		pdf.Line(15, pdf.GetY(), 85, pdf.GetY())
	}

	var buffer bytes.Buffer
	if err := pdf.Output(&buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func parsePDFColor(value string, fallback [3]int) [3]int {
	if !pdfColorPattern.MatchString(value) {
		return fallback
	}
	red, _ := strconv.ParseInt(value[1:3], 16, 32)
	green, _ := strconv.ParseInt(value[3:5], 16, 32)
	blue, _ := strconv.ParseInt(value[5:7], 16, 32)
	return [3]int{int(red), int(green), int(blue)}
}
