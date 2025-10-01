package internal

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
)

// Plex API integration
type PlexItem struct {
	Title    string
	Type     string // "movie" or "series"
	Language string
	Extras   []string
}

// Fetch Plex library items (requires Plex token and server URL)
func FetchPlexLibrary() ([]PlexItem, error) {
	plexToken := os.Getenv("PLEX_TOKEN")
	plexURL := os.Getenv("PLEX_URL")
	if plexToken == "" || plexURL == "" {
		return nil, fmt.Errorf("PLEX_TOKEN or PLEX_URL not set")
	}
	// Example: Get all movies
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/library/sections/1/all", plexURL), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Plex-Token", plexToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// Parse XML response
	type Directory struct {
		Title string `xml:"title,attr"`
		Type  string `xml:"type,attr"`
		// Add more fields as needed
	}
	type MediaContainer struct {
		XMLName     xml.Name    `xml:"MediaContainer"`
		Directories []Directory `xml:"Video"`
	}
	var container MediaContainer
	if err := xml.Unmarshal(body, &container); err != nil {
		return nil, err
	}
	items := []PlexItem{}
	for _, d := range container.Directories {
		items = append(items, PlexItem{
			Title:    d.Title,
			Type:     d.Type,
			Language: "Unknown", // Plex XML may have language info in a different field
			Extras:   []string{},
		})
	}
	return items, nil
}

// Placeholder for extras search and download logic
func SearchExtras(movieTitle string) ([]map[string]string, error) {
	// TMDB API integration
	tmdbKey := "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	tmdbURL := fmt.Sprintf("https://api.themoviedb.org/3/search/movie?api_key=%s&query=%s", tmdbKey, movieTitle)
	resp, err := http.Get(tmdbURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var tmdbResult struct {
		Results []struct {
			Title string `json:"title"`
			ID    int    `json:"id"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &tmdbResult); err != nil {
		return nil, err
	}
	if len(tmdbResult.Results) == 0 {
		return nil, fmt.Errorf("no movie found")
	}
	movieID := tmdbResult.Results[0].ID

	// Get videos (trailers, featurettes) from TMDB
	videosURL := fmt.Sprintf("https://api.themoviedb.org/3/movie/%d/videos?api_key=%s", movieID, tmdbKey)
	resp, err = http.Get(videosURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var videosResult struct {
		Results []struct {
			Name string `json:"name"`
			Type string `json:"type"`
			Site string `json:"site"`
			Key  string `json:"key"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &videosResult); err != nil {
		return nil, err
	}
	extras := []map[string]string{}
	for _, v := range videosResult.Results {
		if v.Site == "YouTube" {
			extraType := v.Type
			if extraType == "" {
				// fallback: try to infer from name
				if name := v.Name; name != "" {
					extraType = name
				} else {
					extraType = "Video"
				}
			}
			extras = append(extras, map[string]string{
				"type":  extraType,
				"title": v.Name,
				"url":   fmt.Sprintf("https://www.youtube.com/watch?v=%s", v.Key),
			})
		}
	}

	// YouTube search (no API key, basic search URL)
	extras = append(extras, map[string]string{
		"type":  "YouTube Search",
		"title": "Search for trailers",
		"url":   fmt.Sprintf("https://www.youtube.com/results?search_query=%s+trailer", movieTitle),
	})
	return extras, nil
}

func DownloadExtra(extraURL string) error {
	// Native download: fetch video page and save (basic, for YouTube)
	resp, err := http.Get(extraURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	// This is a placeholder: actual YouTube video download requires parsing the page and extracting video URLs
	// For robust downloading, use a Go library like github.com/kkdai/youtube
	// Here, just save the HTML page for demonstration
	filename := "video.html"
	if err := os.WriteFile(filename, body, 0644); err != nil {
		return err
	}
	fmt.Println("Saved video page to", filename)
	return nil
}
