package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// ErrTMDBNotFound is returned when a media entry exists in the cache but has no tmdbId
var ErrTMDBNotFound = errors.New("tmdb id not found in cache")

// TMDBCastMember represents a cast member (actor) from TMDB
type TMDBCastMember struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Character   string `json:"character"`
	ProfilePath string `json:"profile_path"`
	Order       int    `json:"order"`
}

// FetchTMDBCast fetches cast info from TMDB for a given media type and TMDB id
func FetchTMDBCast(mediaType MediaType, tmdbId int, tmdbKey string) ([]TMDBCastMember, error) {
	var url string
	switch mediaType {
	case MediaTypeMovie:
		url = fmt.Sprintf("https://api.themoviedb.org/3/movie/%d/credits?api_key=%s", tmdbId, tmdbKey)
	case MediaTypeTV:
		url = fmt.Sprintf("https://api.themoviedb.org/3/tv/%d/credits?api_key=%s", tmdbId, tmdbKey)
	default:
		return nil, fmt.Errorf("unsupported mediaType for cast fetch: %s", mediaType)
	}
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result struct {
		Cast []TMDBCastMember `json:"cast"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Cast, nil
}

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

func GetTMDBId(mediaType MediaType, mediaId int) (int, error) {
	var cachePath string
	switch mediaType {
	case MediaTypeMovie:
		cachePath = MoviesRedisKey
	case MediaTypeTV:
		cachePath = SeriesRedisKey
	}

	tmdb, err := getCachedTMDBId(cachePath, mediaId)
	if err == nil {
		return tmdb, nil
	}
	if errors.Is(err, ErrTMDBNotFound) {
		TrailarrLog(WARN, "TMDB", "TMDB id not found for %s id %d", mediaType, mediaId)
		return 0, fmt.Errorf("TMDB id not found for %s id %d", mediaType, mediaId)
	}

	return 0, err
}

// getCachedTMDBId checks a JSON cache path (Radarr/Sonarr JSON dumps) for a matching id
// and returns the stored tmdbId if present. Returns (tmdbId, found, error).
func getCachedTMDBId(cachePath string, mediaId int) (int, error) {
	var items []map[string]interface{}
	items, err := LoadMediaFromRedis(cachePath)
	if err != nil {
		return 0, fmt.Errorf("failed to read or decode cache %s: %w", cachePath, err)
	}
	for _, it := range items {
		if iid, ok := it["id"].(float64); ok && int(iid) == mediaId {
			if tmdb, ok := it["tmdbId"].(float64); ok && int(tmdb) != 0 {
				return int(tmdb), nil
			}
			return 0, ErrTMDBNotFound
		}
	}
	return 0, ErrTMDBNotFound
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
				ID:         r.ID,
				ExtraType:  r.Type,
				ExtraTitle: r.Name,
				YoutubeId:  r.Key,
			})
		}
	}
	return extras, nil
}
