package internal

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Extra struct {
	ID        string
	Type      string
	Title     string
	YoutubeId string
	Status    string
}

// GetRejectedExtrasForMedia returns rejected extras for a given media type and id
func GetRejectedExtrasForMedia(mediaType MediaType, id int) []RejectedExtra {
	rejectedPath := filepath.Join(TrailarrRoot, "rejected_extras.json")
	var rejected []RejectedExtra
	_ = ReadJSONFile(rejectedPath, &rejected)
	return Filter(rejected, func(r RejectedExtra) bool {
		return r.MediaType == mediaType && r.MediaId == id
	})
}

// Handler to delete an extra and record history
func deleteExtraHandler(c *gin.Context) {
	var req struct {
		MediaType  MediaType `json:"mediaType"`
		MediaId    int       `json:"mediaId"`
		ExtraType  string    `json:"extraType"`
		ExtraTitle string    `json:"extraTitle"`
	}
	if err := c.BindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, ErrInvalidRequest)
		return
	}

	cacheFile, _ := resolveCachePath(req.MediaType)
	mediaPath, err := FindMediaPathByID(cacheFile, req.MediaId)
	if err != nil || mediaPath == "" {
		respondError(c, http.StatusNotFound, "Media not found")
		return
	}

	if err := deleteExtraFiles(mediaPath, req.ExtraType, req.ExtraTitle); err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to delete extra: "+err.Error())
		return
	}

	recordDeleteHistory(req.MediaType, req.MediaId, req.ExtraType, req.ExtraTitle)
	respondJSON(c, http.StatusOK, gin.H{"status": "deleted"})
}

func resolveCachePath(mediaType MediaType) (string, error) {
	switch mediaType {
	case MediaTypeMovie:
		return TrailarrRoot + "/movies.json", nil
	case MediaTypeTV:
		return TrailarrRoot + "/series.json", nil
	}
	return "", fmt.Errorf("unknown media type: %v", mediaType)
}

func lookupMediaTitle(cacheFile string, mediaId int) string {
	items, err := loadCache(cacheFile)
	if CheckErrLog(WARN, "lookupMediaTitle", "Failed to load cache", err) != nil {
		return ""
	}
	for _, m := range items {
		idInt, ok := parseMediaID(m["id"])
		if ok && idInt == mediaId {
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
		CheckErrLog(WARN, "Extras", "Failed to delete extra files", err1)
		return fmt.Errorf("file error: %v, meta error: %v", err1, err2)
	}
	return nil
}

func recordDeleteHistory(mediaType MediaType, mediaId int, extraType, extraTitle string) {
	cacheFile, _ := resolveCachePath(mediaType)
	mediaTitle := lookupMediaTitle(cacheFile, mediaId)
	if mediaTitle == "" {
		mediaTitle = "Unknown"
	}
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
	ExtraTypeOther           ExtraType = "Other"
)

func canonicalizeExtraType(extraType, name string) string {
	if extraType == "" {
		if name != "" {
			extraType = name
		} else {
			extraType = "Video"
		}
	}
	cfg, err := GetCanonicalizeExtraTypeConfig()
	if err == nil {
		if mapped, ok := cfg.Mapping[extraType]; ok {
			return mapped
		}
	}
	return extraType
}

// Placeholder for extras search and download logic
func SearchExtras(mediaType MediaType, id int) ([]Extra, error) {
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

	// Canonicalize ExtraType for each extra before returning
	for i := range extras {
		extras[i].Type = canonicalizeExtraType(extras[i].Type, extras[i].Title)
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
		respondError(c, http.StatusBadRequest, "moviePath required")
		return
	}
	// Scan subfolders for .mkv files and their metadata
	var existing []map[string]interface{}
	entries, err := os.ReadDir(moviePath)
	if CheckErrLog(WARN, "existingExtrasHandler", "ReadDir failed", err) != nil {
		respondJSON(c, http.StatusOK, gin.H{"existing": []map[string]interface{}{}})
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
				if err := ReadJSONFile(metaFile, &meta); err == nil {
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
	respondJSON(c, http.StatusOK, gin.H{"existing": existing})
}

func downloadExtraHandler(c *gin.Context) {
	var req struct {
		MediaType  MediaType `json:"mediaType"`
		MediaId    int       `json:"mediaId"`
		ExtraType  string    `json:"extraType"`
		ExtraTitle string    `json:"extraTitle"`
		YoutubeId  string    `json:"youtubeId"`
	}
	if err := c.BindJSON(&req); CheckErrLog(WARN, "Extras", "[downloadExtraHandler] Invalid request", err) != nil {
		respondError(c, http.StatusBadRequest, ErrInvalidRequest)
		return
	}
	TrailarrLog(INFO, "Extras", "[downloadExtraHandler] Download request: mediaType=%s, mediaId=%d, extraType=%s, extraTitle=%s, youtubeId=%s",
		req.MediaType, req.MediaId, req.ExtraType, req.ExtraTitle, req.YoutubeId)

	// Convert MediaId (int) to string for DownloadYouTubeExtra
	meta, err := DownloadYouTubeExtra(req.MediaType, req.MediaId, req.ExtraType, req.ExtraTitle, req.YoutubeId, true)
	TrailarrLog(INFO, "Extras", "[downloadExtraHandler] DownloadYouTubeExtra returned: meta=%v, err=%v", meta, err)
	if CheckErrLog(WARN, "Extras", "[downloadExtraHandler] Download error", err) != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Lookup media title from cache for history
	recordDownloadHistory(req.MediaType, req.MediaId, req.ExtraType, req.ExtraTitle)
	respondJSON(c, http.StatusOK, gin.H{"status": "downloaded", "meta": meta})
}

func recordDownloadHistory(mediaType MediaType, mediaId int, extraType, extraTitle string) {
	cacheFile, _ := resolveCachePath(mediaType)
	mediaTitle := lookupMediaTitle(cacheFile, mediaId)
	if mediaTitle == "" {
		mediaTitle = "Unknown"
	}
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
		if CheckErrLog(WARN, "walkMediaDirs", "filepath.Walk error", err) != nil || !info.IsDir() {
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
