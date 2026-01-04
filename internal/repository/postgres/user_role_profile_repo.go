package postgres

import (
	"context"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type userRoleProfileRepository struct {
	db *gorm.DB
}

func NewUserRoleProfileRepository(db *gorm.DB) *userRoleProfileRepository {
	return &userRoleProfileRepository{db: db}
}

func (r *userRoleProfileRepository) Create(ctx context.Context, profile *domain.UserRoleProfile) error {
	return r.db.WithContext(ctx).Create(profile).Error
}

func (r *userRoleProfileRepository) CreateMany(ctx context.Context, profiles []*domain.UserRoleProfile) error {
	if len(profiles) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(profiles).Error
}

func (r *userRoleProfileRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.UserRoleProfile, error) {
	var profiles []*domain.UserRoleProfile
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("role").
		Find(&profiles).Error
	if err != nil {
		return nil, err
	}
	return profiles, nil
}

func (r *userRoleProfileRepository) GetByUserIDAndRole(ctx context.Context, userID uuid.UUID, role domain.Role) (*domain.UserRoleProfile, error) {
	var profile domain.UserRoleProfile
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND role = ?", userID, role).
		First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func (r *userRoleProfileRepository) Upsert(ctx context.Context, profile *domain.UserRoleProfile) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "role"}},
			DoUpdates: clause.AssignmentColumns([]string{"league_rank", "mmr", "comfort_rating", "updated_at"}),
		}).
		Create(profile).Error
}

func (r *userRoleProfileRepository) Update(ctx context.Context, profile *domain.UserRoleProfile) error {
	return r.db.WithContext(ctx).Save(profile).Error
}

func (r *userRoleProfileRepository) GetByUserIDs(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]*domain.UserRoleProfile, error) {
	var profiles []*domain.UserRoleProfile
	err := r.db.WithContext(ctx).
		Where("user_id IN ?", userIDs).
		Order("user_id, role").
		Find(&profiles).Error
	if err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID][]*domain.UserRoleProfile)
	for _, p := range profiles {
		result[p.UserID] = append(result[p.UserID], p)
	}
	return result, nil
}
