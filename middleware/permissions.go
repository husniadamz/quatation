package middleware

import (
	"net/http"

	"quotation/models"

	"github.com/gin-gonic/gin"
)

const (
	ResourceDashboard  = "dashboard"
	ResourceCustomers  = "customers"
	ResourceQuotations = "quotations"
	ResourceTemplates  = "templates"
	ResourceUsers      = "users"
	ResourceCompanies  = "companies"

	ActionRead    = "read"
	ActionWrite   = "write"
	ActionDelete  = "delete"
	ActionApprove = "approve"
)

func RequirePermission(resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		value, ok := c.Get("user")
		user, valid := value.(*models.User)
		if !ok || !valid || !CanAccess(user.Role, resource, action) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "your role cannot perform this action"})
			return
		}
		c.Next()
	}
}

func CanAccess(role, resource, action string) bool {
	if role == models.RoleAdmin {
		return true
	}
	if role == models.RoleUser {
		role = models.RoleSalesExecutive
	}

	switch {
	case role == models.RoleSalesExecutive && resource == ResourceDashboard && action == ActionRead:
		return true
	case role == models.RoleSalesExecutive && resource == ResourceCustomers &&
		(action == ActionRead || action == ActionWrite || action == ActionDelete):
		return true
	case role == models.RoleSalesExecutive && resource == ResourceQuotations &&
		(action == ActionRead || action == ActionWrite):
		return true
	case role == models.RoleSalesExecutive && resource == ResourceTemplates && action == ActionRead:
		return true
	case role == models.RoleManager && resource == ResourceDashboard && action == ActionRead:
		return true
	case role == models.RoleManager && resource == ResourceQuotations &&
		(action == ActionRead || action == ActionApprove):
		return true
	default:
		return false
	}
}
