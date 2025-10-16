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

// RemoveAll429Rejections removes all extras with status 'rejected' and reason containing '429' from the extras collection
func RemoveAll429Rejections() error {
	ctx := context.Background()
	client := GetRedisClient()
	key := ExtrasCollectionKey
	vals, err := client.HVals(ctx, key).Result()
	if err != nil {
		return err
	}
	for _, v := range vals {
		var entry ExtrasEntry
		if err := json.Unmarshal([]byte(v), &entry); err == nil {
			if entry.Status == "rejected" && strings.Contains(entry.Reason, "429") {
				entryKey := fmt.Sprintf("%s:%s:%d", entry.YoutubeId, entry.MediaType, entry.MediaId)
				if err := client.HDel(ctx, key, entryKey).Err(); err != nil {
					TrailarrLog(WARN, "Extras", "Failed to remove 429 rejected extra: %v", err)
				}
				// Also remove from per-media hash
				perMediaKey := fmt.Sprintf("trailarr:extras:%s:%d", entry.MediaType, entry.MediaId)
				_ = client.HDel(ctx, perMediaKey, entryKey).Err()
			}
		}
	}
	return nil
}

// GetExtrasForMedia efficiently returns all extras for a given mediaType and mediaId
func GetExtrasForMedia(ctx context.Context, mediaType MediaType, mediaId int) ([]ExtrasEntry, error) {
	client := GetRedisClient()
	perMediaKey := fmt.Sprintf("trailarr:extras:%s:%d", mediaType, mediaId)
	vals, err := client.HVals(ctx, perMediaKey).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	var result []ExtrasEntry
	for _, v := range vals {
		var entry ExtrasEntry
		if err := json.Unmarshal([]byte(v), &entry); err == nil {
			result = append(result, entry)
		}
	}
	// Fallback: if nothing found, try global (legacy)
	if len(result) == 0 {
		key := ExtrasCollectionKey
		vals, err := client.HVals(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		for _, v := range vals {
			var entry ExtrasEntry
			if err := json.Unmarshal([]byte(v), &entry); err == nil {
				if entry.MediaType == mediaType && entry.MediaId == mediaId {
					result = append(result, entry)
				}
			}
		}
	}
	return result, nil
}

// ListSubdirectories returns all subdirectories for a given path
func ListSubdirectories(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, filepath.Join(path, entry.Name()))
		}
	}
	return dirs, nil
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

// ExtrasEntry is the flat structure for each extra in the new collection
type ExtrasEntry struct {
	MediaType  MediaType `json:"mediaType"`
	MediaId    int       `json:"mediaId"`
	MediaTitle string    `json:"mediaTitle"`
	ExtraTitle string    `json:"extraTitle"`
	ExtraType  string    `json:"extraType"`
	FileName   string    `json:"fileName"`
	YoutubeId  string    `json:"youtubeId"`
	Status     string    `json:"status"`
	Reason     string    `json:"reason,omitempty"`
}

// MarkRejectedExtrasInMemory sets Status="rejected" for extras whose YoutubeId is in rejectedYoutubeIds (in-memory only)
func MarkRejectedExtrasInMemory(extras []Extra, rejectedYoutubeIds map[string]struct{}) {
	for i := range extras {
		if _, exists := rejectedYoutubeIds[extras[i].YoutubeId]; exists {
			extras[i].Status = "rejected"
		}
	}
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
	// Write to global hash
	if err := client.HSet(ctx, key, entryKey, data).Err(); err != nil {
		return err
	}
	// Write to per-media hash for fast lookup
	perMediaKey := fmt.Sprintf("trailarr:extras:%s:%d", entry.MediaType, entry.MediaId)
	if err := client.HSet(ctx, perMediaKey, entryKey, data).Err(); err != nil {
		return err
	}
	return nil
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
	// Build a map for quick lookup of media titles
	movieTitles := make(map[int]string)
	seriesTitles := make(map[int]string)
	// Load movie and series caches once
	movieItems, _ := loadCache(MoviesJSONPath)
	for _, m := range movieItems {
		idInt, ok := parseMediaID(m["id"])
		if ok {
			if t, ok := m["title"].(string); ok {
				movieTitles[idInt] = t
			}
		}
	}
	seriesItems, _ := loadCache(SeriesJSONPath)
	for _, m := range seriesItems {
		idInt, ok := parseMediaID(m["id"])
		if ok {
			if t, ok := m["title"].(string); ok {
				seriesTitles[idInt] = t
			}
		}
	}
	for _, v := range vals {
		var entry ExtrasEntry
		if err := json.Unmarshal([]byte(v), &entry); err == nil {
			// Fill in MediaTitle if missing or empty
			if entry.MediaTitle == "" {
				switch entry.MediaType {
				case MediaTypeMovie:
					if t, ok := movieTitles[entry.MediaId]; ok {
						entry.MediaTitle = t
					}
				case MediaTypeTV:
					if t, ok := seriesTitles[entry.MediaId]; ok {
						entry.MediaTitle = t
					}
				}
			}
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
	err := client.HDel(ctx, key, entryKey).Err()
	return err
}

type Extra struct {
	ID         string
	ExtraType  string
	ExtraTitle string
	YoutubeId  string
	Status     string
	Reason     string
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
				ExtraType:  e.ExtraType,
				ExtraTitle: e.ExtraTitle,
				YoutubeId:  e.YoutubeId,
				Reason:     e.Reason,
			})
		}
	}
	return rejected
}

// SetExtraRejectedPersistent sets the Status of an extra to "rejected" in Redis, adding it if not present (persistent)
func SetExtraRejectedPersistent(mediaType MediaType, mediaId int, extraType, extraTitle, youtubeId, reason string) error {
	TrailarrLog(INFO, "SetExtraRejectedPersistent", "Attempting to mark rejected: mediaType=%s, mediaId=%d, extraType=%s, extraTitle=%s, youtubeId=%s, reason=%s", mediaType, mediaId, extraType, extraTitle, youtubeId, reason)
	ctx := context.Background()
	entry := ExtrasEntry{
		MediaType:  mediaType,
		MediaId:    mediaId,
		ExtraType:  extraType,
		ExtraTitle: extraTitle,
		YoutubeId:  youtubeId,
		Status:     "rejected",
		Reason:     reason,
	}
	return AddOrUpdateExtra(ctx, entry)
}

// UnmarkExtraRejected clears the Status of an extra if it is "rejected" in Redis, but keeps the extra in the array
func UnmarkExtraRejected(mediaType MediaType, mediaId int, extraType, extraTitle, youtubeId string) error {
	ctx := context.Background()
	return RemoveExtra(ctx, youtubeId, mediaType, mediaId)
}

// MarkExtraDownloaded sets the Status of an extra to "downloaded" in Redis, if present
func MarkExtraDownloaded(mediaType MediaType, mediaId int, extraType, extraTitle, youtubeId string) error {
	ctx := context.Background()
	entry := ExtrasEntry{
		MediaType:  mediaType,
		MediaId:    mediaId,
		ExtraType:  extraType,
		ExtraTitle: extraTitle,
		YoutubeId:  youtubeId,
		Status:     "downloaded",
	}
	return AddOrUpdateExtra(ctx, entry)
}

// MarkExtraDeleted sets the Status of an extra to "deleted" in Redis, if present (does not remove)
func MarkExtraDeleted(mediaType MediaType, mediaId int, extraType, extraTitle, youtubeId string) error {
	ctx := context.Background()
	entry := ExtrasEntry{
		MediaType:  mediaType,
		MediaId:    mediaId,
		ExtraType:  extraType,
		ExtraTitle: extraTitle,
		YoutubeId:  youtubeId,
		Status:     "deleted",
	}
	return AddOrUpdateExtra(ctx, entry)
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

	// Find the extra's extraType and extraTitle by YoutubeId from the unified collection
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
			// Use "title" for both movies and series
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
		panic(fmt.Errorf("recordDeleteHistory: could not find media title for mediaType=%v, mediaId=%v", mediaType, mediaId))
	}
	event := HistoryEvent{
		Action:     "delete",
		MediaTitle: mediaTitle,
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

func canonicalizeExtraType(extraType string) string {
	cfg, err := GetCanonicalizeExtraTypeConfig()
	if err == nil {
		if mapped, ok := cfg.Mapping[extraType]; ok {
			return mapped
		}
	}
	return extraType
}

// FetchTMDBExtrasForMedia fetches extras from TMDB for a given media item
func FetchTMDBExtrasForMedia(mediaType MediaType, id int) ([]Extra, error) {
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
		extras[i].ExtraType = canonicalizeExtraType(extras[i].ExtraType)
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
	subdirs, err := ListSubdirectories(moviePath)
	if err != nil {
		respondJSON(c, http.StatusOK, gin.H{"existing": []map[string]interface{}{}})
		return
	}
	// Track duplicate index for each extraType/extraTitle
	dupCount := make(map[string]int)
	for _, subdir := range subdirs {
		dirName := filepath.Base(subdir)
		files, _ := os.ReadDir(subdir)
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".mkv") {
				metaFile := subdir + "/" + strings.TrimSuffix(f.Name(), ".mkv") + ".mkv.json"
				var meta struct {
					ExtraType  string `json:"extraType"`
					ExtraTitle string `json:"extraTitle"`
					FileName   string `json:"fileName"`
					YoutubeId  string `json:"youtubeId"`
					Status     string `json:"status"`
				}
				status := "not-downloaded"
				if err := ReadJSONFile(metaFile, &meta); err == nil {
					status = meta.Status
					if status == "" {
						status = "downloaded"
					}
				}
				key := dirName + "|" + meta.ExtraTitle
				dupCount[key]++
				existing = append(existing, map[string]interface{}{
					"type":       dirName,
					"extraType":  meta.ExtraType,
					"extraTitle": meta.ExtraTitle,
					"fileName":   meta.FileName,
					"YoutubeId":  meta.YoutubeId,
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

	// Write .mkv.json meta file for manual download
	cacheFile, _ := resolveCachePath(req.MediaType)
	mediaPath, err := FindMediaPathByID(cacheFile, req.MediaId)
	if err == nil && mediaPath != "" {
		extraDir := mediaPath + "/" + req.ExtraType
		if err := os.MkdirAll(extraDir, 0775); err == nil {
			metaFile := extraDir + "/" + SanitizeFilename(req.ExtraTitle) + ".mkv.json"
			meta := struct {
				ExtraType  string `json:"extraType"`
				ExtraTitle string `json:"extraTitle"`
				FileName   string `json:"fileName"`
				YoutubeId  string `json:"youtubeId"`
				Status     string `json:"status"`
			}{
				ExtraType:  req.ExtraType,
				ExtraTitle: req.ExtraTitle,
				FileName:   SanitizeFilename(req.ExtraTitle) + ".mkv",
				YoutubeId:  req.YoutubeId,
				Status:     "queued",
			}
			if f, err := os.Create(metaFile); err == nil {
				enc := json.NewEncoder(f)
				enc.SetIndent("", "  ")
				_ = enc.Encode(meta)
				f.Close()
			}
		}
	}
	respondJSON(c, http.StatusOK, gin.H{"status": "queued"})
}

// shouldDownloadExtra determines if an extra should be downloaded
func shouldDownloadExtra(extra Extra, config ExtraTypesConfig) bool {
	if extra.Status != "missing" || extra.YoutubeId == "" {
		return false
	}
	if extra.Status == "rejected" {
		return false
	}
	typeName := extra.ExtraType
	canonical := canonicalizeExtraType(typeName)
	return isExtraTypeEnabled(config, canonical)
}

// handleExtraDownload downloads an extra unless it's rejected
func handleExtraDownload(mediaType MediaType, mediaId int, extra Extra) error {
	if extra.Status == "rejected" {
		TrailarrLog(INFO, "DownloadMissingExtras", "Skipping rejected extra: mediaType=%v, mediaId=%v, extraType=%s, extraTitle=%s, youtubeId=%s", mediaType, mediaId, extra.ExtraType, extra.ExtraTitle, extra.YoutubeId)
		return nil
	}
	// Enqueue the extra for download using the queue system
	item := DownloadQueueItem{
		MediaType:  mediaType,
		MediaId:    mediaId,
		ExtraType:  extra.ExtraType,
		ExtraTitle: extra.ExtraTitle,
		YouTubeID:  extra.YoutubeId,
		QueuedAt:   time.Now(),
	}
	AddToDownloadQueue(item, "task")
	TrailarrLog(INFO, "QUEUE", "[handleExtraDownload] Enqueued extra: mediaType=%v, mediaId=%v, extraType=%s, extraTitle=%s, youtubeId=%s", mediaType, mediaId, extra.ExtraType, extra.ExtraTitle, extra.YoutubeId)
	return nil
}

// Scans a media path and returns a map of existing extras (type|title)
func ScanExistingExtras(mediaPath string) map[string]bool {
	existing := map[string]bool{}
	if mediaPath == "" {
		return existing
	}
	subdirs, err := ListSubdirectories(mediaPath)
	if err != nil {
		return existing
	}
	for _, subdir := range subdirs {
		dirName := filepath.Base(subdir)
		files, _ := os.ReadDir(subdir)
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".mkv") {
				title := strings.TrimSuffix(f.Name(), ".mkv")
				key := dirName + "|" + title
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
		typeStr := canonicalizeExtraType(extras[i].ExtraType)
		extras[i].ExtraType = typeStr
		title := SanitizeFilename(extras[i].ExtraTitle)
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
		_, err := FetchTMDBExtrasForMedia(mediaType, idInt)
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
		extras, _ := FetchTMDBExtrasForMedia(mediaType, idInt)
		mediaPath, _ := FindMediaPathByID(cacheFile, idInt)
		MarkDownloadedExtras(extras, mediaPath, "type", "title")
		// Defensive: mark rejected extras before any download
		rejectedExtras := GetRejectedExtrasForMedia(mediaType, idInt)
		rejectedYoutubeIds := make(map[string]struct{})
		for _, r := range rejectedExtras {
			rejectedYoutubeIds[r.YoutubeId] = struct{}{}
		}
		MarkRejectedExtrasInMemory(extras, rejectedYoutubeIds)
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
	MarkRejectedExtrasInMemory(extras, rejectedYoutubeIds)
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
	subdirs, err := ListSubdirectories(mediaPath)
	if err != nil {
		return extrasInfo
	}
	for _, subdir := range subdirs {
		extraType := filepath.Base(subdir)
		files, _ := os.ReadDir(subdir)
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") || !strings.HasSuffix(f.Name(), ".mkv.json") {
				continue
			}
			filePath := filepath.Join(subdir, f.Name())
			var meta map[string]interface{}
			if err := ReadJSONFile(filePath, &meta); err == nil {
				// Standardize keys
				canonical := make(map[string]interface{})
				for k, v := range meta {
					switch strings.ToLower(k) {
					case "title", "extratitle":
						canonical["Title"] = v
					case "filename", "fileName":
						canonical["FileName"] = v
					case "youtubeid":
						canonical["YoutubeId"] = v
					case "status":
						canonical["Status"] = v
					default:
						canonical[k] = v
					}
				}
				extrasInfo[extraType] = append(extrasInfo[extraType], canonical)
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
	for _, extra := range extras {
		if extra.Status == "rejected" {
			rejected = append(rejected, extra)
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
	err := RemoveExtra(ctx, req.YoutubeId, mt, req.MediaId)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "Could not remove extra from collection: "+err.Error())
		return
	}
	respondJSON(c, http.StatusOK, gin.H{"status": "removed"})
}
