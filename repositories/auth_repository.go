package repositories

import (
	"context"
	"errors"
	"time"

	"quotation/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrLastAdmin = errors.New("cannot delete the last administrator")

type AuthRepository interface {
	SetupRequired(context.Context) (bool, error)
	CreateFirstUser(context.Context, *models.User) error
	CreateUser(context.Context, *models.User) error
	UserByEmail(context.Context, string) (*models.User, error)
	UserByID(context.Context, uint) (*models.User, error)
	ListUsers(context.Context) ([]models.User, error)
	CreateSession(context.Context, *models.AuthSession) error
	UserBySession(context.Context, string, time.Time) (*models.User, error)
	DeleteSession(context.Context, string) error
	DeleteUser(context.Context, uint) error
}

type authRepository struct{ db *gorm.DB }

func NewAuthRepository(db *gorm.DB) AuthRepository {
	return &authRepository{db: db}
}

func (r *authRepository) SetupRequired(ctx context.Context) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.User{}).Count(&count).Error; err != nil {
		return false, err
	}
	return count == 0, nil
}

func (r *authRepository) CreateFirstUser(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		state := models.InstallState{ID: 1}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).FirstOrCreate(&state).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&models.User{}).Count(&count).Error; err != nil {
			return err
		}
		if count != 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Create(user).Error
	})
}

func (r *authRepository) CreateUser(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *authRepository) UserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *authRepository) UserByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *authRepository) ListUsers(ctx context.Context) ([]models.User, error) {
	var users []models.User
	err := r.db.WithContext(ctx).Order("id ASC").Find(&users).Error
	return users, err
}

func (r *authRepository) CreateSession(ctx context.Context, session *models.AuthSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *authRepository) UserBySession(ctx context.Context, tokenHash string, now time.Time) (*models.User, error) {
	var session models.AuthSession
	err := r.db.WithContext(ctx).Where("token_hash = ? AND expires_at > ?", tokenHash, now).First(&session).Error
	if err != nil {
		return nil, err
	}
	return r.UserByID(ctx, session.UserID)
}

func (r *authRepository) DeleteSession(ctx context.Context, tokenHash string) error {
	return r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).Delete(&models.AuthSession{}).Error
}

func (r *authRepository) DeleteUser(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.First(&user, id).Error; err != nil {
			return err
		}
		if user.Role == models.RoleAdmin {
			var admins int64
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Model(&models.User{}).
				Where("role = ?", models.RoleAdmin).Count(&admins).Error; err != nil {
				return err
			}
			if admins <= 1 {
				return ErrLastAdmin
			}
		}
		if err := tx.Where("user_id = ?", id).Delete(&models.AuthSession{}).Error; err != nil {
			return err
		}
		return tx.Delete(&user).Error
	})
}
