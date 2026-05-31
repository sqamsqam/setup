package github

import (
	"encoding/json"
	"fmt"
	"io"
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

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Second * time.Duration(attempt))
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			lastErr = fmt.Errorf("create request: %w", err)
			continue
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		token := os.Getenv("GITHUB_TOKEN")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("fetch releases: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			var release Release
			if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
				resp.Body.Close()
				return "", fmt.Errorf("decode release JSON: %w", err)
			}
			resp.Body.Close()

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

		if resp.StatusCode == http.StatusForbidden {
			remaining := resp.Header.Get("X-RateLimit-Remaining")
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if remaining == "0" || remaining == "" {
				return "", fmt.Errorf("GitHub API rate limit exceeded. Set GITHUB_TOKEN environment variable to increase the limit.")
			}
			return "", fmt.Errorf("unexpected status %d from GitHub API: %s", resp.StatusCode, string(body))
		}

		resp.Body.Close()

		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("unexpected status %d from GitHub API", resp.StatusCode)
			continue
		}

		lastErr = fmt.Errorf("unexpected status %d from GitHub API", resp.StatusCode)
		break
	}

	return "", fmt.Errorf("GitHub API request failed after 3 attempts: %w", lastErr)
}
