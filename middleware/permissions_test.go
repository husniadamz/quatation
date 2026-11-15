package middleware

import (
	"testing"

	"quotation/models"
)

func TestRolePermissions(t *testing.T) {
	tests := []struct {
		role, resource, action string
		want                   bool
	}{
		{models.RoleAdmin, ResourceUsers, ActionDelete, true},
		{models.RoleSalesExecutive, ResourceCustomers, ActionDelete, true},
		{models.RoleSalesExecutive, ResourceQuotations, ActionWrite, true},
		{models.RoleSalesExecutive, ResourceQuotations, ActionApprove, false},
		{models.RoleSalesExecutive, ResourceQuotations, ActionDelete, false},
		{models.RoleManager, ResourceQuotations, ActionRead, true},
		{models.RoleManager, ResourceQuotations, ActionApprove, true},
		{models.RoleManager, ResourceCustomers, ActionRead, false},
		{models.RoleManager, ResourceQuotations, ActionWrite, false},
		{models.RoleManager, ResourceUsers, ActionRead, false},
		{models.RoleUser, ResourceCustomers, ActionWrite, true},
	}
	for _, test := range tests {
		t.Run(test.role+"/"+test.resource+"/"+test.action, func(t *testing.T) {
			if got := CanAccess(test.role, test.resource, test.action); got != test.want {
				t.Fatalf("CanAccess(%q, %q, %q) = %t, want %t",
					test.role, test.resource, test.action, got, test.want)
			}
		})
	}
}
