package service_test

import (
	"context"
	"testing"

	"github.com/dom/league-draft-website/internal/domain"
	"github.com/dom/league-draft-website/internal/service"
	"github.com/dom/league-draft-website/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVotingService(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ts := testutil.NewTestServer(t)
	ctx := context.Background()

	t.Run("CastVote_Success", func(t *testing.T) {
		ts.DB.Truncate(t)

		// Create users and lobby with voting enabled
		lobby, users := testutil.SeedLobbyWith10ReadyPlayers(t, ts.DB.DB)
		require.NotNil(t, lobby)

		// Enable voting on the lobby
		lobby.VotingEnabled = true
		lobby.VotingMode = domain.VotingModeMajority
		lobby.Status = domain.LobbyStatusMatchmaking
		err := ts.Repos.Lobby.Update(ctx, lobby)
		require.NoError(t, err)

		// Create a match option
		option := &domain.MatchOption{
			LobbyID:      lobby.ID,
			OptionNumber: 1,
		}
		err = ts.Repos.MatchOption.Create(ctx, option)
		require.NoError(t, err)

		// Cast vote
		status, err := ts.Services.Lobby.CastVote(ctx, lobby.ID, users[0].ID, 1)
		require.NoError(t, err)
		assert.NotNil(t, status)
		assert.Equal(t, 1, status.VotesCast)
		assert.Equal(t, 1, status.VoteCounts[1])
		require.NotNil(t, status.UserVotes)
		assert.Contains(t, status.UserVotes, 1)
	})

	t.Run("CastVote_ToggleVote", func(t *testing.T) {
		ts.DB.Truncate(t)

		// Create users and lobby with voting enabled
		lobby, users := testutil.SeedLobbyWith10ReadyPlayers(t, ts.DB.DB)
		require.NotNil(t, lobby)

		// Enable voting on the lobby
		lobby.VotingEnabled = true
		lobby.VotingMode = domain.VotingModeMajority
		lobby.Status = domain.LobbyStatusMatchmaking
		err := ts.Repos.Lobby.Update(ctx, lobby)
		require.NoError(t, err)

		// Create match options
		for i := 1; i <= 2; i++ {
			option := &domain.MatchOption{
				LobbyID:      lobby.ID,
				OptionNumber: i,
			}
			err := ts.Repos.MatchOption.Create(ctx, option)
			require.NoError(t, err)
		}

		// Cast first vote for option 1
		status, err := ts.Services.Lobby.CastVote(ctx, lobby.ID, users[0].ID, 1)
		require.NoError(t, err)
		assert.Equal(t, 1, status.VoteCounts[1])
		assert.Contains(t, status.UserVotes, 1)

		// Vote for option 2 as well (multi-vote)
		status, err = ts.Services.Lobby.CastVote(ctx, lobby.ID, users[0].ID, 2)
		require.NoError(t, err)
		assert.Equal(t, 1, status.VoteCounts[1]) // Option 1 still has the vote
		assert.Equal(t, 1, status.VoteCounts[2])
		assert.Contains(t, status.UserVotes, 1)
		assert.Contains(t, status.UserVotes, 2)

		// Toggle off option 1 by voting for it again
		status, err = ts.Services.Lobby.CastVote(ctx, lobby.ID, users[0].ID, 1)
		require.NoError(t, err)
		assert.Equal(t, 0, status.VoteCounts[1]) // Option 1 vote removed
		assert.Equal(t, 1, status.VoteCounts[2]) // Option 2 still has vote
		assert.NotContains(t, status.UserVotes, 1)
		assert.Contains(t, status.UserVotes, 2)
	})

	t.Run("CastVote_VotingNotEnabled", func(t *testing.T) {
		ts.DB.Truncate(t)

		// Create users and lobby without voting
		lobby, users := testutil.SeedLobbyWith10ReadyPlayers(t, ts.DB.DB)
		require.NotNil(t, lobby)

		lobby.Status = domain.LobbyStatusMatchmaking
		err := ts.Repos.Lobby.Update(ctx, lobby)
		require.NoError(t, err)

		// Try to cast vote
		_, err = ts.Services.Lobby.CastVote(ctx, lobby.ID, users[0].ID, 1)
		assert.ErrorIs(t, err, service.ErrVotingNotEnabled)
	})

	t.Run("GetVotingStatus_Majority", func(t *testing.T) {
		ts.DB.Truncate(t)

		// Create users and lobby with voting enabled
		lobby, users := testutil.SeedLobbyWith10ReadyPlayers(t, ts.DB.DB)
		require.NotNil(t, lobby)

		// Enable voting on the lobby
		lobby.VotingEnabled = true
		lobby.VotingMode = domain.VotingModeMajority
		lobby.Status = domain.LobbyStatusMatchmaking
		err := ts.Repos.Lobby.Update(ctx, lobby)
		require.NoError(t, err)

		// Create match options
		for i := 1; i <= 2; i++ {
			option := &domain.MatchOption{
				LobbyID:      lobby.ID,
				OptionNumber: i,
			}
			err := ts.Repos.MatchOption.Create(ctx, option)
			require.NoError(t, err)
		}

		// Cast votes: 6 for option 1, 4 for option 2 (majority achieved)
		for i := 0; i < 6; i++ {
			_, err := ts.Services.Lobby.CastVote(ctx, lobby.ID, users[i].ID, 1)
			require.NoError(t, err)
		}
		for i := 6; i < 10; i++ {
			_, err := ts.Services.Lobby.CastVote(ctx, lobby.ID, users[i].ID, 2)
			require.NoError(t, err)
		}

		// Check voting status
		status, err := ts.Services.Lobby.GetVotingStatus(ctx, lobby.ID, nil)
		require.NoError(t, err)
		assert.Equal(t, 10, status.VotesCast)
		assert.Equal(t, 6, status.VoteCounts[1])
		assert.Equal(t, 4, status.VoteCounts[2])
		require.NotNil(t, status.WinningOption)
		assert.Equal(t, 1, *status.WinningOption)
		assert.True(t, status.CanFinalize) // 6/10 > 50%
	})

	t.Run("GetVotingStatus_NoMajority", func(t *testing.T) {
		ts.DB.Truncate(t)

		// Create users and lobby with voting enabled
		lobby, users := testutil.SeedLobbyWith10ReadyPlayers(t, ts.DB.DB)
		require.NotNil(t, lobby)

		// Enable voting on the lobby
		lobby.VotingEnabled = true
		lobby.VotingMode = domain.VotingModeMajority
		lobby.Status = domain.LobbyStatusMatchmaking
		err := ts.Repos.Lobby.Update(ctx, lobby)
		require.NoError(t, err)

		// Create match options
		for i := 1; i <= 2; i++ {
			option := &domain.MatchOption{
				LobbyID:      lobby.ID,
				OptionNumber: i,
			}
			err := ts.Repos.MatchOption.Create(ctx, option)
			require.NoError(t, err)
		}

		// Cast votes: 5 for option 1, 5 for option 2 (no majority)
		for i := 0; i < 5; i++ {
			_, err := ts.Services.Lobby.CastVote(ctx, lobby.ID, users[i].ID, 1)
			require.NoError(t, err)
		}
		for i := 5; i < 10; i++ {
			_, err := ts.Services.Lobby.CastVote(ctx, lobby.ID, users[i].ID, 2)
			require.NoError(t, err)
		}

		// Check voting status
		status, err := ts.Services.Lobby.GetVotingStatus(ctx, lobby.ID, nil)
		require.NoError(t, err)
		// Tie goes to lowest option number
		require.NotNil(t, status.WinningOption)
		assert.Equal(t, 1, *status.WinningOption)
		assert.False(t, status.CanFinalize) // 5/10 = 50%, not > 50%
	})

	t.Run("GetVotingStatus_Unanimous", func(t *testing.T) {
		ts.DB.Truncate(t)

		// Create users and lobby with unanimous voting
		lobby, users := testutil.SeedLobbyWith10ReadyPlayers(t, ts.DB.DB)
		require.NotNil(t, lobby)

		// Enable voting on the lobby
		lobby.VotingEnabled = true
		lobby.VotingMode = domain.VotingModeUnanimous
		lobby.Status = domain.LobbyStatusMatchmaking
		err := ts.Repos.Lobby.Update(ctx, lobby)
		require.NoError(t, err)

		// Create match option
		option := &domain.MatchOption{
			LobbyID:      lobby.ID,
			OptionNumber: 1,
		}
		err = ts.Repos.MatchOption.Create(ctx, option)
		require.NoError(t, err)

		// Cast 9 votes for option 1 (not unanimous)
		for i := 0; i < 9; i++ {
			_, err := ts.Services.Lobby.CastVote(ctx, lobby.ID, users[i].ID, 1)
			require.NoError(t, err)
		}

		status, err := ts.Services.Lobby.GetVotingStatus(ctx, lobby.ID, nil)
		require.NoError(t, err)
		assert.False(t, status.CanFinalize) // Need 10/10 for unanimous

		// Cast last vote
		_, err = ts.Services.Lobby.CastVote(ctx, lobby.ID, users[9].ID, 1)
		require.NoError(t, err)

		status, err = ts.Services.Lobby.GetVotingStatus(ctx, lobby.ID, nil)
		require.NoError(t, err)
		assert.True(t, status.CanFinalize) // 10/10 unanimous
	})

	t.Run("GetVotingStatus_ReturnsVotersList", func(t *testing.T) {
		ts.DB.Truncate(t)

		// Create users and lobby with voting enabled
		lobby, users := testutil.SeedLobbyWith10ReadyPlayers(t, ts.DB.DB)
		require.NotNil(t, lobby)

		// Enable voting on the lobby
		lobby.VotingEnabled = true
		lobby.VotingMode = domain.VotingModeMajority
		lobby.Status = domain.LobbyStatusMatchmaking
		err := ts.Repos.Lobby.Update(ctx, lobby)
		require.NoError(t, err)

		// Create match options
		for i := 1; i <= 2; i++ {
			option := &domain.MatchOption{
				LobbyID:      lobby.ID,
				OptionNumber: i,
			}
			err := ts.Repos.MatchOption.Create(ctx, option)
			require.NoError(t, err)
		}

		// Cast votes: 3 for option 1, 2 for option 2
		for i := 0; i < 3; i++ {
			_, err := ts.Services.Lobby.CastVote(ctx, lobby.ID, users[i].ID, 1)
			require.NoError(t, err)
		}
		for i := 3; i < 5; i++ {
			_, err := ts.Services.Lobby.CastVote(ctx, lobby.ID, users[i].ID, 2)
			require.NoError(t, err)
		}

		// Check voting status includes voters
		status, err := ts.Services.Lobby.GetVotingStatus(ctx, lobby.ID, nil)
		require.NoError(t, err)

		// Verify voters map is populated
		require.NotNil(t, status.Voters)
		assert.Len(t, status.Voters[1], 3) // 3 voters for option 1
		assert.Len(t, status.Voters[2], 2) // 2 voters for option 2

		// Verify voter info contains user IDs and display names
		for _, voter := range status.Voters[1] {
			assert.NotEmpty(t, voter.UserID)
			assert.NotEmpty(t, voter.DisplayName)
		}
		for _, voter := range status.Voters[2] {
			assert.NotEmpty(t, voter.UserID)
			assert.NotEmpty(t, voter.DisplayName)
		}

		// Verify specific users are in the correct voter lists
		option1VoterIDs := make(map[string]bool)
		for _, voter := range status.Voters[1] {
			option1VoterIDs[voter.UserID.String()] = true
		}
		for i := 0; i < 3; i++ {
			assert.True(t, option1VoterIDs[users[i].ID.String()], "User %d should be in option 1 voters", i)
		}

		option2VoterIDs := make(map[string]bool)
		for _, voter := range status.Voters[2] {
			option2VoterIDs[voter.UserID.String()] = true
		}
		for i := 3; i < 5; i++ {
			assert.True(t, option2VoterIDs[users[i].ID.String()], "User %d should be in option 2 voters", i)
		}
	})
}
