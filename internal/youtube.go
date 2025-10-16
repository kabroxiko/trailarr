package internal

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// YouTube trailer search SSE handler (progressive results)
func YouTubeTrailerSearchStreamHandler(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()

	mediaType := c.Query("mediaType")
	mediaIdStr := c.Query("mediaId")
	if mediaType == "" || mediaIdStr == "" {
		TrailarrLog(WARN, "YouTube", "Missing mediaType or mediaId in query params (SSE)")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing mediaType or mediaId"})
		return
	}
	var mediaId int
	_, err := fmt.Sscanf(mediaIdStr, "%d", &mediaId)
	if err != nil || mediaId == 0 {
		TrailarrLog(WARN, "YouTube", "Invalid mediaId in query params (SSE): %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid mediaId"})
		return
	}
	TrailarrLog(INFO, "YouTube", "YouTubeTrailerSearchStreamHandler GET: mediaType=%s, mediaId=%d", mediaType, mediaId)

	// Lookup media title/originalTitle
	var title, originalTitle string
	cacheFile, _ := resolveCachePath(MediaType(mediaType))
	items, err := loadCache(cacheFile)
	if err != nil {
		TrailarrLog(ERROR, "YouTube", "Failed to load cache for mediaType=%s: %v", mediaType, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Media cache not found"})
		return
	}
	for _, m := range items {
		idInt, ok := parseMediaID(m["id"])
		if ok && idInt == mediaId {
			if t, ok := m["title"].(string); ok {
				title = t
			}
			if ot, ok := m["originalTitle"].(string); ok {
				originalTitle = ot
			}
			break
		}
	}
	if title == "" && originalTitle == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
		return
	}

	var searchTerms []string
	if originalTitle != "" {
		searchTerms = append(searchTerms, originalTitle)
	}
	if title != "" && originalTitle != title {
		searchTerms = append(searchTerms, title)
	}
	videoIdSet := make(map[string]bool)
	count := 0
	for _, term := range searchTerms {
		searchQuery := term + " trailer"
		ytDlpArgs := []string{"-j", "ytsearch10:" + searchQuery, "--skip-download"}
		TrailarrLog(INFO, "YouTube", "yt-dlp command (SSE): yt-dlp %v", ytDlpArgs)
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		cmd := exec.CommandContext(ctx, "yt-dlp", ytDlpArgs...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			TrailarrLog(ERROR, "YouTube", "Failed to get StdoutPipe: %v", err)
			cancel()
			continue
		}
		if err := cmd.Start(); err != nil {
			TrailarrLog(ERROR, "YouTube", "Failed to start yt-dlp: %v", err)
			cancel()
			continue
		}
		TrailarrLog(INFO, "YouTube", "Started yt-dlp process (SSE streaming)...")
		reader := bufio.NewReader(stdout)
		for {
			line, err := reader.ReadBytes('\n')
			if len(line) == 0 && err != nil {
				break
			}
			var item struct {
				ID          string `json:"id"`
				Title       string `json:"title"`
				Description string `json:"description"`
				Thumbnail   string `json:"thumbnail"`
				Channel     string `json:"channel"`
				ChannelID   string `json:"channel_id"`
			}
			parseErr := json.Unmarshal(bytes.TrimSpace(line), &item)
			if parseErr == nil {
				if item.ID != "" && !videoIdSet[item.ID] {
					videoIdSet[item.ID] = true
					result := gin.H{
						"id": gin.H{"videoId": item.ID},
						"snippet": gin.H{
							"title":       item.Title,
							"description": item.Description,
							"thumbnails": gin.H{
								"default": gin.H{"url": item.Thumbnail},
							},
							"channelTitle": item.Channel,
							"channelId":    item.ChannelID,
						},
					}
					b, _ := json.Marshal(result)
					fmt.Fprintf(c.Writer, "data: %s\n\n", b)
					c.Writer.Flush()
					count++
					if count >= 10 {
						break
					}
				}
			}
			if err != nil {
				if err != io.EOF {
					TrailarrLog(ERROR, "YouTube", "[SSE] Reader error: %v", err)
				}
				break
			}
		}
		if count >= 10 {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
		cancel()
		if ctx.Err() == context.DeadlineExceeded {
			TrailarrLog(ERROR, "YouTube", "[SSE] yt-dlp search timed out for query: %s", searchQuery)
			continue
		}
		if count >= 10 {
			break
		}
	}
	// Optionally send a done event
	fmt.Fprintf(c.Writer, "event: done\ndata: {}\n\n")
	c.Writer.Flush()
}

type YtdlpFlagsConfig struct {
	Quiet              bool    `yaml:"quiet" json:"quiet"`
	NoProgress         bool    `yaml:"noprogress" json:"noprogress"`
	WriteSubs          bool    `yaml:"writesubs" json:"writesubs"`
	WriteAutoSubs      bool    `yaml:"writeautosubs" json:"writeautosubs"`
	EmbedSubs          bool    `yaml:"embedsubs" json:"embedsubs"`
	RemuxVideo         string  `yaml:"remuxvideo" json:"remuxvideo"`
	SubFormat          string  `yaml:"subformat" json:"subformat"`
	SubLangs           string  `yaml:"sublangs" json:"sublangs"`
	RequestedFormats   string  `yaml:"requestedformats" json:"requestedformats"`
	Timeout            float64 `yaml:"timeout" json:"timeout"`
	SleepInterval      float64 `yaml:"sleepInterval" json:"sleepInterval"`
	MaxDownloads       int     `yaml:"maxDownloads" json:"maxDownloads"`
	LimitRate          string  `yaml:"limitRate" json:"limitRate"`
	SleepRequests      float64 `yaml:"sleepRequests" json:"sleepRequests"`
	MaxSleepInterval   float64 `yaml:"maxSleepInterval" json:"maxSleepInterval"`
	CookiesFromBrowser string  `yaml:"cookiesFromBrowser" json:"cookiesFromBrowser"`
}

// DownloadQueueItem represents a single download request
type DownloadQueueItem struct {
	MediaType  MediaType `json:"mediaType"`
	MediaId    int       `json:"mediaId"`
	MediaTitle string    `json:"mediaTitle"`
	ExtraType  string    `json:"extraType"`
	ExtraTitle string    `json:"extraTitle"`
	YouTubeID  string    `json:"youtubeId"`
	QueuedAt   time.Time `json:"queuedAt"`
	Status     string    `json:"status"` // "queued", "downloading", etc.
	Reason     string    `json:"reason,omitempty"`
}

// DownloadStatus holds the status of a download
type DownloadStatus struct {
	Status    string // e.g. "queued", "downloading", "downloaded", "failed", "exists", "rejected"
	UpdatedAt time.Time
	Error     string
}

var downloadStatusMap = make(map[string]*DownloadStatus) // keyed by YouTubeID
var queueMutex sync.Mutex

// BatchStatusRequest is the request body for batch status queries
type BatchStatusRequest struct {
	YoutubeIds []string `json:"youtubeIds"`
}

// BatchStatusResponse is the response body for batch status queries
type BatchStatusResponse struct {
	Statuses map[string]*DownloadStatus `json:"statuses"`
}

// GetBatchDownloadStatusHandler returns the status for multiple YouTube IDs
func GetBatchDownloadStatusHandler(c *gin.Context) {
	var req BatchStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.YoutubeIds) == 0 {
		TrailarrLog(WARN, "BATCH", "/api/extras/status/batch invalid request: %v, body: %v", err, c.Request.Body)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	TrailarrLog(INFO, "BATCH", "/api/extras/status/batch request: %+v", req)
	statuses := make(map[string]*DownloadStatus, len(req.YoutubeIds))
	// Load persistent queue from Redis
	var queue []DownloadQueueItem
	{
		ctx := context.Background()
		client := GetRedisClient()
		TrailarrLog(INFO, "QUEUE", "[AddToDownloadQueue] RedisKey=%v, RedisClient=%#v", DownloadQueueRedisKey, client)
		items, err := client.LRange(ctx, DownloadQueueRedisKey, 0, -1).Result()
		if err == nil {
			for _, itemStr := range items {
				var item DownloadQueueItem
				if err := json.Unmarshal([]byte(itemStr), &item); err == nil {
					queue = append(queue, item)
				}
			}
		}
	}
	// Build a map for quick lookup of rejected extras using the hash
	ctx := context.Background()
	rejectedMap := make(map[string]RejectedExtra)
	extras, err := GetAllExtras(ctx)
	if err == nil {
		for _, e := range extras {
			if e.Status == "rejected" {
				rejectedMap[e.YoutubeId] = RejectedExtra{
					MediaType:  e.MediaType,
					MediaId:    e.MediaId,
					ExtraType:  e.ExtraType,
					ExtraTitle: e.ExtraTitle,
					YoutubeId:  e.YoutubeId,
					Reason:     e.Reason,
				}
			}
		}
	}
	// Load cache files (movies/series)
	var movieCache, seriesCache []map[string]interface{}
	movieCache, _ = LoadMediaFromRedis(MoviesJSONPath)
	seriesCache, _ = LoadMediaFromRedis(SeriesJSONPath)
	// Helper to check existence in cache
	existsInCache := func(yid string) bool {
		for _, m := range movieCache {
			if v, ok := m["youtubeId"]; ok && v == yid {
				return true
			}
		}
		for _, m := range seriesCache {
			if v, ok := m["youtubeId"]; ok && v == yid {
				return true
			}
		}
		return false
	}
	queueMutex.Lock()
	for _, id := range req.YoutubeIds {
		// 1. In-memory status
		if st, ok := downloadStatusMap[id]; ok {
			statuses[id] = st
			continue
		}
		// 2. Persistent queue file (last known status)
		found := false
		for i := len(queue) - 1; i >= 0; i-- {
			if queue[i].YouTubeID == id {
				statuses[id] = &DownloadStatus{Status: queue[i].Status, UpdatedAt: queue[i].QueuedAt}
				found = true
				break
			}
		}
		if found {
			continue
		}
		// 3. Rejected file
		if r, ok := rejectedMap[id]; ok {
			statuses[id] = &DownloadStatus{Status: "rejected", UpdatedAt: time.Now(), Error: r.Reason}
			continue
		}
		// 4. Cache files (exists)
		if existsInCache(id) {
			statuses[id] = &DownloadStatus{Status: "exists", UpdatedAt: time.Now()}
			continue
		}
		// 5. Fallback to missing
		statuses[id] = &DownloadStatus{Status: "missing"}
	}
	queueMutex.Unlock()
	// Log actual status values, not just pointers
	statusLog := make(map[string]DownloadStatus)
	for k, v := range statuses {
		if v != nil {
			statusLog[k] = *v
		}
	}
	TrailarrLog(INFO, "BATCH", "/api/extras/status/batch response: %+v", statusLog)
	c.JSON(http.StatusOK, BatchStatusResponse{Statuses: statuses})
}

// AddToDownloadQueue adds a new download request to the queue and persists in Redis
// source: "task" (block if queue not empty), "api" (always append)
func AddToDownloadQueue(item DownloadQueueItem, source string) {
	TrailarrLog(INFO, "QUEUE", "[AddToDownloadQueue] Entered. YouTubeID=%s, source=%s", item.YouTubeID, source)
	ctx := context.Background()
	client := GetRedisClient()

	// Removed obsolete 'queue ready' wait loop (was blocking forever)

	// If source is "task", block if queue not empty (i.e., if any item is queued or downloading)
	if source == "task" {
		for {
			queue, err := client.LRange(ctx, DownloadQueueRedisKey, 0, -1).Result()
			busy := false
			if err == nil {
				for _, qstr := range queue {
					var q DownloadQueueItem
					if err := json.Unmarshal([]byte(qstr), &q); err == nil {
						if q.Status == "queued" || q.Status == "downloading" {
							busy = true
							break
						}
					}
				}
			} else {
				TrailarrLog(ERROR, "QUEUE", "[AddToDownloadQueue] Error reading queue from Redis: %v", err)
			}
			if !busy {
				break
			}
			time.Sleep(2 * time.Second)
		}
	}

	// Lookup media title if not set
	if item.MediaTitle == "" {
		cacheFile, _ := resolveCachePath(item.MediaType)
		if cacheFile != "" {
			items, _ := loadCache(cacheFile)
			for _, m := range items {
				idInt, ok := parseMediaID(m["id"])
				if ok && idInt == item.MediaId {
					if t, ok := m["title"].(string); ok {
						item.MediaTitle = t
						break
					}
				}
			}
		}
	}
	item.Status = "queued"
	item.QueuedAt = time.Now()
	b, err := json.Marshal(item)
	TrailarrLog(INFO, "QUEUE", "[AddToDownloadQueue] Marshaled JSON: %s", string(b))
	if err != nil {
		TrailarrLog(ERROR, "QUEUE", "[AddToDownloadQueue] Failed to marshal item: %v", err)
		return
	}
	rpushRes := client.RPush(ctx, DownloadQueueRedisKey, b)
	TrailarrLog(INFO, "QUEUE", "[AddToDownloadQueue] RPush result: %+v", rpushRes)
	err = rpushRes.Err()
	if err != nil {
		TrailarrLog(ERROR, "QUEUE", "[AddToDownloadQueue] Failed to push to Redis: %v", err)
	} else {
		TrailarrLog(INFO, "QUEUE", "[AddToDownloadQueue] Successfully enqueued item. RedisKey=%s, YouTubeID=%s", DownloadQueueRedisKey, item.YouTubeID)
		// Broadcast updated queue to all WebSocket clients
		BroadcastDownloadQueueChanges([]DownloadQueueItem{item})
	}
	downloadStatusMap[item.YouTubeID] = &DownloadStatus{Status: "queued", UpdatedAt: time.Now()}
	TrailarrLog(INFO, "QUEUE", "[AddToDownloadQueue] Enqueued: mediaType=%v, mediaId=%v, extraType=%s, extraTitle=%s, youtubeId=%s, source=%s", item.MediaType, item.MediaId, item.ExtraType, item.ExtraTitle, item.YouTubeID, source)
}

// GetDownloadStatus returns the status for a YouTube ID
func GetDownloadStatus(youtubeID string) *DownloadStatus {
	queueMutex.Lock()
	defer queueMutex.Unlock()
	if status, ok := downloadStatusMap[youtubeID]; ok {
		return status
	}
	return nil
}

// NextQueuedItem fetches the next queued item from Redis and its index
func NextQueuedItem() (int, DownloadQueueItem, bool) {
	ctx := context.Background()
	client := GetRedisClient()
	queue, err := client.LRange(ctx, DownloadQueueRedisKey, 0, -1).Result()
	if err != nil {
		return -1, DownloadQueueItem{}, false
	}
	for i, qstr := range queue {
		var item DownloadQueueItem
		if err := json.Unmarshal([]byte(qstr), &item); err == nil {
			if item.Status == "queued" {
				return i, item, true
			}
		}
	}
	return -1, DownloadQueueItem{}, false
}

// StartDownloadQueueWorker starts a goroutine to process the download queue from Redis
func StartDownloadQueueWorker() {
	go func() {
		ctx := context.Background()
		client := GetRedisClient()
		// Clean the queue at startup
		_ = client.Del(ctx, DownloadQueueRedisKey).Err()
		for {
			idx, item, ok := NextQueuedItem()
			if !ok {
				time.Sleep(2 * time.Second)
				continue
			}
			// Prevent re-downloading if extra is rejected (hash-based)
			entry, err := GetExtraByYoutubeId(ctx, item.YouTubeID, item.MediaType, item.MediaId)
			if err == nil && entry != nil && entry.Status == "rejected" {
				TrailarrLog(WARN, "QUEUE", "[StartDownloadQueueWorker] Skipping rejected extra: mediaType=%v, mediaId=%v, extraType=%s, extraTitle=%s, youtubeId=%s", item.MediaType, item.MediaId, item.ExtraType, item.ExtraTitle, item.YouTubeID)
				// Remove from queue immediately
				b, _ := json.Marshal(item)
				_ = client.LRem(ctx, DownloadQueueRedisKey, 1, b).Err()
				BroadcastDownloadQueueChanges([]DownloadQueueItem{item})
				continue
			}
			// Mark as downloading in Redis
			queue, err := client.LRange(ctx, DownloadQueueRedisKey, 0, -1).Result()
			if err == nil && idx >= 0 && idx < len(queue) {
				var q DownloadQueueItem
				if err := json.Unmarshal([]byte(queue[idx]), &q); err == nil {
					q.Status = "downloading"
					b, _ := json.Marshal(q)
					_ = client.LSet(ctx, DownloadQueueRedisKey, int64(idx), b).Err()
					downloadStatusMap[item.YouTubeID] = &DownloadStatus{Status: "downloading", UpdatedAt: time.Now()}
					BroadcastDownloadQueueChanges([]DownloadQueueItem{q})
				}
			}
			// Perform the download
			meta, metaErr := DownloadYouTubeExtra(item.MediaType, item.MediaId, item.ExtraType, item.ExtraTitle, item.YouTubeID)
			// If we hit a 429, pause the queue for 5 minutes
			if metaErr != nil {
				if tooMany, ok := metaErr.(*TooManyRequestsError); ok {
					TrailarrLog(WARN, "QUEUE", "[StartDownloadQueueWorker] 429 detected, pausing queue for 5 minutes: %s", tooMany.Error())
					pauseUntil := time.Now().Add(5 * time.Minute)
					for time.Now().Before(pauseUntil) {
						TrailarrLog(INFO, "QUEUE", "[StartDownloadQueueWorker] Queue paused for 429. Resuming in %v seconds...", int(time.Until(pauseUntil).Seconds()))
						time.Sleep(30 * time.Second)
					}
					TrailarrLog(INFO, "QUEUE", "[StartDownloadQueueWorker] 5-minute pause for 429 complete. Resuming queue.")
				}
			}
			// Update status in Redis
			queue, err = client.LRange(ctx, DownloadQueueRedisKey, 0, -1).Result()
			var finalStatus string
			var failReason string
			if metaErr != nil {
				finalStatus = "failed"
				failReason = metaErr.Error()
				downloadStatusMap[item.YouTubeID] = &DownloadStatus{Status: finalStatus, UpdatedAt: time.Now(), Error: failReason}
			} else if meta != nil {
				finalStatus = meta.Status
				downloadStatusMap[item.YouTubeID] = &DownloadStatus{Status: finalStatus, UpdatedAt: time.Now()}
			} else {
				// meta == nil && metaErr == nil: treat as failed
				finalStatus = "failed"
				failReason = "No metadata returned from download"
				downloadStatusMap[item.YouTubeID] = &DownloadStatus{Status: finalStatus, UpdatedAt: time.Now(), Error: failReason}
			}
			// After download attempt, update status in Redis and broadcast only the final status
			if err == nil && idx >= 0 && idx < len(queue) {
				var q DownloadQueueItem
				if err := json.Unmarshal([]byte(queue[idx]), &q); err == nil {
					q.Status = finalStatus
					if finalStatus == "failed" && failReason != "" {
						q.Reason = failReason
					}
					b, _ := json.Marshal(q)
					_ = client.LSet(ctx, DownloadQueueRedisKey, int64(idx), b).Err()
					BroadcastDownloadQueueChanges([]DownloadQueueItem{q})
				}
			} else {
				// If we can't update in Redis, still broadcast the failed status
				item.Status = finalStatus
				if finalStatus == "failed" && failReason != "" {
					item.Reason = failReason
				}
				BroadcastDownloadQueueChanges([]DownloadQueueItem{item})
			}
			// Wait 10 seconds, then remove the item from the queue
			time.Sleep(10 * time.Second)
			// Remove the item at idx (by value, since Redis LREM removes by value)
			b, _ := json.Marshal(item)
			_ = client.LRem(ctx, DownloadQueueRedisKey, 1, b).Err()
		}
	}()
}

// GetDownloadStatusHandler returns the status of a download by YouTube ID
func GetDownloadStatusHandler(c *gin.Context) {
	youtubeId := c.Param("youtubeId")
	status := GetDownloadStatus(youtubeId)
	if status == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": status})
}

func DefaultYtdlpFlagsConfig() YtdlpFlagsConfig {
	return YtdlpFlagsConfig{
		Quiet:              false,
		NoProgress:         false,
		WriteSubs:          true,
		WriteAutoSubs:      true,
		EmbedSubs:          true,
		RemuxVideo:         "mkv",
		SubFormat:          "srt",
		SubLangs:           "es.*",
		RequestedFormats:   "best[height<=1080]",
		Timeout:            3.0,
		SleepInterval:      5.0,
		MaxDownloads:       5,
		LimitRate:          "30M",
		SleepRequests:      3.0,
		MaxSleepInterval:   120.0,
		CookiesFromBrowser: "chrome",
	}
}

// Deduplicate a slice of maps by a given key
func DeduplicateByKey(list []map[string]string, key string) []map[string]string {
	seen := make(map[string]bool)
	unique := make([]map[string]string, 0, len(list))
	for _, item := range list {
		k := item[key]
		if !seen[k] {
			unique = append(unique, item)
			seen[k] = true
		}
	}
	return unique
}

type ExtraDownloadMetadata struct {
	MediaType  MediaType // "movie" or "series"
	MediaId    int       // Radarr or Sonarr ID as int
	MediaTitle string    // Movie or Series title
	ExtraType  string    // e.g. "Trailer"
	ExtraTitle string    // e.g. "Official Trailer"
	YouTubeID  string
	FileName   string
	Status     string
}

// NewExtraDownloadMetadata constructs an ExtraDownloadMetadata with status and all fields
func NewExtraDownloadMetadata(info *downloadInfo, youtubeId string, status string) *ExtraDownloadMetadata {
	return &ExtraDownloadMetadata{
		MediaType:  info.MediaType,
		MediaId:    info.MediaId,
		MediaTitle: info.MediaTitle,
		ExtraTitle: info.ExtraTitle,
		ExtraType:  info.ExtraType,
		YouTubeID:  info.YouTubeID,
		FileName:   info.OutFile,
		Status:     status,
	}
}

type RejectedExtra struct {
	MediaType  MediaType `json:"mediaType"`
	MediaId    int       `json:"mediaId"`
	ExtraType  string    `json:"extraType"`
	ExtraTitle string    `json:"extraTitle"`
	YoutubeId  string    `json:"youtubeId"`
	Reason     string    `json:"reason"`
}

// No longer needed

func DownloadYouTubeExtra(mediaType MediaType, mediaId int, extraType, extraTitle, youtubeId string, forceDownload ...bool) (*ExtraDownloadMetadata, error) {
	TrailarrLog(DEBUG, "YouTube", "DownloadYouTubeExtra called with mediaType=%s, mediaId=%d, extraType=%s, extraTitle=%s, youtubeId=%s, forceDownload=%v",
		mediaType, mediaId, extraType, extraTitle, youtubeId, forceDownload)
	var downloadInfo *downloadInfo
	var err error

	// Lookup media title from cache for logging
	var mediaTitle string
	var cacheFile string
	cacheFile, _ = resolveCachePath(mediaType)
	if cacheFile != "" {
		items, _ := loadCache(cacheFile)
		for _, m := range items {
			idInt, ok := parseMediaID(m["id"])
			if ok && idInt == mediaId {
				if t, ok := m["title"].(string); ok {
					mediaTitle = t
					break
				}
			}
		}
	}
	TrailarrLog(INFO, "YouTube", "Downloading YouTube extra: mediaType=%s, mediaTitle=%s, type=%s, title=%s, youtubeId=%s",
		mediaType, mediaTitle, extraType, extraTitle, youtubeId)

	downloadInfo, err = prepareDownloadInfo(mediaType, mediaId, extraType, extraTitle, youtubeId)
	if err != nil {
		return nil, err
	}
	// Always clean up temp dir after download attempt
	defer func() {
		if downloadInfo != nil && downloadInfo.TempDir != "" {
			os.RemoveAll(downloadInfo.TempDir)
		}
	}()

	// Check if extra is rejected or already exists
	if meta, err := checkExistingExtra(downloadInfo, youtubeId); meta != nil || err != nil {
		return meta, err
	}

	// Perform the download
	return performDownload(downloadInfo, youtubeId)
}

type downloadInfo struct {
	MediaType  MediaType
	MediaId    int
	MediaTitle string
	OutDir     string
	OutFile    string
	TempDir    string
	TempFile   string
	YouTubeID  string
	ExtraType  string
	ExtraTitle string
	SafeTitle  string
}

func prepareDownloadInfo(mediaType MediaType, mediaId int, extraType, extraTitle, youtubeID string) (*downloadInfo, error) {
	// Robust base path resolution using cache and path mappings
	var basePath string
	var mappings [][]string
	var err error
	var cacheFile string
	var mediaTitle string

	// Step 1: Resolve cache path
	cacheFile, _ = resolveCachePath(mediaType)

	// Lookup media title from cache (mimic lookupMediaTitle from extras.go)
	if cacheFile != "" {
		items, _ := loadCache(cacheFile)
		for _, m := range items {
			idInt, ok := parseMediaID(m["id"])
			if ok && idInt == mediaId {
				if t, ok := m["title"].(string); ok {
					mediaTitle = t
					break
				}
			}
		}
	}

	// Step 2: Get path mappings using GetPathMappings
	mappings, err = GetPathMappings(mediaType)
	if err != nil {
		TrailarrLog(ERROR, "YouTube", "Failed to get path mappings: %v", err)
		mappings = [][]string{}
	}

	var mappedMediaPath string
	if err == nil && cacheFile != "" {
		// Step 3: Look up media path from cache using mediaId
		mediaPath, lookupErr := FindMediaPathByID(cacheFile, mediaId)
		if lookupErr == nil && mediaPath != "" && len(mappings) > 0 {
			// Step 4: Apply path mappings to convert root folder path
			mappedMediaPath = mediaPath
			for _, m := range mappings {
				if len(m) > 1 && strings.HasPrefix(mediaPath, m[0]) {
					mappedMediaPath = m[1] + mediaPath[len(m[0]):]
					break
				}
			}
		}
	}

	if mappedMediaPath != "" {
		basePath = mappedMediaPath
	} else if len(mappings) > 0 && len(mappings[0]) > 1 && mappings[0][1] != "" {
		basePath = mediaTitle
		if mediaTitle != "" {
			basePath = filepath.Join(mappings[0][1], mediaTitle)
		} else {
			basePath = mappings[0][1]
		}
	} else {
		if mediaTitle != "" {
			basePath = mediaTitle
		} else {
			basePath = ""
		}
	}

	canonicalType := canonicalizeExtraType(extraType)
	outDir := filepath.Join(basePath, canonicalType)

	// Sanitize title for filename
	forbidden := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	safeTitle := extraTitle
	for _, c := range forbidden {
		safeTitle = strings.ReplaceAll(safeTitle, c, "_")
	}

	outExt := "mkv"
	outFile := filepath.Join(outDir, fmt.Sprintf("%s.%s", safeTitle, outExt))

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "yt-dlp-tmp-*")
	if err != nil {
		TrailarrLog(ERROR, "YouTube", "Failed to create temp dir for yt-dlp: %v", err)
		return nil, fmt.Errorf("failed to create temp dir for yt-dlp: %w", err)
	}
	tempFile := filepath.Join(tempDir, fmt.Sprintf("%s.%s", safeTitle, outExt))

	TrailarrLog(DEBUG, "YouTube", "Resolved output directory: %s", outDir)
	TrailarrLog(DEBUG, "YouTube", "Resolved safe title: %s", safeTitle)
	TrailarrLog(DEBUG, "YouTube", "mediaType=%s, mediaTitle=%s, canonicalType=%s, outDir=%s, outFile=%s, tempDir=%s, tempFile=%s",
		mediaType, mediaTitle, canonicalType, outDir, outFile, tempDir, tempFile)

	return &downloadInfo{
		MediaType:  mediaType,
		MediaId:    mediaId,
		MediaTitle: mediaTitle,
		OutDir:     outDir,
		OutFile:    outFile,
		TempDir:    tempDir,
		TempFile:   tempFile,
		YouTubeID:  youtubeID,
		ExtraType:  extraType,
		ExtraTitle: extraTitle,
		SafeTitle:  safeTitle,
	}, nil
}

func checkExistingExtra(info *downloadInfo, youtubeId string) (*ExtraDownloadMetadata, error) {
	// Check if extra is in rejected_extras.json
	if meta := checkRejectedExtras(info, youtubeId); meta != nil {
		return meta, nil
	}

	// Skip download if file already exists
	if _, err := os.Stat(info.OutFile); err == nil {
		TrailarrLog(INFO, "YouTube", "File already exists, skipping: %s", info.OutFile)
		return NewExtraDownloadMetadata(info, youtubeId, "exists"), nil
	}

	return nil, nil
}

func checkRejectedExtras(info *downloadInfo, youtubeId string) *ExtraDownloadMetadata {
	// Use the hash-based approach: check if extra is marked as rejected in the hash
	ctx := context.Background()
	entry, err := GetExtraByYoutubeId(ctx, youtubeId, info.MediaType, info.MediaId)
	if err == nil && entry != nil && entry.Status == "rejected" {
		return NewExtraDownloadMetadata(info, youtubeId, "rejected")
	}
	return nil
}

func performDownload(info *downloadInfo, youtubeId string) (*ExtraDownloadMetadata, error) {

	// Build yt-dlp command args
	args := buildYtDlpArgs(info, youtubeId, true)
	// Execute yt-dlp command
	cmd := exec.Command("yt-dlp", args...)
	cmd.Dir = info.TempDir
	output, err := cmd.CombinedOutput()

	if err != nil && isImpersonationErrorNative(string(output)) {
		fmt.Printf("[DownloadYouTubeExtra] Impersonation failed, retrying without impersonation: %s\n", youtubeId)
		args = buildYtDlpArgs(info, youtubeId, false)
		cmd = exec.Command("yt-dlp", args...)
		cmd.Dir = info.TempDir
		output, err = cmd.CombinedOutput()
	}

	if len(output) > 0 {
		for _, line := range strings.Split(string(output), "\n") {
			if strings.TrimSpace(line) != "" {
				TrailarrLog(DEBUG, "YouTube", "yt-dlp output for %s: %s", youtubeId, line)
			}
		}
	}
	if err != nil {
		// Check for 429/Too Many Requests in output
		if strings.Contains(string(output), "429") || strings.Contains(strings.ToLower(string(output)), "too many requests") {
			return nil, &TooManyRequestsError{Message: "yt-dlp hit 429 Too Many Requests"}
		}
		return nil, handleDownloadErrorNative(info, youtubeId, err, string(output))
	}

	// Move file to final location
	if err := moveDownloadedFile(info); err != nil {
		return nil, err
	}

	// Create metadata
	return createSuccessMetadata(info, youtubeId)
}

// TooManyRequestsError is returned when a 429/Too Many Requests is detected
type TooManyRequestsError struct {
	Message string
}

func (e *TooManyRequestsError) Error() string {
	return e.Message
}

func isImpersonationErrorNative(output string) bool {
	return strings.Contains(output, "Impersonate target") ||
		strings.Contains(output, "is not available") ||
		strings.Contains(output, "missing dependencies required to support this target")
}

func buildYtDlpArgs(info *downloadInfo, youtubeId string, impersonate bool) []string {
	cfg, _ := GetYtdlpFlagsConfig()
	args := []string{
		"--no-progress",
		"--quiet",
		"--write-subs",
		"--write-auto-subs",
		"--embed-subs",
		"--sub-format", cfg.SubFormat,
		"--sub-langs", cfg.SubLangs,
		"--remux-video", cfg.RemuxVideo,
		"--format", cfg.RequestedFormats,
		"--output", info.TempFile,
		"--max-downloads", fmt.Sprintf("%d", cfg.MaxDownloads),
		"--limit-rate", cfg.LimitRate,
		"--sleep-interval", fmt.Sprintf("%.0f", cfg.SleepInterval),
		"--sleep-requests", fmt.Sprintf("%.0f", cfg.SleepRequests),
		"--max-sleep-interval", fmt.Sprintf("%.0f", cfg.MaxSleepInterval),
		"--socket-timeout", fmt.Sprintf("%.0f", cfg.Timeout),
	}

	if impersonate {
		args = append(args, "--impersonate", "chrome")
	}

	// args = append(args, "--cookies-from-browser", "chrome")
	args = append(args, "--cookies", CookiesFile)
	args = append(args, "--", youtubeId)
	return args
}

func handleDownloadErrorNative(info *downloadInfo, youtubeId string, err error, output string) error {
	reason := err.Error()
	if output != "" {
		reason += " | output: " + output
	}

	TrailarrLog(ERROR, "YouTube", "Download failed for %s: %s", youtubeId, reason)
	addToRejectedExtras(info, youtubeId, reason)
	// Also update the unified extras collection in Redis
	errMark := SetExtraRejectedPersistent(info.MediaType, info.MediaId, info.ExtraType, info.ExtraTitle, youtubeId, reason)
	if errMark != nil {
		TrailarrLog(ERROR, "YouTube", "Failed to mark extra as rejected in Redis: %v", errMark)
	}
	return fmt.Errorf(reason+": %w", err)
}

func addToRejectedExtras(info *downloadInfo, youtubeId, reason string) {
	// Use the hash-based approach: mark as rejected in the hash only if not already rejected
	ctx := context.Background()
	entry, err := GetExtraByYoutubeId(ctx, youtubeId, info.MediaType, info.MediaId)
	if err == nil && entry != nil && entry.Status == "rejected" {
		return
	}
	// Add or update as rejected
	_ = SetExtraRejectedPersistent(info.MediaType, info.MediaId, info.ExtraType, info.ExtraTitle, youtubeId, reason)
}

func moveDownloadedFile(info *downloadInfo) error {
	if _, statErr := os.Stat(info.TempFile); statErr != nil {
		TrailarrLog(ERROR, "YouTube", "yt-dlp did not produce expected output file: %s", info.TempFile)
		return fmt.Errorf("yt-dlp did not produce expected output file: %s", info.TempFile)
	}

	if err := os.MkdirAll(info.OutDir, 0755); err != nil {
		TrailarrLog(ERROR, "YouTube", "Failed to create output dir '%s': %v", info.OutDir, err)
		return fmt.Errorf("failed to create output dir '%s': %w", info.OutDir, err)
	}

	if moveErr := os.Rename(info.TempFile, info.OutFile); moveErr != nil {
		return handleCrossDeviceMove(info.TempFile, info.OutFile, moveErr)
	}

	return nil
}

func handleCrossDeviceMove(tempFile, outFile string, moveErr error) error {
	if linkErr, ok := moveErr.(*os.LinkError); ok && strings.Contains(linkErr.Error(), "cross-device link") {
		return copyFileAcrossDevices(tempFile, outFile)
	}
	TrailarrLog(ERROR, "YouTube", "Failed to move downloaded file to output dir: %v", moveErr)
	return fmt.Errorf("failed to move downloaded file to output dir: %w", moveErr)
}

func copyFileAcrossDevices(tempFile, outFile string) error {
	in, err := os.Open(tempFile)
	if err != nil {
		TrailarrLog(ERROR, "YouTube", "Failed to open temp file for copy: %v", err)
		return fmt.Errorf("failed to open temp file for copy: %w", err)
	}
	defer in.Close()

	out, err := os.Create(outFile)
	if err != nil {
		TrailarrLog(ERROR, "YouTube", "Failed to create output file for copy: %v", err)
		return fmt.Errorf("failed to create output file for copy: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		TrailarrLog(ERROR, "YouTube", "Failed to copy file across devices: %v", err)
		return fmt.Errorf("failed to copy file across devices: %w", err)
	}

	if err := out.Sync(); err != nil {
		TrailarrLog(ERROR, "YouTube", "Failed to sync output file: %v", err)
		return fmt.Errorf("failed to sync output file: %w", err)
	}

	if rmErr := os.Remove(tempFile); rmErr != nil {
		TrailarrLog(WARN, "YouTube", "Failed to remove temp file after copy: %v", rmErr)
	}

	return nil
}

func createSuccessMetadata(info *downloadInfo, youtubeId string) (*ExtraDownloadMetadata, error) {
	meta := NewExtraDownloadMetadata(info, youtubeId, "downloaded")

	// Add/update the extra in the unified collection in Redis
	entry := ExtrasEntry{
		MediaType:  info.MediaType,
		MediaId:    info.MediaId,
		ExtraTitle: info.ExtraTitle,
		ExtraType:  info.ExtraType,
		FileName:   info.OutFile,
		YoutubeId:  youtubeId,
		Status:     "downloaded",
	}
	ctx := context.Background()
	if err := AddOrUpdateExtra(ctx, entry); err != nil {
		TrailarrLog(WARN, "YouTube", "Failed to add/update extra in Redis after download: %v", err)
	}

	// Record download event in history
	cacheFile, _ := resolveCachePath(info.MediaType)
	mediaTitle := ""
	if cacheFile != "" {
		items, _ := loadCache(cacheFile)
		for _, m := range items {
			idInt, ok := parseMediaID(m["id"])
			if ok && idInt == info.MediaId {
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
	event := HistoryEvent{
		Action:     "download",
		MediaTitle: mediaTitle,
		MediaType:  info.MediaType,
		MediaId:    info.MediaId,
		ExtraType:  info.ExtraType,
		ExtraTitle: info.ExtraTitle,
		Date:       time.Now(),
	}
	_ = AppendHistoryEvent(event)

	metaFile := info.OutFile + ".json"
	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(metaFile, metaBytes, 0644)

	TrailarrLog(INFO, "YouTube", "Downloaded %s to %s", info.ExtraTitle, info.OutFile)
	return meta, nil
}

// YouTube trailer search proxy handler (POST: mediaType, mediaId)
func YouTubeTrailerSearchHandler(c *gin.Context) {
	var req struct {
		MediaType string `json:"mediaType"`
		MediaId   int    `json:"mediaId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.MediaType == "" || req.MediaId == 0 {
		TrailarrLog(WARN, "YouTube", "Invalid POST body for YouTube search: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing mediaType or mediaId"})
		return
	}
	TrailarrLog(INFO, "YouTube", "YouTubeTrailerSearchHandler POST: mediaType=%s, mediaId=%d", req.MediaType, req.MediaId)

	// Lookup media title/originalTitle
	var title, originalTitle string
	cacheFile, _ := resolveCachePath(MediaType(req.MediaType))
	items, err := loadCache(cacheFile)
	if err != nil {
		TrailarrLog(ERROR, "YouTube", "Failed to load cache for mediaType=%s: %v", req.MediaType, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Media cache not found"})
		return
	}
	for _, m := range items {
		idInt, ok := parseMediaID(m["id"])
		if ok && idInt == req.MediaId {
			if t, ok := m["title"].(string); ok {
				title = t
			}
			if ot, ok := m["originalTitle"].(string); ok {
				originalTitle = ot
			}
			break
		}
	}
	if title == "" && originalTitle == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
		return
	}

	// Search both originalTitle and title (if different), aggregate up to 10 unique results
	var searchTerms []string
	if originalTitle != "" {
		searchTerms = append(searchTerms, originalTitle)
	}
	if title != "" && originalTitle != title {
		searchTerms = append(searchTerms, title)
	}
	var allResults []gin.H
	videoIdSet := make(map[string]bool)
	for _, term := range searchTerms {
		searchQuery := term + " trailer"
		ytDlpArgs := []string{"-j", "ytsearch10:" + searchQuery, "--skip-download"}
		TrailarrLog(INFO, "YouTube", "yt-dlp command: yt-dlp %v", ytDlpArgs)
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		cmd := exec.CommandContext(ctx, "yt-dlp", ytDlpArgs...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			TrailarrLog(ERROR, "YouTube", "Failed to get StdoutPipe: %v", err)
			cancel()
			continue
		}
		if err := cmd.Start(); err != nil {
			TrailarrLog(ERROR, "YouTube", "Failed to start yt-dlp: %v", err)
			cancel()
			continue
		}
		TrailarrLog(INFO, "YouTube", "Started yt-dlp process (streaming)...")
		reader := bufio.NewReader(stdout)
		for {
			line, err := reader.ReadBytes('\n')
			if len(line) == 0 && err != nil {
				break
			}
			TrailarrLog(DEBUG, "YouTube", "Raw yt-dlp output line: %s", string(line))
			var item struct {
				ID          string `json:"id"`
				Title       string `json:"title"`
				Description string `json:"description"`
				Thumbnail   string `json:"thumbnail"`
				Channel     string `json:"channel"`
				ChannelID   string `json:"channel_id"`
			}
			parseErr := json.Unmarshal(bytes.TrimSpace(line), &item)
			if parseErr == nil {
				TrailarrLog(DEBUG, "YouTube", "Parsed yt-dlp item: id=%s title=%s", item.ID, item.Title)
				if item.ID != "" && !videoIdSet[item.ID] {
					allResults = append(allResults, gin.H{
						"id": gin.H{"videoId": item.ID},
						"snippet": gin.H{
							"title":       item.Title,
							"description": item.Description,
							"thumbnails": gin.H{
								"default": gin.H{"url": item.Thumbnail},
							},
							"channelTitle": item.Channel,
							"channelId":    item.ChannelID,
						},
					})
					videoIdSet[item.ID] = true
				}
			} else if len(bytes.TrimSpace(line)) > 0 {
				TrailarrLog(WARN, "YouTube", "Failed to parse yt-dlp output line: %s | error: %v", string(line), parseErr)
			}
			if len(allResults) >= 10 {
				break
			}
			if err != nil {
				if err != io.EOF {
					TrailarrLog(ERROR, "YouTube", "Reader error: %v", err)
				}
				break
			}
		}
		// Wait for process to finish or kill if we already have enough results
		if len(allResults) >= 10 {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
		cancel()
		if ctx.Err() == context.DeadlineExceeded {
			TrailarrLog(ERROR, "YouTube", "yt-dlp search timed out for query: %s", searchQuery)
			continue
		}
	}
	// Limit to 10 results
	if len(allResults) > 10 {
		allResults = allResults[:10]
	}
	TrailarrLog(INFO, "YouTube", "YouTubeTrailerSearchHandler returning %d results", len(allResults))
	c.JSON(http.StatusOK, gin.H{"items": allResults})
}

// urlQueryEscape safely escapes a string for use in a URL query
