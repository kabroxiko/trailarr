package internal

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kkdai/youtube/v2"
)

// ExtractYouTubeID parses a YouTube URL and returns the video ID or an error
func ExtractYouTubeID(url string) (string, error) {
	if strings.Contains(url, "youtube.com/watch?v=") {
		parts := strings.Split(url, "v=")
		if len(parts) < 2 {
			return "", fmt.Errorf("Could not extract YouTube video ID from URL: %s", url)
		}
		return strings.Split(parts[1], "&")[0], nil
	} else if strings.Contains(url, "youtu.be/") {
		parts := strings.Split(url, "/")
		if len(parts) < 2 {
			return "", fmt.Errorf("Could not extract YouTube video ID from URL: %s", url)
		}
		return parts[len(parts)-1], nil
	}
	return "", fmt.Errorf("Not a valid YouTube URL: %s", url)
}

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
func SearchExtras(mediaType string, id int) ([]map[string]string, error) {
	// Read TMDB key from settings.json
	var tmdbKey string
	settingsData, err := os.ReadFile("settings.json")
	if err == nil {
		var allSettings struct {
			General struct {
				TMDBKey string `json:"tmdbKey"`
			} `json:"general"`
		}
		if err := json.Unmarshal(settingsData, &allSettings); err == nil {
			tmdbKey = allSettings.General.TMDBKey
		}
	}
	if tmdbKey == "" {
		return nil, fmt.Errorf("TMDB key not set in general settings")
	}
	var tmdbId int
	if mediaType == "movie" {
		// Lookup TMDB id from Radarr
		radarrCache := "/var/lib/extrazarr/movies_cache.json"
		cacheData, err := os.ReadFile(radarrCache)
		if err != nil {
			return nil, fmt.Errorf("Failed to read Radarr cache: %w", err)
		}
		var movies []map[string]interface{}
		if err := json.Unmarshal(cacheData, &movies); err != nil {
			return nil, fmt.Errorf("Failed to decode Radarr cache: %w", err)
		}
		for _, m := range movies {
			if mid, ok := m["id"].(float64); ok && int(mid) == id {
				if tmdb, ok := m["tmdbId"].(float64); ok {
					tmdbId = int(tmdb)
				}
				break
			}
		}
		if tmdbId == 0 {
			return nil, fmt.Errorf("TMDB id not found for Radarr movie id %d", id)
		}
	} else if mediaType == "tv" {
		// Lookup title from Sonarr
		sonarrCache := "/var/lib/extrazarr/series_cache.json"
		cacheData, err := os.ReadFile(sonarrCache)
		if err != nil {
			return nil, fmt.Errorf("Failed to read Sonarr cache: %w", err)
		}
		var series []map[string]interface{}
		if err := json.Unmarshal(cacheData, &series); err != nil {
			return nil, fmt.Errorf("Failed to decode Sonarr cache: %w", err)
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
			return nil, fmt.Errorf("Title not found for Sonarr series id %d", id)
		}
		// Search TMDB for TV series by title
		tmdbSearchURL := fmt.Sprintf("https://api.themoviedb.org/3/search/tv?api_key=%s&query=%s", tmdbKey, url.QueryEscape(title))
		resp, err := http.Get(tmdbSearchURL)
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
				ID int `json:"id"`
			} `json:"results"`
		}
		if err := json.Unmarshal(body, &tmdbResult); err != nil {
			return nil, err
		}
		if len(tmdbResult.Results) == 0 {
			return nil, fmt.Errorf("No TMDB TV series found for title %s", title)
		}
		tmdbId = tmdbResult.Results[0].ID
	} else {
		return nil, fmt.Errorf("Unknown mediaType: %s", mediaType)
	}

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
	extraTypes := map[string]bool{
		"Behind The Scenes": true,
		"Featurettes":       true,
		"Scenes":            true,
		"Trailers":          true,
		"Others":            true,
	}
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
			// Canonicalize type
			if extraTypes["Behind The Scenes"] && extraType == "Behind the Scenes" {
				extraType = "Behind The Scenes"
			} else if extraTypes["Featurettes"] && extraType == "Featurette" {
				extraType = "Featurettes"
			} else if extraTypes["Scenes"] && extraType == "Clip" {
				extraType = "Scenes"
			} else if extraTypes["Trailers"] && (extraType == "Trailer" || extraType == "Teaser") {
				extraType = "Trailers"
			} else if extraTypes["Others"] && extraType == "Bloopers" {
				extraType = "Others"
			}
			extras = append(extras, map[string]string{
				"type":  extraType,
				"title": v.Name,
				"url":   fmt.Sprintf("https://www.youtube.com/watch?v=%s", v.Key),
			})
		}
	}
	return extras, nil
}

// Sanitize filename for OS conflicts (remove/replace invalid chars)
func SanitizeFilename(name string) string {
	// Remove any character not allowed in filenames
	// Windows: \/:*?"<>|, Linux: /
	re := regexp.MustCompile(`[\\/:*?"<>|]`)
	name = re.ReplaceAllString(name, "_")
	name = strings.TrimSpace(name)
	return name
}

// Download YouTube video and record metadata
type ExtraDownloadMetadata struct {
	Title     string `json:"title"`
	Type      string `json:"type"`
	YouTubeID string `json:"youtube_id"`
	FileName  string `json:"file_name"`
	Status    string `json:"status"`
	URL       string `json:"url"`
}

func DownloadYouTubeExtra(moviePath, extraType, extraTitle, extraURL string) (*ExtraDownloadMetadata, error) {
	youtubeID, err := ExtractYouTubeID(extraURL)
	if err != nil {
		return nil, err
	}
	fmt.Printf("[DownloadYouTubeExtra] Requested URL: %s, Extracted YouTube ID: %s\n", extraURL, youtubeID)

	// Sanitize type and title for filename
	safeType := SanitizeFilename(extraType)
	safeTitle := SanitizeFilename(extraTitle)
	outDir := filepath.Join(moviePath, safeType)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("Failed to create output dir '%s': %w", outDir, err)
	}
	// Find existing files with same title to determine incremental index
	files, _ := os.ReadDir(outDir)
	count := 1
	for _, f := range files {
		if !f.IsDir() && strings.HasPrefix(f.Name(), safeTitle) && strings.HasSuffix(f.Name(), ".mp4") {
			count++
		}
	}
	outFile := filepath.Join(outDir, fmt.Sprintf("%s (%d).mp4", safeTitle, count))

	// Download using kkdai/youtube
	client := youtube.Client{}
	video, err := client.GetVideo(youtubeID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get video info for YouTube ID '%s': %w", youtubeID, err)
	}
	formats := video.Formats.WithAudioChannels()
	if len(formats) == 0 {
		return nil, fmt.Errorf("No downloadable video format found for YouTube ID '%s'", youtubeID)
	}
	stream, _, err := client.GetStream(video, &formats[0])
	if err != nil {
		return nil, fmt.Errorf("Failed to get stream for YouTube ID '%s': %w", youtubeID, err)
	}
	f, err := os.Create(outFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to create file '%s': %w", outFile, err)
	}
	defer f.Close()
	if _, err := io.Copy(f, stream); err != nil {
		return nil, fmt.Errorf("Failed to save video to '%s': %w", outFile, err)
	}

	meta := &ExtraDownloadMetadata{
		Title:     extraTitle,
		Type:      extraType,
		YouTubeID: youtubeID,
		FileName:  outFile,
		Status:    "downloaded",
		URL:       extraURL,
	}
	// Optionally, save metadata to a file (e.g., outFile+".json")
	metaFile := outFile + ".json"
	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(metaFile, metaBytes, 0644)

	fmt.Printf("Downloaded %s to %s\n", extraTitle, outFile)
	return meta, nil
}
