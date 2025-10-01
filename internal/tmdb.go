package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"gopkg.in/yaml.v3"
)

func GetTMDBKey() (string, error) {
	settingsData, err := os.ReadFile(ConfigPath)
	if err != nil {
		return "", fmt.Errorf("Failed to read config: %w", err)
	}
	var allSettings struct {
		General struct {
			TMDBKey string `yaml:"tmdbKey"`
		} `yaml:"general"`
	}
	if err := yaml.Unmarshal(settingsData, &allSettings); err != nil {
		return "", fmt.Errorf("Failed to decode config: %w", err)
	}
	if allSettings.General.TMDBKey == "" {
		return "", fmt.Errorf("TMDB key not set in general settings")
	}
	return allSettings.General.TMDBKey, nil
}

func GetTMDBId(mediaType string, id int, tmdbKey string) (int, error) {
	switch mediaType {
	case "movie":
		return GetMovieTMDBId(id)
	case "tv":
		return GetTVTMDBId(id, tmdbKey)
	default:
		return 0, fmt.Errorf("Unknown mediaType: %s", mediaType)
	}
}

func GetMovieTMDBId(id int) (int, error) {
	radarrCache := MoviesCachePath
	cacheData, err := os.ReadFile(radarrCache)
	if err != nil {
		return 0, fmt.Errorf("Failed to read Radarr cache: %w", err)
	}
	var movies []map[string]interface{}
	if err := json.Unmarshal(cacheData, &movies); err != nil {
		return 0, fmt.Errorf("Failed to decode Radarr cache: %w", err)
	}
	for _, m := range movies {
		if mid, ok := m["id"].(float64); ok && int(mid) == id {
			if tmdb, ok := m["tmdbId"].(float64); ok {
				return int(tmdb), nil
			}
			break
		}
	}
	return 0, fmt.Errorf("TMDB id not found for Radarr movie id %d", id)
}

func GetTVTMDBId(id int, tmdbKey string) (int, error) {
	sonarrCache := SeriesCachePath
	cacheData, err := os.ReadFile(sonarrCache)
	if err != nil {
		return 0, fmt.Errorf("Failed to read Sonarr cache: %w", err)
	}
	var series []map[string]interface{}
	if err := json.Unmarshal(cacheData, &series); err != nil {
		return 0, fmt.Errorf("Failed to decode Sonarr cache: %w", err)
	}
	var title string
	for _, s := range series {
		if sid, ok := s["id"].(float64); ok && int(sid) == id {
			if t, ok := s["title"].(string); ok {
				title = t
			}
			break
		}
	}
	if title == "" {
		return 0, fmt.Errorf("Title not found for Sonarr series id %d", id)
	}
	tmdbSearchURL := fmt.Sprintf("https://api.themoviedb.org/3/search/tv?api_key=%s&query=%s", tmdbKey, url.QueryEscape(title))
	resp, err := http.Get(tmdbSearchURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	var tmdbResult struct {
		Results []struct {
			ID int `json:"id"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &tmdbResult); err != nil {
		return 0, err
	}
	if len(tmdbResult.Results) == 0 {
		return 0, fmt.Errorf("No TMDB TV series found for title %s", title)
	}
	return tmdbResult.Results[0].ID, nil
}

func FetchTMDBExtras(mediaType string, tmdbId int, tmdbKey string) ([]map[string]string, error) {
	videosURL := fmt.Sprintf("https://api.themoviedb.org/3/%s/%d/videos?api_key=%s", mediaType, tmdbId, tmdbKey)
	resp, err := http.Get(videosURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result struct {
		Results []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Key  string `json:"key"`
			Site string `json:"site"`
			Type string `json:"type"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	extras := make([]map[string]string, 0)
	for _, r := range result.Results {
		if r.Site == "YouTube" {
			extras = append(extras, map[string]string{
				"id":    r.ID,
				"type":  r.Type,
				"title": r.Name,
				"url":   fmt.Sprintf("https://www.youtube.com/watch?v=%s", r.Key),
			})
		}
	}
	return extras, nil
}
