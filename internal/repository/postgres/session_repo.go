package postgres

import (
	"context"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type sessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) *sessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(ctx context.Context, session *domain.UserSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *sessionRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.UserSession, error) {
	var session domain.UserSession
	err := r.db.WithContext(ctx).First(&session, "user_id = ?", userID).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *sessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.UserSession{}, "id = ?", id).Error
}

func (r *sessionRepository) DeleteByUserID(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.UserSession{}, "user_id = ?", userID).Error
}
