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

	cachePath, err := resolveCachePath(req.MediaType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	mediaPath, err := FindMediaPathByID(cachePath, fmt.Sprintf("%d", req.MediaId))
	if err != nil || mediaPath == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
		return
	}

	mediaTitle := lookupMediaTitle(cachePath, req.MediaId)

	if err := deleteExtraFiles(mediaPath, req.ExtraType, req.ExtraTitle); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete extra", "detail": err.Error()})
		return
	}

	recordDeleteHistory(mediaTitle, req.MediaType, req.ExtraType, req.ExtraTitle)
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func resolveCachePath(mediaType string) (string, error) {
	switch mediaType {
	case "movie":
		return TrailarrRoot + "/movies.json", nil
	case "tv":
		return TrailarrRoot + "/series.json", nil
	default:
		TrailarrLog("Warn", "Extras", "Invalid mediaType: %s", mediaType)
		return "", fmt.Errorf("invalid mediaType")
	}
}

func lookupMediaTitle(cachePath string, mediaId int) string {
	items, err := loadCache(cachePath)
	if err != nil {
		return ""
	}
	for _, m := range items {
		if mid, ok := m["id"]; ok && fmt.Sprintf("%v", mid) == fmt.Sprintf("%d", mediaId) {
			if t, ok := m["title"].(string); ok {
				return t
			}
		}
	}
	return ""
}

func deleteExtraFiles(mediaPath, extraType, extraTitle string) error {
	extraDir := mediaPath + "/" + extraType
	extraFile := extraDir + "/" + SanitizeFilename(extraTitle) + ".mkv"
	metaFile := extraDir + "/" + SanitizeFilename(extraTitle) + ".mkv.json"
	err1 := os.Remove(extraFile)
	err2 := os.Remove(metaFile)
	if err1 != nil && err2 != nil {
		TrailarrLog("Warn", "Extras", "Failed to delete extra files: %v, meta: %v", err1, err2)
		return fmt.Errorf("file error: %v, meta error: %v", err1, err2)
	}
	return nil
}

func recordDeleteHistory(mediaTitle, mediaType, extraType, extraTitle string) {
	event := HistoryEvent{
		Action:     "delete",
		Title:      mediaTitle,
		MediaType:  mediaType,
		ExtraType:  extraType,
		ExtraTitle: extraTitle,
		Date:       time.Now(),
	}
	_ = AppendHistoryEvent(event)
}

type ExtraType string

const (
	ExtraTypeBehindTheScenes ExtraType = "Behind The Scenes"
	ExtraTypeFeaturettes     ExtraType = "Featurettes"
	ExtraTypeScenes          ExtraType = "Scenes"
	ExtraTypeTrailers        ExtraType = "Trailers"
	ExtraTypeOthers          ExtraType = "Others"
)

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
			TrailarrLog("Warn", "Extras", "Could not extract YouTube video ID from URL: %s", url)
			return "", fmt.Errorf("could not extract YouTube video ID from URL: %s", url)
		}
		return strings.Split(parts[1], "&")[0], nil
	} else if strings.Contains(url, "youtu.be/") {
		parts := strings.Split(url, "/")
		if len(parts) < 2 {
			TrailarrLog("Warn", "Extras", "Could not extract YouTube video ID from URL: %s", url)
			return "", fmt.Errorf("could not extract YouTube video ID from URL: %s", url)
		}
		return parts[len(parts)-1], nil
	}
	TrailarrLog("Warn", "Extras", "Not a valid YouTube URL: %s", url)
	return "", fmt.Errorf("not a valid YouTube URL: %s", url)
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
	// Scan subfolders for .mkv files and their metadata
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
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".mkv") {
				metaFile := subdir + "/" + strings.TrimSuffix(f.Name(), ".mkv") + ".mkv.json"
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
		MediaType  string `json:"mediaType"`
		MediaId    int    `json:"mediaId"`
		ExtraType  string `json:"extraType"`
		ExtraTitle string `json:"extraTitle"`
		URL        string `json:"url"`
	}
	if err := c.BindJSON(&req); err != nil {
		TrailarrLog("Warn", "Extras", "[downloadExtraHandler] Invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRequest})
		return
	}
	TrailarrLog("Info", "Extras", "[downloadExtraHandler] Download request: mediaType=%s, mediaId=%d, extraType=%s, extraTitle=%s, url=%s", req.MediaType, req.MediaId, req.ExtraType, req.ExtraTitle, req.URL)

	// Convert MediaId (int) to string for DownloadYouTubeExtra
	mediaIdStr := fmt.Sprintf("%d", req.MediaId)
	meta, err := DownloadYouTubeExtra(req.MediaType, mediaIdStr, req.ExtraType, req.ExtraTitle, req.URL, true)
	if err != nil {
		TrailarrLog("Warn", "Extras", "[downloadExtraHandler] Download error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Lookup media title from cache for history
	cachePath, _ := resolveCachePath(req.MediaType)
	mediaTitle := ""
	if cachePath != "" {
		items, _ := loadCache(cachePath)
		for _, m := range items {
			if mid, ok := m["id"]; ok && fmt.Sprintf("%v", mid) == mediaIdStr {
				if t, ok := m["title"].(string); ok {
					mediaTitle = t
					break
				}
			}
		}
	}
	if mediaTitle == "" {
		mediaTitle = "Unknown"
	}
	recordDownloadHistory(mediaTitle, req.MediaType, req.ExtraType, req.ExtraTitle)
	c.JSON(http.StatusOK, gin.H{"status": "downloaded", "meta": meta})
}

func resolveDownloadMediaTitle(mediaType, mediaName string) string {
	var cachePath string
	switch mediaType {
	case "movie":
		cachePath = TrailarrRoot + "/movies.json"
	case "series", "tv":
		cachePath = TrailarrRoot + "/series.json"
	}
	if cachePath != "" {
		items, err := loadCache(cachePath)
		if err == nil {
			for _, m := range items {
				if title, ok := m["title"].(string); ok && title == mediaName {
					return title
				}
			}
		}
	}
	return mediaName
}

func recordDownloadHistory(mediaTitle, mediaType, extraType, extraTitle string) {
	event := HistoryEvent{
		Action:     "download",
		Title:      mediaTitle,
		MediaType:  mediaType,
		ExtraType:  extraType,
		ExtraTitle: extraTitle,
		Date:       time.Now(),
	}
	_ = AppendHistoryEvent(event)
}

// Utility: Recursively find all media paths with Trailers containing video files (with debug logging)
func findMediaWithTrailers(baseDirs ...string) map[string]bool {
	found := make(map[string]bool)
	for _, baseDir := range baseDirs {
		walkMediaDirs(baseDir, found)
	}
	return found
}

func walkMediaDirs(baseDir string, found map[string]bool) {
	_ = filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		if isTrailerDir(path) && hasVideoFiles(path) {
			parent := filepath.Dir(path)
			found[parent] = true
		}
		return nil
	})
}

func isTrailerDir(path string) bool {
	return filepath.Base(path) == "Trailers"
}

func hasVideoFiles(dir string) bool {
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if !entry.IsDir() && isVideoFile(entry.Name()) {
			return true
		}
	}
	return false
}

func isVideoFile(name string) bool {
	return strings.HasSuffix(name, ".mp4") ||
		strings.HasSuffix(name, ".mkv") ||
		strings.HasSuffix(name, ".avi")
}
