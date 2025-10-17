package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var GlobalSyncQueue []TaskStatus

// Parametric force sync for Radarr/Sonarr
type SyncQueueItem struct {
	TaskId   string
	Queued   time.Time
	Started  time.Time
	Ended    time.Time
	Duration time.Duration
	Status   string
	Error    string
}

// Unified struct for queue, persistent state, and reporting
type TaskStatus struct {
	TaskId        string    `json:"taskId"`
	Name          string    `json:"name,omitempty"`
	Queued        time.Time `json:"queued,omitempty"`
	Started       time.Time `json:"started,omitempty"`
	Ended         time.Time `json:"ended,omitempty"`
	Duration      float64   `json:"duration,omitempty"`
	Interval      int       `json:"interval,omitempty"`
	LastExecution time.Time `json:"lastExecution,omitempty"`
	NextExecution time.Time `json:"nextExecution,omitempty"`
	Status        string    `json:"status"`
	Error         string    `json:"error,omitempty"`
}

// Unified Task struct: combines metadata, state, and scheduling info
type Task struct {
	Meta      TaskMeta
	State     TaskState
	Interval  int
	LogPrefix string
}

// Unified TaskSchedule struct for status/schedule reporting
type TaskSchedule struct {
	TaskID        TaskID    `json:"taskId"`
	Name          string    `json:"name"`
	Interval      int       `json:"interval"`
	LastExecution time.Time `json:"lastExecution"`
	LastDuration  float64   `json:"lastDuration"`
	NextExecution time.Time `json:"nextExecution"`
	Status        string    `json:"status"`
}

var taskStatusClientsMu sync.Mutex
var taskStatusClients = make(map[*websocket.Conn]struct{})

// Broadcasts the full status of all tasks, ignoring partial input
func broadcastTaskStatus(_ map[string]interface{}) {
	// Always send the current status of all tasks
	status := getCurrentTaskStatus()
	taskStatusClientsMu.Lock()
	for conn := range taskStatusClients {
		sendTaskStatus(conn, status)
	}
	taskStatusClientsMu.Unlock()
}

var GlobalTaskStates TaskStates

// Note: task_times are stored in Redis (TaskTimesRedisKey). Disk file support removed.

// TaskID is a string identifier for a scheduled task
type TaskID string

// TaskMeta holds static metadata for a task, including its function and order
type TaskMeta struct {
	ID       TaskID
	Name     string
	Function func()
	Order    int
}

// TaskState holds the persistent state for a scheduled task
type TaskState struct {
	ID            TaskID    `json:"taskId"`
	LastExecution time.Time `json:"lastExecution"`
	LastDuration  float64   `json:"lastDuration"`
	Status        string    `json:"status"`
}

// TaskStates maps TaskID to TaskState
type TaskStates map[TaskID]TaskState

// tasksMeta holds all static task metadata, including the function
var tasksMeta map[TaskID]TaskMeta

func init() {
	// On startup, update any 'running' tasks in Redis queue to 'queued'
	client := GetRedisClient()
	ctx := context.Background()
	vals, err := client.LRange(ctx, TaskQueueRedisKey, 0, -1).Result()
	if err == nil {
		for i, v := range vals {
			var qi SyncQueueItem
			if err := json.Unmarshal([]byte(v), &qi); err == nil {
				if qi.Status == "running" {
					qi.Status = "queued"
					if b, err := json.Marshal(qi); err == nil {
						// set the element back at index i
						_ = client.LSet(ctx, TaskQueueRedisKey, int64(i), b).Err()
					}
				}
			}
		}
	}
	tasksMeta = map[TaskID]TaskMeta{
		"radarr": {ID: "radarr", Name: "Sync with Radarr", Function: wrapWithQueue("radarr", func() error { return SyncMediaType(MediaTypeMovie) }), Order: 1},
		"sonarr": {ID: "sonarr", Name: "Sync with Sonarr", Function: wrapWithQueue("sonarr", func() error { return SyncMediaType(MediaTypeTV) }), Order: 2},
		"extras": {ID: "extras", Name: "Search for Missing Extras", Function: wrapWithQueue("extras", func() error { processExtras(context.Background()); return nil }), Order: 3},
	}
}

// Helper to get all known TaskIDs
func AllTaskIDs() []TaskID {
	ids := make([]TaskID, 0, len(tasksMeta))
	for id := range tasksMeta {
		ids = append(ids, id)
	}
	return ids
}

func LoadTaskStates() (TaskStates, error) {
	// Redis-backed task states; disk fallback removed
	client := GetRedisClient()
	ctx := context.Background()
	vals, err := client.LRange(ctx, TaskTimesRedisKey, 0, -1).Result()

	states := make(TaskStates)
	if err == nil && len(vals) > 0 {
		states = parseStatesFromVals(vals)
	}

	if len(states) == 0 {
		initializeDefaultStates(states)
		_ = saveTaskStates(states)
	}

	ensureAllTasksExist(states)

	GlobalTaskStates = states
	return states, nil
}

// parseStatesFromVals parses JSON entries from Redis into TaskStates and marks them idle.
func parseStatesFromVals(vals []string) TaskStates {
	states := make(TaskStates)
	for _, v := range vals {
		var t TaskState
		if err := json.Unmarshal([]byte(v), &t); err == nil {
			t.Status = "idle"
			states[t.ID] = t
		}
	}
	return states
}

// initializeDefaultStates populates states with sensible defaults based on Timings.
func initializeDefaultStates(states TaskStates) {
	zeroTime := time.Time{}
	for id := range tasksMeta {
		interval := 0
		if v, ok := Timings[string(id)]; ok {
			interval = v
		}
		if interval == 0 {
			states[id] = TaskState{ID: id, LastExecution: time.Now(), LastDuration: 0}
		} else {
			states[id] = TaskState{ID: id, LastExecution: zeroTime, LastDuration: 0}
		}
	}
}

// ensureAllTasksExist makes sure every task from tasksMeta has an entry in states.
func ensureAllTasksExist(states TaskStates) {
	for id := range tasksMeta {
		if _, ok := states[id]; !ok {
			states[id] = TaskState{ID: id}
		}
	}
}

func saveTaskStates(states TaskStates) error {
	GlobalTaskStates = states
	arr := make([]struct {
		ID            TaskID    `json:"taskId"`
		LastExecution time.Time `json:"lastExecution"`
		LastDuration  float64   `json:"lastDuration"`
	}, 0, len(states))
	for id, t := range states {
		taskId := t.ID
		if taskId == "" {
			taskId = id
		}
		arr = append(arr, struct {
			ID            TaskID    `json:"taskId"`
			LastExecution time.Time `json:"lastExecution"`
			LastDuration  float64   `json:"lastDuration"`
		}{
			ID:            taskId,
			LastExecution: t.LastExecution,
			LastDuration:  t.LastDuration,
		})
	}
	// Persist to Redis as list of task states (overwrite by deleting and RPUSH)
	client := GetRedisClient()
	ctx := context.Background()
	_ = client.Del(ctx, TaskTimesRedisKey).Err()
	for _, s := range arr {
		if b, err := json.Marshal(s); err == nil {
			_ = client.RPush(ctx, TaskTimesRedisKey, b).Err()
		}
	}
	return nil
}

func GetAllTasksStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use running flags for Radarr/Sonarr
		states := GlobalTaskStates
		schedules := buildSchedules(states)
		respondJSON(c, http.StatusOK, gin.H{
			"schedules": schedules,
		})
	}
}

// Helper to calculate next execution time
func calcNext(lastExecution time.Time, interval int) time.Time {
	if lastExecution.IsZero() {
		return time.Now().Add(time.Duration(interval) * time.Minute)
	}
	return lastExecution.Add(time.Duration(interval) * time.Minute)
}

// Helper to build schedules array
func buildSchedules(states TaskStates) []TaskSchedule {
	schedules := make([]TaskSchedule, 0, len(tasksMeta))
	// Build a slice of (order, id) pairs for sorting
	type orderedTask struct {
		order int
		id    TaskID
	}
	ordered := make([]orderedTask, 0, len(tasksMeta))
	for id, meta := range tasksMeta {
		ordered = append(ordered, orderedTask{order: meta.Order, id: id})
	}
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].order < ordered[j].order
	})
	for _, ot := range ordered {
		meta := tasksMeta[ot.id]
		state := states[ot.id]
		interval := Timings[string(ot.id)]
		schedules = append(schedules, TaskSchedule{
			TaskID:        state.ID,
			Name:          meta.Name,
			Interval:      interval,
			LastExecution: state.LastExecution,
			LastDuration:  state.LastDuration,
			NextExecution: calcNext(state.LastExecution, interval),
			Status:        state.Status,
		})
	}
	return schedules
}

func buildTaskQueues() []TaskStatus {
	return GlobalSyncQueue
}

func sortTaskQueuesByQueuedDesc(queues []TaskStatus) {
	sort.Slice(queues, func(i, j int) bool {
		return queues[i].Queued.After(queues[j].Queued)
	})
}

// pushTaskQueueItem appends a sync queue item to Redis list
func pushTaskQueueItem(item SyncQueueItem) error {
	client := GetRedisClient()
	ctx := context.Background()
	b, err := json.Marshal(item)
	if err != nil {
		return err
	}
	if err := client.RPush(ctx, TaskQueueRedisKey, b).Err(); err != nil {
		return err
	}
	// Trim to reasonable max
	return client.LTrim(ctx, TaskQueueRedisKey, -int64(TaskQueueMaxLen), -1).Err()
}

// updateTaskQueueItem searches from the end to find a matching TaskId+Queued and updates it
func updateTaskQueueItem(taskId string, queued time.Time, update func(*SyncQueueItem)) error {
	client := GetRedisClient()
	ctx := context.Background()
	vals, err := client.LRange(ctx, TaskQueueRedisKey, 0, -1).Result()
	if err != nil {
		return err
	}
	// search from the back
	for i := len(vals) - 1; i >= 0; i-- {
		var qi SyncQueueItem
		if err := json.Unmarshal([]byte(vals[i]), &qi); err != nil {
			continue
		}
		if qi.TaskId == taskId && qi.Queued.Equal(queued) {
			update(&qi)
			b, err := json.Marshal(qi)
			if err != nil {
				return err
			}
			// set list element at index i
			return client.LSet(ctx, TaskQueueRedisKey, int64(i), b).Err()
		}
	}
	return nil
}

func TaskHandler() gin.HandlerFunc {
	// Build tasks map from tasksMeta
	type forceTask struct {
		id       TaskID
		started  *bool
		syncFunc func()
		respond  string
	}
	tasks := make(map[string]forceTask)
	for id, meta := range tasksMeta {
		if meta.Function == nil {
			TrailarrLog(WARN, "Tasks", "No sync function for taskId=%s", id)
			continue
		}
		tasks[string(id)] = forceTask{
			id:       id,
			started:  nil,
			syncFunc: meta.Function,
			respond:  fmt.Sprintf("Sync %s forced", meta.Name),
		}
	}
	return func(c *gin.Context) {
		var req struct {
			TaskId string `json:"taskId"`
		}
		if err := c.BindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request")
			return
		}
		println("[FORCE] Requested force execution for:", req.TaskId)
		t, ok := tasks[req.TaskId]
		if !ok {
			respondError(c, http.StatusBadRequest, "unknown task")
			return
		}
		// Run all tasks async, status managed in goroutine
		go func(taskId TaskID, syncFunc func()) {
			// Copy current in-memory state to avoid overwriting other running statuses
			states := make(TaskStates)
			for k, v := range GlobalTaskStates {
				states[k] = v
			}
			// Set running flag for this task only
			states[taskId] = TaskState{
				ID:            taskId,
				LastExecution: states[taskId].LastExecution,
				LastDuration:  states[taskId].LastDuration,
				Status:        "running",
			}
			GlobalTaskStates = states
			broadcastTaskStatus(getCurrentTaskStatus())
			start := time.Now()
			syncFunc()
			duration := time.Since(start)
			// Set idle flag for this task only
			states[taskId] = TaskState{
				ID:            taskId,
				LastExecution: start,
				LastDuration:  duration.Seconds(),
				Status:        "idle",
			}
			GlobalTaskStates = states
			broadcastTaskStatus(getCurrentTaskStatus())
			saveTaskStates(states)
		}(t.id, t.syncFunc)
		respondJSON(c, http.StatusOK, gin.H{"status": t.respond})
	}
}

type bgTask struct {
	id        TaskID
	started   *bool
	syncFunc  func()
	interval  time.Duration
	lastExec  time.Time
	logPrefix string
}

func StartBackgroundTasks() {
	TrailarrLog(INFO, "Tasks", "StartBackgroundTasks called. PID=%d, time=%s", os.Getpid(), time.Now().Format(time.RFC3339Nano))
	states, err := LoadTaskStates()
	if err != nil {
		TrailarrLog(WARN, "Tasks", "Could not load last task times: %v", err)
	}

	taskList := buildBgTasks(states)

	for i := range taskList {
		t := taskList[i]
		if t.interval <= 0 {
			TrailarrLog(WARN, "Tasks", "Task %s has non-positive interval, skipping scheduling", t.logPrefix)
			continue
		}
		go scheduleTask(t)
	}
	TrailarrLog(INFO, "Tasks", "Native Go scheduler started. Jobs will persist last execution times to Redis key %s", TaskTimesRedisKey)
}

func buildBgTasks(states TaskStates) []bgTask {
	var taskList []bgTask
	for id, meta := range tasksMeta {
		intervalVal, ok := Timings[string(id)]
		if !ok {
			TrailarrLog(WARN, "Tasks", "No interval found in Timings for %s", id)
			intervalVal = 0
		}
		interval := time.Duration(intervalVal) * time.Minute
		lastExec := states[id].LastExecution
		if meta.Function == nil {
			TrailarrLog(WARN, "Tasks", "No sync function for taskId=%s", id)
			continue
		}
		taskList = append(taskList, bgTask{
			id:        id,
			started:   nil,
			syncFunc:  meta.Function,
			interval:  interval,
			lastExec:  lastExec,
			logPrefix: meta.Name,
		})
	}
	return taskList
}

func scheduleTask(t bgTask) {
	now := time.Now()
	initialDelay := t.lastExec.Add(t.interval).Sub(now)
	if initialDelay < 0 {
		initialDelay = 0
	}
	time.Sleep(initialDelay)

	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	for {
		if t.id == "extras" {
			// Wait until radarr and sonarr have executed at least once
			for {
				st := GlobalTaskStates
				radLast := st["radarr"].LastExecution
				sonLast := st["sonarr"].LastExecution
				if !radLast.IsZero() && !sonLast.IsZero() {
					break
				}
				TrailarrLog(INFO, "Tasks", "Waiting for radarr/sonarr to run before extras")
				time.Sleep(5 * time.Second)
			}
		}
		go runTaskAsync(TaskID(t.id), t.syncFunc)
		<-ticker.C
	}
}

func processExtras(ctx context.Context) {
	// Clean all 429 rejections before starting extras task
	if err := RemoveAll429Rejections(); err != nil {
		TrailarrLog(WARN, "Tasks", "Failed to clean 429 rejections: %v", err)
	} else {
		TrailarrLog(INFO, "Tasks", "Cleaned all 429 rejections before starting extras task.")
	}
	extraTypesCfg, err := GetExtraTypesConfig()
	if err != nil {
		TrailarrLog(WARN, "Tasks", "Could not load extra types config: %v", err)
		return
	}
	TrailarrLog(INFO, "Tasks", "[TASK] Searching for missing movie extras...")
	downloadMissingExtrasWithTypeFilter(ctx, extraTypesCfg, MediaTypeMovie, MoviesRedisKey)
	TrailarrLog(INFO, "Tasks", "[TASK] Searching for missing series extras...")
	downloadMissingExtrasWithTypeFilter(ctx, extraTypesCfg, MediaTypeTV, SeriesRedisKey)
}

func StopExtrasDownloadTask() {
	states, _ := LoadTaskStates()
	if states["extras"].Status == "running" {
		TrailarrLog(INFO, "Tasks", "Stopping extras download task... extrasTaskState.Status=%v", states["extras"].Status)
		states["extras"] = TaskState{
			ID:            "extras",
			LastExecution: states["extras"].LastExecution,
			LastDuration:  states["extras"].LastDuration,
			Status:        "idle",
		}
		saveTaskStates(states)
	} else {
		TrailarrLog(INFO, "Tasks", "StopExtrasDownloadTask called but extrasTaskState.Status is not running")
	}
}

// Shared logic for type-filtered extras download
func downloadMissingExtrasWithTypeFilter(ctx context.Context, cfg ExtraTypesConfig, mediaType MediaType, cacheFile string) {

	items, err := loadCache(cacheFile)
	if err != nil {
		return
	}

	enabledTypes := GetEnabledCanonicalExtraTypes(cfg)
	// Filter items using the same wanted logic as GetMissingExtrasHandler
	wantedItems := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		id := item["id"]
		var mediaId int
		switch v := id.(type) {
		case float64:
			mediaId = int(v)
		case int:
			mediaId = v
		default:
			continue
		}
		if !HasAnyEnabledExtras(mediaType, mediaId, enabledTypes) {
			wantedItems = append(wantedItems, item)
		}
	}

	for _, item := range wantedItems {
		if ctx != nil && ctx.Err() != nil {
			TrailarrLog(INFO, "Tasks", "Extras download cancelled before processing item.")
			break
		}
		processWantedItem(ctx, cfg, mediaType, cacheFile, item, enabledTypes)
	}
}

// processWantedItem encapsulates per-item processing previously inline in the large function.
func processWantedItem(ctx context.Context, cfg ExtraTypesConfig, mediaType MediaType, cacheFile string, item map[string]interface{}, enabledTypes interface{}) {
	mediaId, _ := parseMediaID(item["id"])
	title, _ := item["title"].(string)

	extras, usedTMDB, err := fetchExtrasOrTMDB(mediaType, mediaId, title, enabledTypes)
	if err != nil {
		TrailarrLog(WARN, "Tasks", "SearchExtras/TMDB failed for mediaId=%v, title=%q: %v", mediaId, title, err)
		return
	}
	if len(extras) == 0 {
		// Nothing to do
		return
	}

	mediaPath, err := FindMediaPathByID(cacheFile, mediaId)
	if err != nil || mediaPath == "" {
		TrailarrLog(WARN, "Tasks", "FindMediaPathByID failed for mediaId=%v, title=%q: %v", mediaId, title, err)
		return
	}

	TrailarrLog(INFO, "Tasks", "Searching extras for %s %v: %s", mediaType, mediaId, item["title"])

	var toDownload []Extra
	if usedTMDB {
		toDownload = extras
	} else {
		MarkDownloadedExtras(extras, mediaPath, "type", "title")
		// Defensive: mark rejected extras before any download
		rejectedExtras := GetRejectedExtrasForMedia(mediaType, mediaId)
		rejectedYoutubeIds := make(map[string]struct{}, len(rejectedExtras))
		for _, r := range rejectedExtras {
			rejectedYoutubeIds[r.YoutubeId] = struct{}{}
		}
		MarkRejectedExtrasInMemory(extras, rejectedYoutubeIds)
		toDownload = extras
	}

	// For each extra, download sequentially using a helper to reduce nesting.
	for _, extra := range toDownload {
		if ctx != nil && ctx.Err() != nil {
			TrailarrLog(INFO, "Tasks", "Extras download cancelled before processing extra.")
			break
		}
		processExtraDownload(ctx, cfg, mediaType, mediaId, extra, usedTMDB)
	}
}

// fetchExtrasOrTMDB centralizes SearchExtras + TMDB fallback and reduces branching in the caller.
func fetchExtrasOrTMDB(mediaType MediaType, mediaId int, title string, enabledTypes interface{}) ([]Extra, bool, error) {
	extras, err := SearchExtras(mediaType, mediaId)
	if err != nil {
		return nil, false, err
	}
	if len(extras) == 0 {
		TrailarrLog(INFO, "Tasks", "No extras found for mediaId=%v, title=%q, enabledTypes=%v, attempting TMDB fetch...", mediaId, title, enabledTypes)
		tmdbExtras, err := FetchTMDBExtrasForMedia(mediaType, mediaId)
		if err != nil {
			return nil, false, err
		}
		if len(tmdbExtras) == 0 {
			TrailarrLog(INFO, "Tasks", "Still no extras after TMDB fetch for mediaId=%v, title=%q", mediaId, title)
			return nil, false, nil
		}
		return tmdbExtras, true, nil
	}
	return extras, false, nil
}

// processExtraDownload handles the per-extra checks and enqueues downloads when appropriate.
func processExtraDownload(ctx context.Context, cfg ExtraTypesConfig, mediaType MediaType, mediaId int, extra Extra, usedTMDB bool) {
	typ := canonicalizeExtraType(extra.ExtraType)
	if !isExtraTypeEnabled(cfg, typ) {
		return
	}
	// Only check rejection for local extras, not TMDB-fetched
	if !usedTMDB && extra.Status == "rejected" {
		return
	}
	// For TMDB-fetched, always treat as missing if not present locally
	if (usedTMDB && extra.YoutubeId != "") || (!usedTMDB && extra.Status == "missing" && extra.YoutubeId != "") {
		if err := handleTypeFilteredExtraDownload(mediaType, mediaId, extra); err != nil {
			TrailarrLog(WARN, "Tasks", "[SEQ] Download failed: %v", err)
		}
	}
}

// Handles downloading a single extra and appending to history if successful
func handleTypeFilteredExtraDownload(mediaType MediaType, mediaId int, extra Extra) error {
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
	TrailarrLog(INFO, "QUEUE", "[handleTypeFilteredExtraDownload] Enqueued extra: mediaType=%v, mediaId=%v, type=%s, title=%s, youtubeId=%s", mediaType, mediaId, extra.ExtraType, extra.ExtraTitle, extra.YoutubeId)

	// Do not record a "queued" history event here. The downloader will record
	// the final "download" event when the download completes.
	_, _ = resolveCachePath(mediaType) // keep functionality that may require cache resolution
	return nil
}

// Helper: check if extra type is enabled in config
func isExtraTypeEnabled(cfg ExtraTypesConfig, typ string) bool {
	switch typ {
	case "Trailers":
		return cfg.Trailers
	case "Scenes":
		return cfg.Scenes
	case "Behind The Scenes":
		return cfg.BehindTheScenes
	case "Interviews":
		return cfg.Interviews
	case "Featurettes":
		return cfg.Featurettes
	case "Deleted Scenes":
		return cfg.DeletedScenes
	case "Other":
		return cfg.Other
	default:
		return false
	}
}

func getWebSocketUpgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
}

func addTaskStatusClient(conn *websocket.Conn) {
	taskStatusClientsMu.Lock()
	taskStatusClients[conn] = struct{}{}
	taskStatusClientsMu.Unlock()
	// Send initial status
	go sendCurrentTaskStatus(conn)
}

func removeTaskStatusClient(conn *websocket.Conn) {
	taskStatusClientsMu.Lock()
	delete(taskStatusClients, conn)
	taskStatusClientsMu.Unlock()
}

func sendCurrentTaskStatus(conn *websocket.Conn) {
	status := getCurrentTaskStatus()
	sendTaskStatus(conn, status)
}

func sendTaskStatus(conn *websocket.Conn, status interface{}) {
	data, err := json.Marshal(status)
	if err != nil {
		return
	}
	conn.WriteMessage(websocket.TextMessage, data)
}

// Returns a map with all tasks' current status for broadcasting
func getCurrentTaskStatus() map[string]interface{} {
	states := GlobalTaskStates
	return map[string]interface{}{
		"schedules": buildSchedules(states),
	}
}

// Helper to run a task async and manage status
func runTaskAsync(taskId TaskID, syncFunc func()) {
	// Set running flag
	GlobalTaskStates[taskId] = TaskState{
		ID:            taskId,
		LastExecution: GlobalTaskStates[taskId].LastExecution, // unchanged until end
		LastDuration:  GlobalTaskStates[taskId].LastDuration,
		Status:        "running",
	}
	broadcastTaskStatus(getCurrentTaskStatus())
	start := time.Now()
	syncFunc()
	duration := time.Since(start)
	// Set idle flag and update LastExecution to NOW (end of task)
	GlobalTaskStates[taskId] = TaskState{
		ID:            taskId,
		LastExecution: time.Now(),
		LastDuration:  duration.Seconds(),
		Status:        "idle",
	}
	broadcastTaskStatus(getCurrentTaskStatus())
	saveTaskStates(GlobalTaskStates)
}

// Handler to return only the queue items as 'queues' array
func GetTaskQueueHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		queues := buildTaskQueues()
		sortTaskQueuesByQueuedDesc(queues)
		c.JSON(http.StatusOK, gin.H{
			"queues": queues,
		})
	}
}

// Centralized queue wrapper for all tasks
func wrapWithQueue(taskId TaskID, syncFunc func() error) func() {
	return func() {
		// Add new queue item to Redis on start
		queued := time.Now()
		item := SyncQueueItem{
			TaskId:  string(taskId),
			Queued:  queued,
			Status:  "running",
			Started: queued,
		}
		_ = pushTaskQueueItem(item)

		err := syncFunc()
		ended := time.Now()
		duration := ended.Sub(queued)
		status := "success"
		if err != nil {
			status = "failed"
			TrailarrLog(ERROR, "Tasks", "Task %s error: %s", taskId, err.Error())
		} else {
			TrailarrLog(INFO, "Tasks", "Task %s completed successfully.", taskId)
		}
		// Update the last queue item for this task (by TaskId and Queued) in Redis
		_ = updateTaskQueueItem(string(taskId), queued, func(qi *SyncQueueItem) {
			qi.Status = status
			qi.Started = queued
			qi.Ended = ended
			qi.Duration = duration
			if err != nil {
				qi.Error = err.Error()
			} else {
				qi.Error = ""
			}
		})
	}
}
