package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"quotation/models"
	"quotation/repositories"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrUnauthorized = errors.New("unauthorized")
var ErrForbidden = errors.New("forbidden")
var ErrLastAdmin = errors.New("cannot delete the last administrator")

type UserInput struct {
	Name     string
	Email    string
	Password string
	Role     string
}

type AuthService interface {
	SetupRequired(context.Context) (bool, error)
	Setup(context.Context, UserInput) (string, *models.User, error)
	Login(context.Context, string, string) (string, *models.User, error)
	Authenticate(context.Context, string) (*models.User, error)
	Logout(context.Context, string) error
	ListUsers(context.Context) ([]models.User, error)
	CreateUser(context.Context, UserInput) (*models.User, error)
	DeleteUser(context.Context, uint) error
}

type authService struct{ repo repositories.AuthRepository }

const sessionDuration = 24 * time.Hour

func NewAuthService(repo repositories.AuthRepository) AuthService {
	return &authService{repo: repo}
}

func (s *authService) SetupRequired(ctx context.Context) (bool, error) {
	return s.repo.SetupRequired(ctx)
}

func (s *authService) Setup(ctx context.Context, input UserInput) (string, *models.User, error) {
	user, err := newUser(input, models.RoleAdmin)
	if err != nil {
		return "", nil, err
	}
	if err := s.repo.CreateFirstUser(ctx, user); errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil, ErrConflict
	} else if err != nil {
		return "", nil, mapUniqueError(err)
	}
	token, err := s.createSession(ctx, user.ID)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

func (s *authService) Login(ctx context.Context, email, password string) (string, *models.User, error) {
	user, err := s.repo.UserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil, ErrUnauthorized
	}
	if err != nil {
		return "", nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return "", nil, ErrUnauthorized
	}
	token, err := s.createSession(ctx, user.ID)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

func (s *authService) Authenticate(ctx context.Context, token string) (*models.User, error) {
	if len(token) != 64 {
		return nil, ErrUnauthorized
	}
	user, err := s.repo.UserBySession(ctx, tokenHash(token), time.Now())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrUnauthorized
	}
	return user, err
}

func (s *authService) Logout(ctx context.Context, token string) error {
	if len(token) != 64 {
		return ErrUnauthorized
	}
	return s.repo.DeleteSession(ctx, tokenHash(token))
}

func (s *authService) ListUsers(ctx context.Context) ([]models.User, error) {
	return s.repo.ListUsers(ctx)
}

func (s *authService) CreateUser(ctx context.Context, input UserInput) (*models.User, error) {
	user, err := newUser(input, input.Role)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, mapUniqueError(err)
	}
	return user, nil
}

func (s *authService) DeleteUser(ctx context.Context, id uint) error {
	err := s.repo.DeleteUser(ctx, id)
	if err == nil {
		return nil
	}
	if errors.Is(err, repositories.ErrLastAdmin) {
		return ErrLastAdmin
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

func (s *authService) createSession(ctx context.Context, userID uint) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(bytes)
	session := &models.AuthSession{
		UserID: userID, TokenHash: tokenHash(token), ExpiresAt: time.Now().Add(sessionDuration),
	}
	if err := s.repo.CreateSession(ctx, session); err != nil {
		return "", err
	}
	return token, nil
}

func newUser(input UserInput, role string) (*models.User, error) {
	name := strings.TrimSpace(input.Name)
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if name == "" || utf8.RuneCountInString(name) > 150 || len(input.Password) < 10 ||
		len(input.Password) > 72 || !validEmail(email) {
		return nil, ErrInvalidInput
	}
	if role != models.RoleAdmin && role != models.RoleUser &&
		role != models.RoleSalesExecutive && role != models.RoleManager {
		return nil, ErrInvalidInput
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	return &models.User{Name: name, Email: email, PasswordHash: string(hash), Role: role}, nil
}

func validEmail(email string) bool {
	if len(email) > 255 {
		return false
	}
	address, err := mail.ParseAddress(email)
	return err == nil && address.Address == email
}

func tokenHash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func mapUniqueError(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "duplicate") ||
		strings.Contains(strings.ToLower(err.Error()), "unique constraint") {
		return ErrConflict
	}
	return err
}
