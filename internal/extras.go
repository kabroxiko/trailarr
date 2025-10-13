package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Sanitize filename for OS conflicts (remove/replace invalid chars)
func SanitizeFilename(name string) string {
	// Remove any character not allowed in filenames
	// Windows: \/:*?"<>|, Linux: /
	re := regexp.MustCompile(`[\\/:*?"<>|]`)
	name = re.ReplaceAllString(name, "_")
	name = strings.TrimSpace(name)
	return name
}

// ExtrasEntry is the flat structure for each extra in the new collection
type ExtrasEntry struct {
	MediaType  MediaType `json:"mediaType"`
	MediaId    int       `json:"mediaId"`
	ExtraTitle string    `json:"extraTitle"`
	ExtraType  string    `json:"extraType"`
	FileName   string    `json:"fileName"`
	YoutubeId  string    `json:"youtubeId"`
	Status     string    `json:"status"`
	Reason     string    `json:"reason,omitempty"`
}

// AddOrUpdateExtra stores or updates an extra in the unified collection
func AddOrUpdateExtra(ctx context.Context, entry ExtrasEntry) error {
	client := GetRedisClient()
	key := ExtrasCollectionKey
	// Use YoutubeId+MediaType+MediaId as unique identifier
	entryKey := fmt.Sprintf("%s:%s:%d", entry.YoutubeId, entry.MediaType, entry.MediaId)
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return client.HSet(ctx, key, entryKey, data).Err()
}

// GetExtraByYoutubeId fetches an extra by YoutubeId, MediaType, and MediaId
func GetExtraByYoutubeId(ctx context.Context, youtubeId string, mediaType MediaType, mediaId int) (*ExtrasEntry, error) {
	client := GetRedisClient()
	key := ExtrasCollectionKey
	entryKey := fmt.Sprintf("%s:%s:%d", youtubeId, mediaType, mediaId)
	val, err := client.HGet(ctx, key, entryKey).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var entry ExtrasEntry
	if err := json.Unmarshal([]byte(val), &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

// GetAllExtras returns all extras in the collection
func GetAllExtras(ctx context.Context) ([]ExtrasEntry, error) {
	client := GetRedisClient()
	key := ExtrasCollectionKey
	vals, err := client.HVals(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	var result []ExtrasEntry
	for _, v := range vals {
		var entry ExtrasEntry
		if err := json.Unmarshal([]byte(v), &entry); err == nil {
			result = append(result, entry)
		}
	}
	return result, nil
}

// RemoveExtra removes an extra from the collection
func RemoveExtra(ctx context.Context, youtubeId string, mediaType MediaType, mediaId int) error {
	client := GetRedisClient()
	key := ExtrasCollectionKey
	entryKey := fmt.Sprintf("%s:%s:%d", youtubeId, mediaType, mediaId)
	return client.HDel(ctx, key, entryKey).Err()
}

type Extra struct {
	ID        string
	Type      string
	Title     string
	YoutubeId string
	Status    string
}

// GetRejectedExtrasForMedia returns rejected extras for a given media type and id, using Redis cache
func GetRejectedExtrasForMedia(mediaType MediaType, id int) []RejectedExtra {
	ctx := context.Background()
	extras, err := GetAllExtras(ctx)
	if err != nil {
		return nil
	}
	var rejected []RejectedExtra
	for _, e := range extras {
		if e.MediaType == mediaType && e.MediaId == id && e.Status == "rejected" {
			rejected = append(rejected, RejectedExtra{
				MediaType:  e.MediaType,
				MediaId:    e.MediaId,
				MediaTitle: "", // MediaTitle not stored in unified collection
				ExtraType:  e.ExtraType,
				ExtraTitle: e.ExtraTitle,
				YoutubeId:  e.YoutubeId,
				Reason:     e.Reason,
			})
		}
	}
	return rejected
}

// MarkExtraRejected sets the Status of an extra to "rejected" in Redis, adding it if not present
func MarkExtraRejected(mediaType MediaType, mediaId int, extraType, extraTitle, youtubeId, reason string) error {
	TrailarrLog(INFO, "MarkExtraRejected", "Attempting to mark rejected: mediaType=%s, mediaId=%d, extraType=%s, extraTitle=%s, youtubeId=%s, reason=%s", mediaType, mediaId, extraType, extraTitle, youtubeId, reason)
	ctx := context.Background()
	// Try to fetch the extra from the unified collection
	entry, _ := GetExtraByYoutubeId(ctx, youtubeId, mediaType, mediaId)
	if entry != nil {
		TrailarrLog(INFO, "MarkExtraRejected", "Found existing extra, updating status to rejected: %+v", entry)
		entry.Status = "rejected"
		entry.Reason = reason
		err := AddOrUpdateExtra(ctx, *entry)
		if err != nil {
			TrailarrLog(WARN, "MarkExtraRejected", "Failed to update extra: %v", err)
		}
		return err
	}
	TrailarrLog(INFO, "MarkExtraRejected", "No existing extra found, creating new rejected entry.")
	newEntry := ExtrasEntry{
		MediaType:  mediaType,
		MediaId:    mediaId,
		ExtraTitle: extraTitle,
		ExtraType:  extraType,
		FileName:   "",
		YoutubeId:  youtubeId,
		Status:     "rejected",
		Reason:     reason,
	}
	err := AddOrUpdateExtra(ctx, newEntry)
	if err != nil {
		TrailarrLog(WARN, "MarkExtraRejected", "Failed to add new rejected extra: %v", err)
	}
	return err
}

// UnmarkExtraRejected clears the Status of an extra if it is "rejected" in Redis, but keeps the extra in the array
func UnmarkExtraRejected(mediaType MediaType, mediaId int, extraType, extraTitle, youtubeId string) error {
	cacheFile, _ := resolveCachePath(mediaType)
	items, err := loadCache(cacheFile)
	if err != nil {
		return err
	}
	updated := false
	for _, m := range items {
		idInt, ok := parseMediaID(m["id"])
		if !ok || idInt != mediaId {
			continue
		}
		extras, ok := m["extras"].([]interface{})
		if !ok {
			continue
		}
		for _, e := range extras {
			em, ok := e.(map[string]interface{})
			if !ok {
				continue
			}
			if toString(em["Type"]) == extraType && toString(em["Title"]) == extraTitle && toString(em["YoutubeId"]) == youtubeId {
				if em["Status"] == "rejected" {
					em["Status"] = ""
					em["reason"] = ""
					updated = true
				}
			}
		}
	}
	if updated {
		return SaveMediaToRedis(cacheFile, items)
	}
	return nil
}

// MarkExtraDownloaded sets the Status of an extra to "downloaded" in Redis, adding it if not present
func MarkExtraDownloaded(mediaType MediaType, mediaId int, extraType, extraTitle, youtubeId string) error {
	cacheFile, _ := resolveCachePath(mediaType)
	items, err := loadCache(cacheFile)
	if err != nil {
		return err
	}
	updated := false
	for _, m := range items {
		idInt, ok := parseMediaID(m["id"])
		if !ok || idInt != mediaId {
			continue
		}
		extras, ok := m["extras"].([]interface{})
		if !ok {
			continue
		}
		found := false
		for _, e := range extras {
			em, ok := e.(map[string]interface{})
			if !ok {
				continue
			}
			if toString(em["Type"]) == extraType && toString(em["Title"]) == extraTitle && toString(em["YoutubeId"]) == youtubeId {
				em["Status"] = "downloaded"
				updated = true
				found = true
			}
		}
		if !found {
			newExtra := map[string]interface{}{
				"Type":      extraType,
				"Title":     extraTitle,
				"YoutubeId": youtubeId,
				"Status":    "downloaded",
			}
			m["extras"] = append(extras, newExtra)
			updated = true
		}
	}
	if updated {
		return SaveMediaToRedis(cacheFile, items)
	}
	return nil
}

// MarkExtraDeleted sets the Status of an extra to "deleted" in Redis, adding it if not present
// MarkExtraDeleted removes the extra from the 'extras' array in Redis for the corresponding movie/series
func MarkExtraDeleted(mediaType MediaType, mediaId int, extraType, extraTitle, youtubeId string) error {
	cacheFile, _ := resolveCachePath(mediaType)
	items, err := loadCache(cacheFile)
	if err != nil {
		return err
	}
	updated := false
	for _, m := range items {
		idInt, ok := parseMediaID(m["id"])
		if !ok || idInt != mediaId {
			continue
		}
		extras, ok := m["extras"].([]interface{})
		if !ok {
			continue
		}
		newExtras := make([]interface{}, 0, len(extras))
		for _, e := range extras {
			em, ok := e.(map[string]interface{})
			if !ok {
				newExtras = append(newExtras, e)
				continue
			}
			if youtubeId != "" && toString(em["YoutubeId"]) == youtubeId {
				updated = true
				continue // skip (delete)
			}
			newExtras = append(newExtras, em)
		}
		m["extras"] = newExtras
	}
	if updated {
		return SaveMediaToRedis(cacheFile, items)
	}
	return nil
}

// Helper to safely convert interface{} to string
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return fmt.Sprintf("%v", t)
	}
}

// Handler to delete an extra and record history
func deleteExtraHandler(c *gin.Context) {
	var req struct {
		MediaType MediaType `json:"mediaType"`
		MediaId   int       `json:"mediaId"`
		YoutubeId string    `json:"youtubeId"`
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

	// Find the extra's type and title by YoutubeId from the unified collection
	ctx := context.Background()
	entry, err := GetExtraByYoutubeId(ctx, req.YoutubeId, req.MediaType, req.MediaId)
	if err != nil || entry == nil {
		respondError(c, http.StatusNotFound, "Extra not found in collection")
		return
	}
	// Try to delete files, but do not fail if missing
	_ = deleteExtraFiles(mediaPath, entry.ExtraType, entry.ExtraTitle)

	// Remove from the unified collection in Redis
	if err := RemoveExtra(ctx, req.YoutubeId, req.MediaType, req.MediaId); err != nil {
		TrailarrLog(WARN, "Extras", "Failed to remove extra from Redis: %v", err)
	}

	recordDeleteHistory(req.MediaType, req.MediaId, entry.ExtraType, entry.ExtraTitle)
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
	if err != nil {
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
		MediaId:    mediaId,
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
	if err != nil {
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
	if err := c.BindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, ErrInvalidRequest)
		return
	}
	TrailarrLog(INFO, "Extras", "[downloadExtraHandler] Download request: mediaType=%s, mediaId=%d, extraType=%s, extraTitle=%s, youtubeId=%s",
		req.MediaType, req.MediaId, req.ExtraType, req.ExtraTitle, req.YoutubeId)

	// Enqueue the download request
	item := DownloadQueueItem{
		MediaType:  req.MediaType,
		MediaId:    req.MediaId,
		ExtraType:  req.ExtraType,
		ExtraTitle: req.ExtraTitle,
		YouTubeID:  req.YoutubeId,
		QueuedAt:   time.Now(),
	}
	AddToDownloadQueue(item, "api")
	TrailarrLog(INFO, "Extras", "[downloadExtraHandler] Enqueued download: mediaType=%s, mediaId=%d, extraType=%s, extraTitle=%s, youtubeId=%s", req.MediaType, req.MediaId, req.ExtraType, req.ExtraTitle, req.YoutubeId)
	respondJSON(c, http.StatusOK, gin.H{"status": "queued"})
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
	// Enqueue the extra for download using the queue system
	item := DownloadQueueItem{
		MediaType:  mediaType,
		MediaId:    mediaId,
		ExtraType:  extra.Type,
		ExtraTitle: extra.Title,
		YouTubeID:  extra.YoutubeId,
		QueuedAt:   time.Now(),
	}
	AddToDownloadQueue(item, "task")
	TrailarrLog(INFO, "QUEUE", "[handleExtraDownload] Enqueued extra: mediaType=%v, mediaId=%v, type=%s, title=%s, youtubeId=%s", mediaType, mediaId, extra.Type, extra.Title, extra.YoutubeId)
	return nil
}

// Scans a media path and returns a map of existing extras (type|title)
func ScanExistingExtras(mediaPath string) map[string]bool {
	existing := map[string]bool{}
	if mediaPath == "" {
		return existing
	}
	entries, err := os.ReadDir(mediaPath)
	if err != nil {
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
	if err != nil {
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
		if err != nil {
			TrailarrLog(WARN, "Extras", "Failed to download extra: %v", err)
		}
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
// BlacklistExtrasHandler aggregates all rejected extras from Redis for both movies and series
func BlacklistExtrasHandler(c *gin.Context) {
	ctx := context.Background()
	extras, err := GetAllExtras(ctx)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Failed to fetch extras: "+err.Error())
		return
	}
	var rejected []ExtrasEntry
	for _, e := range extras {
		if e.Status == "rejected" {
			rejected = append(rejected, e)
		}
	}
	if rejected == nil {
		rejected = make([]ExtrasEntry, 0)
	}
	respondJSON(c, http.StatusOK, rejected)
}

// Handler to remove an entry from the rejected extras blacklist
func RemoveBlacklistExtraHandler(c *gin.Context) {
	var req struct {
		MediaType  string `json:"mediaType"`
		MediaId    int    `json:"mediaId"`
		ExtraType  string `json:"extraType"`
		ExtraTitle string `json:"extraTitle"`
		YoutubeId  string `json:"youtubeId"`
	}
	if err := c.BindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, ErrInvalidRequest)
		return
	}
	var mt MediaType
	switch req.MediaType {
	case string(MediaTypeMovie):
		mt = MediaTypeMovie
	case string(MediaTypeTV):
		mt = MediaTypeTV
	default:
		respondError(c, http.StatusBadRequest, "Invalid mediaType")
		return
	}
	ctx := context.Background()
	// Remove from hash (legacy, for compatibility)
	err := RemoveExtra(ctx, req.YoutubeId, mt, req.MediaId)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Could not remove extra from collection: "+err.Error())
		return
	}
	// Remove from Redis list (current rejected extras storage)
	client := GetRedisClient()
	items, lerr := client.LRange(ctx, ExtrasCollectionKey, 0, -1).Result()
	if lerr == nil {
		for _, itemStr := range items {
			var r RejectedExtra
			if err := json.Unmarshal([]byte(itemStr), &r); err == nil {
				if r.YoutubeId == req.YoutubeId && r.MediaType == mt && r.MediaId == req.MediaId && r.ExtraType == req.ExtraType && r.ExtraTitle == req.ExtraTitle {
					// Remove this item from the list
					_ = client.LRem(ctx, ExtrasCollectionKey, 1, itemStr).Err()
				}
			}
		}
	}
	respondJSON(c, http.StatusOK, gin.H{"status": "removed"})
}

// removeRejectedExtrasWithReasons removes all rejected extras with reasons matching certain substrings from the rejected_extras.json file.
func removeRejectedExtrasWithReasons() {
	TrailarrLog(INFO, "Tasks", "[removeRejectedExtrasWithReasons] Entered. Redis Key: %s", ExtrasCollectionKey)
	// TODO: Read from Redis ExtrasCollectionKey instead of file
	// TODO: Filter and write back to Redis ExtrasCollectionKey
}
