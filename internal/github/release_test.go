package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLatestReleaseAsset(t *testing.T) {
	oldAPIBase := apiBase
	oldClient := httpClient
	t.Cleanup(func() {
		apiBase = oldAPIBase
		httpClient = oldClient
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := Release{
			Assets: []Asset{
				{Name: "ripgrep_14.1.0_amd64.deb", DownloadURL: "https://example.com/rg.deb"},
				{Name: "ripgrep_14.1.0_arm64.deb", DownloadURL: "https://example.com/rg-arm.deb"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	SetAPIBase(server.URL)
	SetHTTPClient(server.Client())
	t.Setenv("GITHUB_TOKEN", "")

	u, err := LatestReleaseAsset("BurntSushi/ripgrep", `ripgrep_.*_amd64\.deb$`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u != "https://example.com/rg.deb" {
		t.Errorf("unexpected URL: %s", u)
	}
}

func TestLatestReleaseAssetNoMatch(t *testing.T) {
	oldAPIBase := apiBase
	oldClient := httpClient
	t.Cleanup(func() {
		apiBase = oldAPIBase
		httpClient = oldClient
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := Release{
			Assets: []Asset{
				{Name: "ripgrep_14.1.0_arm64.deb", DownloadURL: "https://example.com/rg-arm.deb"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	SetAPIBase(server.URL)
	SetHTTPClient(server.Client())
	t.Setenv("GITHUB_TOKEN", "")

	_, err := LatestReleaseAsset("BurntSushi/ripgrep", `amd64\.deb$`)
	if err == nil {
		t.Fatal("expected error for no matching asset")
	}
}

func TestLatestReleaseAssetInvalidPattern(t *testing.T) {
	_, err := LatestReleaseAsset("owner/repo", `[invalid`)
	if err == nil {
		t.Fatal("expected error for invalid regex pattern")
	}
}
