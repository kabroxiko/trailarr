package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

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

// Sanitize filename for OS conflicts (remove/replace invalid chars)
func SanitizeFilename(name string) string {
	// Remove any character not allowed in filenames
	// Windows: \/:*?"<>|, Linux: /
	re := regexp.MustCompile(`[\\/:*?"<>|]`)
	name = re.ReplaceAllString(name, "_")
	name = strings.TrimSpace(name)
	return name
}

// Handler to list existing extras for a movie path
func existingExtrasHandler(c *gin.Context) {
	moviePath := c.Query("moviePath")
	if moviePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "moviePath required"})
		return
	}
	// Scan subfolders for .mp4 files and their metadata
	var existing []map[string]interface{}
	entries, err := os.ReadDir(moviePath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"existing": []map[string]interface{}{}})
		return
	}
	// Track duplicate index for each type/title
	dupCount := make(map[string]int)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subdir := moviePath + "/" + entry.Name()
		files, _ := os.ReadDir(subdir)
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".mp4") {
				metaFile := subdir + "/" + strings.TrimSuffix(f.Name(), ".mp4") + ".mp4.json"
				var meta struct {
					Type      string `json:"type"`
					Title     string `json:"title"`
					YouTubeID string `json:"youtube_id"`
				}
				status := "not-downloaded"
				if metaBytes, err := os.ReadFile(metaFile); err == nil {
					_ = json.Unmarshal(metaBytes, &meta)
					status = "downloaded"
				}
				key := entry.Name() + "|" + meta.Title
				dupCount[key]++
				existing = append(existing, map[string]interface{}{
					"type":       entry.Name(),
					"title":      meta.Title,
					"youtube_id": meta.YouTubeID,
					"_dupIndex":  dupCount[key],
					"status":     status,
				})
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"existing": existing})
}

func downloadExtraHandler(c *gin.Context) {
	var req struct {
		MoviePath  string `json:"moviePath"`
		ExtraType  string `json:"extraType"`
		ExtraTitle string `json:"extraTitle"`
		URL        string `json:"url"`
	}
	if err := c.BindJSON(&req); err != nil {
		fmt.Printf("[downloadExtraHandler] Invalid request: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRequest})
		return
	}
	fmt.Printf("[downloadExtraHandler] Download request: moviePath=%s, extraType=%s, extraTitle=%s, url=%s\n", req.MoviePath, req.ExtraType, req.ExtraTitle, req.URL)
	meta, err := DownloadYouTubeExtra(req.MoviePath, req.ExtraType, req.ExtraTitle, req.URL)
	if err != nil {
		fmt.Printf("[downloadExtraHandler] Download error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "downloaded", "meta": meta})
}
