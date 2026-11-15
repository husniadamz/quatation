package controllers

import (
	"net/http"
	"strconv"

	"quotation/models"
	"quotation/services"

	"github.com/gin-gonic/gin"
)

type AuthController struct{ service services.AuthService }

func NewAuthController(service services.AuthService) *AuthController {
	return &AuthController{service: service}
}

type authRequest struct {
	Name     string `json:"name" binding:"required,max=150"`
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,min=10,max=72"`
	Role     string `json:"role"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Password string `json:"password" binding:"required,max=72"`
}

func (r authRequest) input() services.UserInput {
	return services.UserInput{Name: r.Name, Email: r.Email, Password: r.Password, Role: r.Role}
}

func (h *AuthController) SetupStatus(c *gin.Context) {
	required, err := h.service.SetupRequired(c.Request.Context())
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"setup_required": required}})
}

func (h *AuthController) Setup(c *gin.Context) {
	var request authRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "provide a valid name, email, and password of at least 10 characters")
		return
	}
	token, user, err := h.service.Setup(c.Request.Context(), request.input())
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{"token": token, "user": user}})
}

func (h *AuthController) Login(c *gin.Context) {
	var request loginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "invalid login data")
		return
	}
	token, user, err := h.service.Login(c.Request.Context(), request.Email, request.Password)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"token": token, "user": user}})
}

func (h *AuthController) Me(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "authentication required")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": user})
}

func (h *AuthController) Logout(c *gin.Context) {
	token, _ := c.Get("auth_token")
	if err := h.service.Logout(c.Request.Context(), token.(string)); err != nil {
		handleServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *AuthController) ListUsers(c *gin.Context) {
	users, err := h.service.ListUsers(c.Request.Context())
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

func (h *AuthController) CreateUser(c *gin.Context) {
	var request authRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "provide a valid name, email, role, and password of at least 10 characters")
		return
	}
	user, err := h.service.CreateUser(c.Request.Context(), request.input())
	if err != nil {
		handleServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": user})
}

func (h *AuthController) DeleteUser(c *gin.Context) {
	value, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || value == 0 {
		writeError(c, http.StatusBadRequest, "invalid user ID")
		return
	}
	user, ok := currentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "authentication required")
		return
	}
	if user.ID == uint(value) {
		writeError(c, http.StatusBadRequest, "you cannot delete your own account")
		return
	}
	if err := h.service.DeleteUser(c.Request.Context(), uint(value)); err != nil {
		handleServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func currentUser(c *gin.Context) (*models.User, bool) {
	value, ok := c.Get("user")
	if !ok {
		return nil, false
	}
	user, ok := value.(*models.User)
	return user, ok
}
