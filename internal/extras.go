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
		return MoviesJSONPath, nil
	case MediaTypeTV:
		return SeriesJSONPath, nil
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

// shouldDownloadExtra determines if an extra should be downloaded
func shouldDownloadExtra(extra Extra, config ExtraTypesConfig) bool {
	if extra.Status != "missing" || extra.YoutubeId == "" {
		return false
	}
	if extra.Status == "rejected" {
		return false
	}
	typeName := extra.Type
	canonical := canonicalizeExtraType(typeName, "")
	return isExtraTypeEnabled(config, canonical)
}

// handleExtraDownload downloads an extra unless it's rejected
func handleExtraDownload(mediaType MediaType, mediaId int, extra Extra) error {
	if extra.Status == "rejected" {
		TrailarrLog(INFO, "DownloadMissingExtras", "Skipping rejected extra: mediaType=%v, mediaId=%v, type=%s, title=%s, youtubeId=%s", mediaType, mediaId, extra.Type, extra.Title, extra.YoutubeId)
		return nil
	}
	_, err := DownloadYouTubeExtra(mediaType, mediaId, extra.Type, extra.Title, extra.YoutubeId)
	return err
}

// Scans a media path and returns a map of existing extras (type|title)
func ScanExistingExtras(mediaPath string) map[string]bool {
	existing := map[string]bool{}
	if mediaPath == "" {
		return existing
	}
	entries, err := os.ReadDir(mediaPath)
	if CheckErrLog(WARN, "ScanExistingExtras", "ReadDir failed", err) != nil {
		return existing
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subdir := mediaPath + "/" + entry.Name()
		files, _ := os.ReadDir(subdir)
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".mkv") {
				title := strings.TrimSuffix(f.Name(), ".mkv")
				key := entry.Name() + "|" + title
				existing[key] = true
			}
		}
	}
	return existing
}

// Checks which extras are downloaded in the given media path and marks them in the extras list
// extras: slice of Extra (from TMDB), mediaPath: path to the movie/series folder
// typeKey: the key in the extra map for the type (usually "type"), titleKey: the key for the title (usually "title")
func MarkDownloadedExtras(extras []Extra, mediaPath string, typeKey, titleKey string) {
	existing := ScanExistingExtras(mediaPath)
	for i := range extras {
		typeStr := canonicalizeExtraType(extras[i].Type, extras[i].Title)
		extras[i].Type = typeStr
		title := SanitizeFilename(extras[i].Title)
		key := typeStr + "|" + title
		extras[i].Status = "missing"
		if existing[key] {
			extras[i].Status = "downloaded"
		}
	}
}

// DownloadMissingExtras downloads missing extras for a given media type ("movie" or "tv")
func DownloadMissingExtras(mediaType MediaType, cacheFile string) error {
	TrailarrLog(INFO, "DownloadMissingExtras", "DownloadMissingExtras: mediaType=%s, cacheFile=%s", mediaType, cacheFile)

	items, err := loadCache(cacheFile)
	if CheckErrLog(WARN, "DownloadMissingExtras", "Failed to load cache", err) != nil {
		TrailarrLog(ERROR, "QUEUE", "[EXTRAS] Failed to load cache for %s: %v", mediaType, err)
		return err
	}
	type downloadItem struct {
		idInt     int
		mediaPath string
		extras    []Extra
	}
	config, _ := GetExtraTypesConfig()
	filtered := Filter(items, func(m map[string]interface{}) bool {
		idInt, ok := parseMediaID(m["id"])
		if !ok {
			TrailarrLog(WARN, "DownloadMissingExtras", "Missing or invalid id in item: %v", m)
			return false
		}
		_, err := SearchExtras(mediaType, idInt)
		if err != nil {
			TrailarrLog(WARN, "DownloadMissingExtras", "SearchExtras error: %v", err)
			return false
		}
		mediaPath, err := FindMediaPathByID(cacheFile, idInt)
		if err != nil || mediaPath == "" {
			TrailarrLog(WARN, "DownloadMissingExtras", "FindMediaPathByID error or empty: %v, mediaPath=%s", err, mediaPath)
			return false
		}
		return true
	})
	mapped := Map(filtered, func(media map[string]interface{}) downloadItem {
		idInt, _ := parseMediaID(media["id"])
		extras, _ := SearchExtras(mediaType, idInt)
		mediaPath, _ := FindMediaPathByID(cacheFile, idInt)
		MarkDownloadedExtras(extras, mediaPath, "type", "title")
		// Defensive: mark rejected extras before any download
		rejectedExtras := GetRejectedExtrasForMedia(mediaType, idInt)
		rejectedYoutubeIds := make(map[string]struct{})
		for _, r := range rejectedExtras {
			rejectedYoutubeIds[r.YoutubeId] = struct{}{}
		}
		for i := range extras {
			if _, exists := rejectedYoutubeIds[extras[i].YoutubeId]; exists {
				extras[i].Status = "rejected"
			}
		}
		return downloadItem{idInt, mediaPath, extras}
	})
	for _, di := range mapped {
		filterAndDownloadExtras(mediaType, di.idInt, di.extras, config)
	}
	return nil
}

// filterAndDownloadExtras filters extras and downloads them if enabled
func filterAndDownloadExtras(mediaType MediaType, mediaId int, extras []Extra, config ExtraTypesConfig) {
	// Mark extras as rejected if their YouTube ID matches any in rejected_extras.json
	rejectedExtras := GetRejectedExtrasForMedia(mediaType, mediaId)
	rejectedYoutubeIds := make(map[string]struct{})
	for _, r := range rejectedExtras {
		rejectedYoutubeIds[r.YoutubeId] = struct{}{}
	}
	for i := range extras {
		if _, exists := rejectedYoutubeIds[extras[i].YoutubeId]; exists {
			extras[i].Status = "rejected"
		}
	}
	filtered := Filter(extras, func(extra Extra) bool {
		return shouldDownloadExtra(extra, config)
	})
	for _, extra := range filtered {
		err := handleExtraDownload(mediaType, mediaId, extra)
		CheckErrLog(WARN, "DownloadMissingExtras", "Failed to download", err)
	}
}

// Helper: Scan extras directories and collect info for a media path
func scanExtrasInfo(mediaPath string) map[string][]map[string]interface{} {
	extrasInfo := make(map[string][]map[string]interface{})
	if mediaPath == "" {
		return extrasInfo
	}
	entries, err := os.ReadDir(mediaPath)
	if err != nil {
		return extrasInfo
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		extraType := entry.Name()
		subdir := filepath.Join(mediaPath, extraType)
		files, _ := os.ReadDir(subdir)
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") || !strings.HasSuffix(f.Name(), ".mkv.json") {
				continue
			}
			filePath := filepath.Join(subdir, f.Name())
			var meta map[string]interface{}
			if err := ReadJSONFile(filePath, &meta); err == nil {
				extrasInfo[extraType] = append(extrasInfo[extraType], meta)
			}
		}
	}
	return extrasInfo
}

// Handler for serving the rejected extras blacklist
func BlacklistExtrasHandler(c *gin.Context) {
	file, err := os.Open(RejectedExtrasPath)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Could not open rejected_extras.json: "+err.Error())
		return
	}
	defer file.Close()
	var data interface{}
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		respondError(c, http.StatusInternalServerError, "Could not decode rejected_extras.json: "+err.Error())
		return
	}
	respondJSON(c, http.StatusOK, data)
}

// Handler to remove an entry from the rejected extras blacklist
func RemoveBlacklistExtraHandler(c *gin.Context) {
	var req struct {
		MediaType  string      `json:"mediaType"`
		MediaId    interface{} `json:"mediaId"`
		ExtraType  string      `json:"extraType"`
		ExtraTitle string      `json:"extraTitle"`
		YoutubeId  string      `json:"youtubeId"`
	}
	if err := c.BindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, ErrInvalidRequest)
		return
	}
	// Read current blacklist
	var blacklist []map[string]interface{}
	f, err := os.Open(RejectedExtrasPath)
	if err == nil {
		_ = json.NewDecoder(f).Decode(&blacklist)
		f.Close()
	}
	// Remove matching entry
	newList := make([]map[string]interface{}, 0, len(blacklist))
	for _, entry := range blacklist {
		match := true
		if req.MediaType != "" && entry["media_type"] != req.MediaType && entry["mediaType"] != req.MediaType {
			match = false
		}
		if req.MediaId != nil && entry["media_id"] != req.MediaId && entry["mediaId"] != req.MediaId {
			match = false
		}
		if req.ExtraType != "" && entry["extra_type"] != req.ExtraType && entry["extraType"] != req.ExtraType {
			match = false
		}
		if req.ExtraTitle != "" && entry["extra_title"] != req.ExtraTitle && entry["extraTitle"] != req.ExtraTitle {
			match = false
		}
		if req.YoutubeId != "" && entry["youtube_id"] != req.YoutubeId && entry["youtubeId"] != req.YoutubeId {
			match = false
		}
		if !match {
			newList = append(newList, entry)
		}
	}
	// Write updated blacklist
	f2, err := os.Create(RejectedExtrasPath)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Could not update rejected_extras.json: "+err.Error())
		return
	}
	defer f2.Close()
	if err := json.NewEncoder(f2).Encode(newList); err != nil {
		respondError(c, http.StatusInternalServerError, "Could not encode rejected_extras.json: "+err.Error())
		return
	}
	respondJSON(c, http.StatusOK, gin.H{"status": "removed"})
}
