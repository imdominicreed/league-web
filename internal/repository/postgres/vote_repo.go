package postgres

import (
	"context"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type voteRepository struct {
	db *gorm.DB
}

func NewVoteRepository(db *gorm.DB) *voteRepository {
	return &voteRepository{db: db}
}

func (r *voteRepository) Create(ctx context.Context, vote *domain.Vote) error {
	return r.db.WithContext(ctx).Create(vote).Error
}

func (r *voteRepository) Update(ctx context.Context, vote *domain.Vote) error {
	return r.db.WithContext(ctx).Save(vote).Error
}

func (r *voteRepository) GetByLobbyAndUser(ctx context.Context, lobbyID, userID uuid.UUID) (*domain.Vote, error) {
	var vote domain.Vote
	err := r.db.WithContext(ctx).
		Where("lobby_id = ? AND user_id = ?", lobbyID, userID).
		First(&vote).Error
	if err != nil {
		return nil, err
	}
	return &vote, nil
}

func (r *voteRepository) GetVotesByLobby(ctx context.Context, lobbyID uuid.UUID) ([]*domain.Vote, error) {
	var votes []*domain.Vote
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("lobby_id = ?", lobbyID).
		Find(&votes).Error
	if err != nil {
		return nil, err
	}
	return votes, nil
}

func (r *voteRepository) GetVoteCounts(ctx context.Context, lobbyID uuid.UUID) (map[int]int, error) {
	type countResult struct {
		MatchOptionNum int
		Count          int
	}

	var results []countResult
	err := r.db.WithContext(ctx).
		Model(&domain.Vote{}).
		Select("match_option_num, COUNT(*) as count").
		Where("lobby_id = ?", lobbyID).
		Group("match_option_num").
		Find(&results).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[int]int)
	for _, r := range results {
		counts[r.MatchOptionNum] = r.Count
	}
	return counts, nil
}

func (r *voteRepository) DeleteByLobby(ctx context.Context, lobbyID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("lobby_id = ?", lobbyID).
		Delete(&domain.Vote{}).Error
}

func (r *voteRepository) DeleteByLobbyAndUser(ctx context.Context, lobbyID, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("lobby_id = ? AND user_id = ?", lobbyID, userID).
		Delete(&domain.Vote{}).Error
}
