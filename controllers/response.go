package controllers

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"quotation/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func parseID(c *gin.Context) (uint, bool) {
	value, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || value == 0 {
		writeError(c, http.StatusBadRequest, "invalid ID")
		return 0, false
	}
	return uint(value), true
}

func pagination(c *gin.Context) (int, int, bool) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		writeError(c, http.StatusBadRequest, "page must be a positive integer")
		return 0, 0, false
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		writeError(c, http.StatusBadRequest, "page_size must be between 1 and 100")
		return 0, 0, false
	}
	return page, pageSize, true
}

func handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrNotFound), errors.Is(err, gorm.ErrRecordNotFound):
		writeError(c, http.StatusNotFound, "resource not found")
	case errors.Is(err, services.ErrInvalidInput):
		writeError(c, http.StatusBadRequest, "invalid input")
	case errors.Is(err, services.ErrConflict):
		writeError(c, http.StatusConflict, "resource already exists or setup is complete")
	case errors.Is(err, services.ErrUnauthorized):
		writeError(c, http.StatusUnauthorized, "invalid credentials or expired session")
	case errors.Is(err, services.ErrForbidden), errors.Is(err, services.ErrLastAdmin):
		writeError(c, http.StatusForbidden, "operation not allowed")
	default:
		log.Printf("request failed: %v", err)
		writeError(c, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}
