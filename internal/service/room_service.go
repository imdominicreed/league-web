package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/repository"
	"github.com/google/uuid"
)

var (
	ErrRoomNotFound = errors.New("room not found")
	ErrSideTaken    = errors.New("side is already taken")
)

type RoomService struct {
	roomRepo       repository.RoomRepository
	draftStateRepo repository.DraftStateRepository
}

func NewRoomService(roomRepo repository.RoomRepository, draftStateRepo repository.DraftStateRepository) *RoomService {
	return &RoomService{
		roomRepo:       roomRepo,
		draftStateRepo: draftStateRepo,
	}
}

type CreateRoomInput struct {
	CreatedBy     uuid.UUID
	DraftMode     domain.DraftMode
	TimerDuration int
	SeriesID      *uuid.UUID
}

func (s *RoomService) CreateRoom(ctx context.Context, input CreateRoomInput) (*domain.Room, error) {
	shortCode := generateShortCode()

	room := &domain.Room{
		ID:                   uuid.New(),
		ShortCode:            shortCode,
		CreatedBy:            input.CreatedBy,
		DraftMode:            input.DraftMode,
		TimerDurationSeconds: input.TimerDuration,
		Status:               domain.RoomStatusWaiting,
		SeriesID:             input.SeriesID,
		GameNumber:           1,
	}

	if err := s.roomRepo.Create(ctx, room); err != nil {
		return nil, err
	}

	// Create initial draft state
	draftState := &domain.DraftState{
		ID:           uuid.New(),
		RoomID:       room.ID,
		CurrentPhase: 0,
		BlueBans:     []byte("[]"),
		RedBans:      []byte("[]"),
		BluePicks:    []byte("[]"),
		RedPicks:     []byte("[]"),
		IsComplete:   false,
	}

	if err := s.draftStateRepo.Create(ctx, draftState); err != nil {
		return nil, err
	}

	return room, nil
}

func (s *RoomService) GetRoom(ctx context.Context, idOrCode string) (*domain.Room, error) {
	// Try UUID first
	if id, err := uuid.Parse(idOrCode); err == nil {
		return s.roomRepo.GetByID(ctx, id)
	}

	// Try short code
	return s.roomRepo.GetByShortCode(ctx, strings.ToUpper(idOrCode))
}

func (s *RoomService) JoinRoom(ctx context.Context, roomID uuid.UUID, userID uuid.UUID, side domain.Side) (*domain.Room, error) {
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return nil, ErrRoomNotFound
	}

	switch side {
	case domain.SideBlue:
		if room.BlueSideUserID != nil && *room.BlueSideUserID != userID {
			return nil, ErrSideTaken
		}
		room.BlueSideUserID = &userID
	case domain.SideRed:
		if room.RedSideUserID != nil && *room.RedSideUserID != userID {
			return nil, ErrSideTaken
		}
		room.RedSideUserID = &userID
	}

	if err := s.roomRepo.Update(ctx, room); err != nil {
		return nil, err
	}

	return room, nil
}

func (s *RoomService) GetUserRooms(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Room, error) {
	return s.roomRepo.GetByUserID(ctx, userID, limit, offset)
}

func generateShortCode() string {
	bytes := make([]byte, 3)
	rand.Read(bytes)
	return strings.ToUpper(hex.EncodeToString(bytes))
}
