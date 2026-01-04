package service

import (
	"context"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/repository"
	"github.com/google/uuid"
)

type DraftService struct {
	draftStateRepo  repository.DraftStateRepository
	draftActionRepo repository.DraftActionRepository
	fearlessBanRepo repository.FearlessBanRepository
}

func NewDraftService(
	draftStateRepo repository.DraftStateRepository,
	draftActionRepo repository.DraftActionRepository,
	fearlessBanRepo repository.FearlessBanRepository,
) *DraftService {
	return &DraftService{
		draftStateRepo:  draftStateRepo,
		draftActionRepo: draftActionRepo,
		fearlessBanRepo: fearlessBanRepo,
	}
}

func (s *DraftService) GetDraftState(ctx context.Context, roomID uuid.UUID) (*domain.DraftState, error) {
	return s.draftStateRepo.GetByRoomID(ctx, roomID)
}

func (s *DraftService) UpdateDraftState(ctx context.Context, state *domain.DraftState) error {
	return s.draftStateRepo.Update(ctx, state)
}

func (s *DraftService) RecordAction(ctx context.Context, action *domain.DraftAction) error {
	return s.draftActionRepo.Create(ctx, action)
}

func (s *DraftService) GetDraftHistory(ctx context.Context, roomID uuid.UUID) ([]*domain.DraftAction, error) {
	return s.draftActionRepo.GetByRoomID(ctx, roomID)
}

func (s *DraftService) GetFearlessBans(ctx context.Context, seriesID uuid.UUID) ([]string, error) {
	bans, err := s.fearlessBanRepo.GetBySeriesID(ctx, seriesID)
	if err != nil {
		return nil, err
	}

	championIDs := make([]string, len(bans))
	for i, ban := range bans {
		championIDs[i] = ban.ChampionID
	}
	return championIDs, nil
}

func (s *DraftService) AddFearlessBan(ctx context.Context, seriesID uuid.UUID, championID string, gameNumber int, team domain.Side) error {
	ban := &domain.FearlessBan{
		ID:           uuid.New(),
		SeriesID:     seriesID,
		ChampionID:   championID,
		BannedInGame: gameNumber,
		PickedByTeam: team,
	}
	return s.fearlessBanRepo.Create(ctx, ban)
}
