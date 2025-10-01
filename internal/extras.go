package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler to delete an extra and record history
func deleteExtraHandler(c *gin.Context) {
	var req struct {
		MediaType  string `json:"mediaType"`
		MediaId    int    `json:"mediaId"`
		ExtraType  string `json:"extraType"`
		ExtraTitle string `json:"extraTitle"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRequest})
		return
	}
	// Resolve media path from mediaType and mediaId
	var cachePath string
	if req.MediaType == "movie" {
		cachePath = MoviesCachePath
	} else if req.MediaType == "tv" {
		cachePath = SeriesCachePath
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mediaType"})
		return
	}
	mediaPath, err := FindMediaPathByID(cachePath, fmt.Sprintf("%d", req.MediaId))
	if err != nil || mediaPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
		return
	}
	// Get media title from cache
	var mediaTitle string
	items, err := loadCache(cachePath)
	if err == nil {
		for _, m := range items {
			if mid, ok := m["id"]; ok && fmt.Sprintf("%v", mid) == fmt.Sprintf("%d", req.MediaId) {
				if t, ok := m["title"].(string); ok {
					mediaTitle = t
				}
				break
			}
		}
	}
	extraDir := mediaPath + "/" + req.ExtraType
	extraFile := extraDir + "/" + SanitizeFilename(req.ExtraTitle) + ".mp4"
	metaFile := extraDir + "/" + SanitizeFilename(req.ExtraTitle) + ".mp4.json"
	err1 := os.Remove(extraFile)
	err2 := os.Remove(metaFile)
	if err1 != nil && err2 != nil {
		fmt.Printf("[deleteExtraHandler] Failed to delete extra: file error: %v, meta error: %v\n", err1, err2)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete extra", "detail": fmt.Sprintf("file error: %v, meta error: %v", err1, err2)})
		return
	}
	// Record history event
	event := HistoryEvent{
		Action:     "delete",
		Title:      mediaTitle,
		MediaType:  req.MediaType,
		ExtraType:  req.ExtraType,
		ExtraTitle: req.ExtraTitle,
		Date:       time.Now(),
	}
	_ = AppendHistoryEvent(event)
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

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
	// Record history event
	var mediaTitle string
	// Try to resolve title from cache
	var cachePath string
	if strings.Contains(req.MoviePath, "/Movies/") {
		cachePath = MoviesCachePath
	} else if strings.Contains(req.MoviePath, "/Series/") {
		cachePath = SeriesCachePath
	}
	if cachePath != "" {
		items, err := loadCache(cachePath)
		if err == nil {
			for _, m := range items {
				if p, ok := m["path"].(string); ok && p == req.MoviePath {
					if t, ok := m["title"].(string); ok {
						mediaTitle = t
					}
					break
				}
			}
		}
	}
	event := HistoryEvent{
		Action:     "download",
		Title:      mediaTitle,
		MediaType:  "movie", // Could be "movie" or "tv"; adjust as needed
		ExtraType:  req.ExtraType,
		ExtraTitle: req.ExtraTitle,
		Date:       time.Now(),
	}
	_ = AppendHistoryEvent(event)
	c.JSON(http.StatusOK, gin.H{"status": "downloaded", "meta": meta})
}

// Utility: Recursively find all media paths with Trailers containing video files (with debug logging)
func findMediaWithTrailers(baseDirs ...string) map[string]bool {
	found := make(map[string]bool)
	for _, baseDir := range baseDirs {
		_ = filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() {
				return nil
			}
			if filepath.Base(path) == "Trailers" {
				entries, _ := os.ReadDir(path)
				for _, entry := range entries {
					if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".mp4") || strings.HasSuffix(entry.Name(), ".mkv") || strings.HasSuffix(entry.Name(), ".avi")) {
						parent := filepath.Dir(path)
						found[parent] = true
						break
					}
				}
			}
			return nil
		})
	}
	return found
}
