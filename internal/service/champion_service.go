package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dom/league-draft-website/internal/config"
	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/repository"
)

const (
	dataDragonBaseURL = "https://ddragon.leagueoflegends.com"
)

type ChampionService struct {
	championRepo repository.ChampionRepository
	cfg          *config.Config
	httpClient   *http.Client
}

func NewChampionService(championRepo repository.ChampionRepository, cfg *config.Config) *ChampionService {
	return &ChampionService{
		championRepo: championRepo,
		cfg:          cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *ChampionService) GetAllChampions(ctx context.Context) ([]*domain.Champion, error) {
	return s.championRepo.GetAll(ctx)
}

func (s *ChampionService) GetChampion(ctx context.Context, id string) (*domain.Champion, error) {
	return s.championRepo.GetByID(ctx, id)
}

type DataDragonVersionResponse []string

type DataDragonChampionsResponse struct {
	Type    string                       `json:"type"`
	Format  string                       `json:"format"`
	Version string                       `json:"version"`
	Data    map[string]DataDragonChampion `json:"data"`
}

type DataDragonChampion struct {
	ID    string   `json:"id"`
	Key   string   `json:"key"`
	Name  string   `json:"name"`
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
	Image struct {
		Full string `json:"full"`
	} `json:"image"`
}

func (s *ChampionService) SyncFromDataDragon(ctx context.Context) (int, string, error) {
	// Get latest version
	version, err := s.getLatestVersion()
	if err != nil {
		return 0, "", fmt.Errorf("failed to get latest version: %w", err)
	}

	// Get champions
	championsURL := fmt.Sprintf("%s/cdn/%s/data/en_US/champion.json", dataDragonBaseURL, version)
	resp, err := s.httpClient.Get(championsURL)
	if err != nil {
		return 0, "", fmt.Errorf("failed to fetch champions: %w", err)
	}
	defer resp.Body.Close()

	var championsResp DataDragonChampionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&championsResp); err != nil {
		return 0, "", fmt.Errorf("failed to decode champions: %w", err)
	}

	champions := make([]*domain.Champion, 0, len(championsResp.Data))
	for _, c := range championsResp.Data {
		tagsJSON, _ := json.Marshal(c.Tags)
		champion := &domain.Champion{
			ID:           c.ID,
			Key:          c.Key,
			Name:         c.Name,
			Title:        c.Title,
			ImageURL:     fmt.Sprintf("%s/cdn/%s/img/champion/%s", dataDragonBaseURL, version, c.Image.Full),
			Tags:         tagsJSON,
			LastSyncedAt: time.Now(),
		}
		champions = append(champions, champion)
	}

	if err := s.championRepo.UpsertMany(ctx, champions); err != nil {
		return 0, "", fmt.Errorf("failed to upsert champions: %w", err)
	}

	return len(champions), version, nil
}

func (s *ChampionService) getLatestVersion() (string, error) {
	if s.cfg.DataDragonVersion != "" {
		return s.cfg.DataDragonVersion, nil
	}

	resp, err := s.httpClient.Get(dataDragonBaseURL + "/api/versions.json")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var versions DataDragonVersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no versions available")
	}

	return versions[0], nil
}

func (s *ChampionService) GetLatestVersion() (string, error) {
	return s.getLatestVersion()
}
