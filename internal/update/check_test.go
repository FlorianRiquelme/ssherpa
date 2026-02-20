package update

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckForUpdate_NewVersionAvailable(t *testing.T) {
	// Mock GitHub API
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := githubRelease{TagName: "v0.3.0"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer apiServer.Close()

	// Mock raw changelog
	rawServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testChangelog))
	}))
	defer rawServer.Close()

	info, err := checkRemote("0.1.0", apiServer.URL, rawServer.URL+"/CHANGELOG.md")
	require.NoError(t, err)
	require.NotNil(t, info)
	assert.Equal(t, "0.3.0", info.LatestVersion)
	assert.Len(t, info.Changes, 2) // 0.3.0 and 0.2.0
}

func TestCheckForUpdate_AlreadyLatest(t *testing.T) {
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := githubRelease{TagName: "v0.1.0"}
		json.NewEncoder(w).Encode(resp)
	}))
	defer apiServer.Close()

	info, err := checkRemote("0.1.0", apiServer.URL, "")
	require.NoError(t, err)
	assert.Nil(t, info) // No update
}

func TestCheckForUpdate_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	info, err := checkRemote("0.1.0", server.URL, "")
	assert.Error(t, err)
	assert.Nil(t, info)
}
