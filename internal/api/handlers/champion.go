package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/dom/league-draft-website/internal/service"
	"github.com/go-chi/chi/v5"
)

type ChampionHandler struct {
	championService *service.ChampionService
}

func NewChampionHandler(championService *service.ChampionService) *ChampionHandler {
	return &ChampionHandler{championService: championService}
}

type ChampionResponse struct {
	ID       string   `json:"id"`
	Key      string   `json:"key"`
	Name     string   `json:"name"`
	Title    string   `json:"title"`
	ImageURL string   `json:"imageUrl"`
	Tags     []string `json:"tags"`
}

type ChampionsResponse struct {
	Champions []ChampionResponse `json:"champions"`
	Version   string             `json:"version"`
}

type SyncResponse struct {
	Synced  int    `json:"synced"`
	Version string `json:"version"`
}

func (h *ChampionHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	champions, err := h.championService.GetAllChampions(r.Context())
	if err != nil {
		log.Printf("ERROR [champion.GetAll]: %v", err)
		http.Error(w, "Failed to get champions", http.StatusInternalServerError)
		return
	}

	version, _ := h.championService.GetLatestVersion()

	resp := ChampionsResponse{
		Champions: make([]ChampionResponse, len(champions)),
		Version:   version,
	}

	for i, c := range champions {
		var tags []string
		json.Unmarshal(c.Tags, &tags)

		resp.Champions[i] = ChampionResponse{
			ID:       c.ID,
			Key:      c.Key,
			Name:     c.Name,
			Title:    c.Title,
			ImageURL: c.ImageURL,
			Tags:     tags,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *ChampionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	champion, err := h.championService.GetChampion(r.Context(), id)
	if err != nil {
		log.Printf("ERROR [champion.Get] championID=%s: %v", id, err)
		http.Error(w, "Champion not found", http.StatusNotFound)
		return
	}

	var tags []string
	json.Unmarshal(champion.Tags, &tags)

	resp := ChampionResponse{
		ID:       champion.ID,
		Key:      champion.Key,
		Name:     champion.Name,
		Title:    champion.Title,
		ImageURL: champion.ImageURL,
		Tags:     tags,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *ChampionHandler) Sync(w http.ResponseWriter, r *http.Request) {
	count, version, err := h.championService.SyncFromDataDragon(r.Context())
	if err != nil {
		log.Printf("ERROR [champion.Sync]: %v", err)
		http.Error(w, "Failed to sync champions", http.StatusInternalServerError)
		return
	}

	resp := SyncResponse{
		Synced:  count,
		Version: version,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
