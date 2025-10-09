package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Common TMDB extra types (singular)
var TMDBExtraTypes = []string{
	"Trailer",
	"Teaser",
	"Clip",
	"Featurette",
	"Behind the Scene",
	"Bloopers",
	"Opening Credit",
	"Recap",
	"Interview",
	"Scene",
	"Promo",
	"Short",
	"Music Video",
	"Commercial",
	"Other",
}

func GetTMDBKey() (string, error) {
	if Config == nil {
		TrailarrLog(WARN, "TMDB", "Config not loaded")
		return "", fmt.Errorf("config not loaded")
	}
	general, ok := Config["general"].(map[string]interface{})
	if !ok {
		TrailarrLog(WARN, "TMDB", "general section missing in config")
		return "", fmt.Errorf("general section missing in config")
	}
	tmdbKey, ok := general["tmdbKey"].(string)
	if !ok || tmdbKey == "" {
		TrailarrLog(WARN, "TMDB", "TMDB key not set in general settings")
		return "", fmt.Errorf("TMDB key not set in general settings")
	}
	return tmdbKey, nil
}

func GetTMDBId(mediaType MediaType, id int, tmdbKey string) (int, error) {
	switch mediaType {
	case MediaTypeMovie:
		return GetMovieTMDBId(id)
	case MediaTypeTV:
		return GetTVTMDBId(id, tmdbKey)
	default:
		TrailarrLog(WARN, "TMDB", "unknown mediaType: %s", mediaType)
		return 0, fmt.Errorf("unknown mediaType: %s", mediaType)
	}
}

func GetMovieTMDBId(id int) (int, error) {
	var movies []map[string]interface{}
	if err := ReadJSONFile(MoviesJSONPath, &movies); err != nil {
		return 0, fmt.Errorf("failed to read or decode Radarr cache: %w", err)
	}
	for _, m := range movies {
		if mid, ok := m["id"].(float64); ok && int(mid) == id {
			if tmdb, ok := m["tmdbId"].(float64); ok {
				return int(tmdb), nil
			}
			break
		}
	}
	TrailarrLog(WARN, "TMDB", "TMDB id not found for Radarr movie id %d", id)
	return 0, fmt.Errorf("TMDB id not found for Radarr movie id %d", id)
}

func GetTVTMDBId(id int, tmdbKey string) (int, error) {
	var series []map[string]interface{}
	if err := ReadJSONFile(SeriesJSONPath, &series); err != nil {
		return 0, fmt.Errorf("failed to read or decode Sonarr cache: %w", err)
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
		TrailarrLog(WARN, "TMDB", "title not found for Sonarr series id %d", id)
		return 0, fmt.Errorf("title not found for Sonarr series id %d", id)
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
		TrailarrLog(WARN, "TMDB", "no TMDB TV series found for title %s", title)
		return 0, fmt.Errorf("no TMDB TV series found for title %s", title)
	}
	return tmdbResult.Results[0].ID, nil
}

func FetchTMDBExtras(mediaType MediaType, tmdbId int, tmdbKey string) ([]Extra, error) {
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
	extras := make([]Extra, 0)
	for _, r := range result.Results {
		if r.Site == "YouTube" {
			extras = append(extras, Extra{
				ID:        r.ID,
				Type:      r.Type,
				Title:     r.Name,
				YoutubeId: r.Key,
				// URL:   fmt.Sprintf("https://www.youtube.com/watch?v=%s", r.Key),
			})
		}
	}
	return extras, nil
}
