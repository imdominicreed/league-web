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
	merakiLaneDataURL = "https://cdn.merakianalytics.com/riot/lol/resources/latest/en-US/championrates.json"
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

// Meraki Analytics champion rates data
type MerakiChampionRates struct {
	Data  map[string]map[string]MerakiPlayRate `json:"data"`
	Patch string                               `json:"patch"`
}

type MerakiPlayRate struct {
	PlayRate float64 `json:"playRate"`
}

func (s *ChampionService) SyncFromDataDragon(ctx context.Context) (int, string, error) {
	// Get latest version
	version, err := s.getLatestVersion()
	if err != nil {
		return 0, "", fmt.Errorf("failed to get latest version: %w", err)
	}

	// Get champions from Data Dragon
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

	// Get lane data from Meraki Analytics
	laneData := s.fetchLaneData()

	champions := make([]*domain.Champion, 0, len(championsResp.Data))
	for _, c := range championsResp.Data {
		tagsJSON, _ := json.Marshal(c.Tags)

		// Get lanes for this champion (keyed by numeric key)
		lanes := laneData[c.Key]
		if len(lanes) == 0 {
			lanes = []string{"mid"} // default
		}
		lanesJSON, _ := json.Marshal(lanes)

		champion := &domain.Champion{
			ID:           c.ID,
			Key:          c.Key,
			Name:         c.Name,
			Title:        c.Title,
			ImageURL:     fmt.Sprintf("%s/cdn/%s/img/champion/%s", dataDragonBaseURL, version, c.Image.Full),
			Tags:         tagsJSON,
			Lanes:        lanesJSON,
			LastSyncedAt: time.Now(),
		}
		champions = append(champions, champion)
	}

	if err := s.championRepo.UpsertMany(ctx, champions); err != nil {
		return 0, "", fmt.Errorf("failed to upsert champions: %w", err)
	}

	return len(champions), version, nil
}

// fetchLaneData fetches champion lane data from Meraki Analytics
// Returns a map of champion key -> ordered list of lanes (by playrate)
func (s *ChampionService) fetchLaneData() map[string][]string {
	result := make(map[string][]string)

	resp, err := s.httpClient.Get(merakiLaneDataURL)
	if err != nil {
		fmt.Printf("Warning: failed to fetch lane data: %v\n", err)
		return result
	}
	defer resp.Body.Close()

	var merakiData MerakiChampionRates
	if err := json.NewDecoder(resp.Body).Decode(&merakiData); err != nil {
		fmt.Printf("Warning: failed to decode lane data: %v\n", err)
		return result
	}

	// Position mapping from Meraki to our format
	positionToLane := map[string]string{
		"TOP":     "top",
		"JUNGLE":  "jungle",
		"MIDDLE":  "mid",
		"BOTTOM":  "bot",
		"UTILITY": "support",
	}

	// Minimum playrate threshold to include a lane (1%)
	const minPlayRate = 1.0

	for champKey, positions := range merakiData.Data {
		// Collect lanes with significant playrate
		type laneRate struct {
			lane string
			rate float64
		}
		var lanes []laneRate

		for pos, data := range positions {
			if lane, ok := positionToLane[pos]; ok && data.PlayRate >= minPlayRate {
				lanes = append(lanes, laneRate{lane, data.PlayRate})
			}
		}

		// Sort by playrate descending
		for i := 0; i < len(lanes); i++ {
			for j := i + 1; j < len(lanes); j++ {
				if lanes[j].rate > lanes[i].rate {
					lanes[i], lanes[j] = lanes[j], lanes[i]
				}
			}
		}

		// Extract lane names
		laneNames := make([]string, len(lanes))
		for i, l := range lanes {
			laneNames[i] = l.lane
		}

		if len(laneNames) > 0 {
			result[champKey] = laneNames
		}
	}

	return result
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
