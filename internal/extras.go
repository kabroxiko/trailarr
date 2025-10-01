package internal

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
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
	tmdbKey, err := GetTMDBKey()
	if err != nil {
		return nil, err
	}

	tmdbId, err := GetTMDBId(mediaType, id, tmdbKey)
	if err != nil {
		return nil, err
	}

	extras, err := FetchTMDBExtras(mediaType, tmdbId, tmdbKey)
	if err != nil {
		return nil, err
	}

	return extras, nil
}

func canonicalizeExtraType(extraType, name string) string {
	if extraType == "" {
		if name != "" {
			extraType = name
		} else {
			extraType = "Video"
		}
	}
	switch extraType {
	case "Behind the Scenes":
		return string(ExtraTypeBehindTheScenes)
	case "Featurette":
		return string(ExtraTypeFeaturettes)
	case "Clip":
		return string(ExtraTypeScenes)
	case "Trailer", "Teaser":
		return string(ExtraTypeTrailers)
	case "Bloopers":
		return string(ExtraTypeOthers)
	default:
		return extraType
	}
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

// ExtraType enum

type ExtraType string

const (
	ExtraTypeBehindTheScenes ExtraType = "Behind The Scenes"
	ExtraTypeFeaturettes     ExtraType = "Featurettes"
	ExtraTypeScenes          ExtraType = "Scenes"
	ExtraTypeTrailers        ExtraType = "Trailers"
	ExtraTypeOthers          ExtraType = "Others"
)

var extraTypes = map[ExtraType]bool{
	ExtraTypeBehindTheScenes: true,
	ExtraTypeFeaturettes:     true,
	ExtraTypeScenes:          true,
	ExtraTypeTrailers:        true,
	ExtraTypeOthers:          true,
}
