package services

import (
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"quotation/models"
)

func TestNewUserHashesPasswordAndNormalizesEmail(t *testing.T) {
	user, err := newUser(UserInput{
		Name: "  Alex Example ", Email: "ALEX@example.com", Password: "strong-password-123",
	}, models.RoleAdmin)
	if err != nil {
		t.Fatal(err)
	}
	if user.Name != "Alex Example" || user.Email != "alex@example.com" || user.Role != models.RoleAdmin {
		t.Fatalf("unexpected normalized user: %#v", user)
	}
	if user.PasswordHash == "strong-password-123" {
		t.Fatal("password was stored without hashing")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("strong-password-123")); err != nil {
		t.Fatal("stored password hash does not verify")
	}
}

func TestNewUserRejectsWeakPasswordsAndInvalidRoles(t *testing.T) {
	tests := []struct {
		name  string
		input UserInput
		role  string
	}{
		{name: "short password", input: UserInput{Name: "Alex", Email: "alex@example.com", Password: "short"}, role: models.RoleUser},
		{name: "invalid email", input: UserInput{Name: "Alex", Email: "not-an-email", Password: "strong-password-123"}, role: models.RoleUser},
		{name: "invalid role", input: UserInput{Name: "Alex", Email: "alex@example.com", Password: "strong-password-123"}, role: "owner"},
		{name: "blank name", input: UserInput{Name: "   ", Email: "alex@example.com", Password: "strong-password-123"}, role: models.RoleUser},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := newUser(test.input, test.role); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("newUser error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestTokenHashDoesNotExposeRawToken(t *testing.T) {
	token := "a-private-session-token"
	hash := tokenHash(token)
	if hash == token || len(hash) != 64 {
		t.Fatalf("unexpected token hash %q", hash)
	}
	if tokenHash(token) != hash {
		t.Fatal("token hash is not deterministic")
	}
}
