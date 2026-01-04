package postgres_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/repository/postgres"
	"github.com/dom/league-draft-website/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func TestChampionRepository_Upsert(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repo := postgres.NewChampionRepository(testDB.DB)
	ctx := context.Background()

	tagsJSON, _ := json.Marshal([]string{"Fighter", "Tank"})
	champion := &domain.Champion{
		ID:           "Aatrox",
		Key:          "Aatrox",
		Name:         "Aatrox",
		Title:        "The Darkin Blade",
		ImageURL:     "https://example.com/aatrox.png",
		Tags:         datatypes.JSON(tagsJSON),
		LastSyncedAt: time.Now(),
	}

	// Create
	err := repo.Upsert(ctx, champion)
	require.NoError(t, err)

	// Verify creation
	got, err := repo.GetByID(ctx, "Aatrox")
	require.NoError(t, err)
	assert.Equal(t, "Aatrox", got.Name)
	assert.Equal(t, "The Darkin Blade", got.Title)

	// Update
	champion.Title = "The World Ender"
	err = repo.Upsert(ctx, champion)
	require.NoError(t, err)

	// Verify update
	got, err = repo.GetByID(ctx, "Aatrox")
	require.NoError(t, err)
	assert.Equal(t, "The World Ender", got.Title)
}

func TestChampionRepository_UpsertMany(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repo := postgres.NewChampionRepository(testDB.DB)
	ctx := context.Background()

	tagsJSON1, _ := json.Marshal([]string{"Mage"})
	tagsJSON2, _ := json.Marshal([]string{"Assassin"})

	champions := []*domain.Champion{
		{
			ID:           "Ahri",
			Key:          "Ahri",
			Name:         "Ahri",
			Title:        "The Nine-Tailed Fox",
			ImageURL:     "https://example.com/ahri.png",
			Tags:         datatypes.JSON(tagsJSON1),
			LastSyncedAt: time.Now(),
		},
		{
			ID:           "Akali",
			Key:          "Akali",
			Name:         "Akali",
			Title:        "The Rogue Assassin",
			ImageURL:     "https://example.com/akali.png",
			Tags:         datatypes.JSON(tagsJSON2),
			LastSyncedAt: time.Now(),
		},
	}

	err := repo.UpsertMany(ctx, champions)
	require.NoError(t, err)

	// Verify all were created
	all, err := repo.GetAll(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestChampionRepository_GetAll(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repo := postgres.NewChampionRepository(testDB.DB)
	ctx := context.Background()

	// Empty database
	champions, err := repo.GetAll(ctx)
	require.NoError(t, err)
	assert.Empty(t, champions)

	// Add some champions
	testutil.SeedChampions(t, testDB.DB, 5)

	// Verify all are returned and sorted by name
	champions, err = repo.GetAll(ctx)
	require.NoError(t, err)
	assert.Len(t, champions, 5)

	// Verify sorted by name
	for i := 1; i < len(champions); i++ {
		assert.LessOrEqual(t, champions[i-1].Name, champions[i].Name)
	}
}

func TestChampionRepository_GetByID(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repo := postgres.NewChampionRepository(testDB.DB)
	ctx := context.Background()

	// Create a champion
	champion := testutil.NewChampionBuilder().
		WithID("Ezreal").
		WithName("Ezreal").
		WithTitle("The Prodigal Explorer").
		Build(t, testDB.DB)

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "existing champion",
			id:      champion.ID,
			wantErr: false,
		},
		{
			name:    "non-existent champion",
			id:      "NonExistent",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.GetByID(ctx, tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, champion.ID, got.ID)
			assert.Equal(t, champion.Name, got.Name)
		})
	}
}

func TestChampionRepository_TagsJSONHandling(t *testing.T) {
	testDB := testutil.NewTestDB(t)
	repo := postgres.NewChampionRepository(testDB.DB)
	ctx := context.Background()

	// Create champion with multiple tags
	champion := testutil.NewChampionBuilder().
		WithID("Garen").
		WithTags([]string{"Fighter", "Tank"}).
		Build(t, testDB.DB)

	// Verify tags are properly stored and retrieved
	got, err := repo.GetByID(ctx, champion.ID)
	require.NoError(t, err)

	var tags []string
	err = json.Unmarshal(got.Tags, &tags)
	require.NoError(t, err)
	assert.Contains(t, tags, "Fighter")
	assert.Contains(t, tags, "Tank")
}
