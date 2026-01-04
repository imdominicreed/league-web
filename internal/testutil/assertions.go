package testutil

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertStatusCode verifies the HTTP response status code
func AssertStatusCode(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	assert.Equal(t, expected, resp.StatusCode, "unexpected status code")
}

// AssertJSONResponse decodes JSON response into v and verifies success
func AssertJSONResponse(t *testing.T, resp *http.Response, v interface{}) {
	t.Helper()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")

	err = json.Unmarshal(body, v)
	require.NoError(t, err, "failed to unmarshal response: %s", string(body))
}

// AssertErrorResponse verifies error response with expected status and message
func AssertErrorResponse(t *testing.T, resp *http.Response, expectedStatus int, expectedMessage string) {
	t.Helper()

	assert.Equal(t, expectedStatus, resp.StatusCode, "unexpected status code")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "failed to read response body")

	// Error responses are plain text in this API
	assert.Contains(t, string(body), expectedMessage, "error message mismatch")
}

// AssertContainsChampion verifies a champion ID exists in a slice
func AssertContainsChampion(t *testing.T, picks []string, championID string) {
	t.Helper()
	assert.Contains(t, picks, championID, "champion %s not found in picks", championID)
}

// AssertNotContainsChampion verifies a champion ID does not exist in a slice
func AssertNotContainsChampion(t *testing.T, picks []string, championID string) {
	t.Helper()
	assert.NotContains(t, picks, championID, "champion %s should not be in picks", championID)
}

// AssertDraftPhase verifies the draft phase state
func AssertDraftPhase(t *testing.T, phase int, expectedPhase int, team string, expectedTeam string, action string, expectedAction string) {
	t.Helper()
	assert.Equal(t, expectedPhase, phase, "unexpected phase")
	assert.Equal(t, expectedTeam, team, "unexpected team")
	assert.Equal(t, expectedAction, action, "unexpected action")
}

// RequireNoError fails immediately if err is not nil
func RequireNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	t.Helper()
	require.NoError(t, err, msgAndArgs...)
}

// RequireEqual fails immediately if expected != actual
func RequireEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	require.Equal(t, expected, actual, msgAndArgs...)
}

// AssertEqual checks if expected == actual
func AssertEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Equal(t, expected, actual, msgAndArgs...)
}

// AssertNotNil checks if object is not nil
func AssertNotNil(t *testing.T, object interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	assert.NotNil(t, object, msgAndArgs...)
}

// AssertNil checks if object is nil
func AssertNil(t *testing.T, object interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Nil(t, object, msgAndArgs...)
}

// AssertTrue checks if value is true
func AssertTrue(t *testing.T, value bool, msgAndArgs ...interface{}) {
	t.Helper()
	assert.True(t, value, msgAndArgs...)
}

// AssertFalse checks if value is false
func AssertFalse(t *testing.T, value bool, msgAndArgs ...interface{}) {
	t.Helper()
	assert.False(t, value, msgAndArgs...)
}

// AssertLen checks if object has expected length
func AssertLen(t *testing.T, object interface{}, length int, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Len(t, object, length, msgAndArgs...)
}
