package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"
)

type Release struct {
	Assets []Asset `json:"assets"`
}

type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}

var (
	httpClient = &http.Client{Timeout: 30 * time.Second}
	apiBase    = "https://api.github.com"
)

func SetHTTPClient(c *http.Client) {
	httpClient = c
}

func SetAPIBase(base string) {
	apiBase = base
}

func LatestReleaseAsset(repo, pattern string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/releases/latest", apiBase, repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d from GitHub API", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("decode release JSON: %w", err)
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("compile pattern: %w", err)
	}

	for _, a := range release.Assets {
		if re.MatchString(a.Name) {
			return a.DownloadURL, nil
		}
	}

	return "", fmt.Errorf("no asset matching %q found in %s release", pattern, repo)
}
