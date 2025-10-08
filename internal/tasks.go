package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
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

const taskTimesFile = "task_times.json"

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
	// On startup, update any 'running' tasks in queue.json to 'queued'
	var fileQueue []SyncQueueItem
	if err := ReadJSONFile(QueueFile, &fileQueue); err == nil {
		changed := false
		for i := range fileQueue {
			if fileQueue[i].Status == "running" {
				fileQueue[i].Status = "queued"
				changed = true
			}
		}
		if changed {
			_ = WriteJSONFile(QueueFile, fileQueue)
		}
	}
	tasksMeta = map[TaskID]TaskMeta{
		"radarr": {ID: "radarr", Name: "Sync with Radarr", Function: wrapWithQueue("radarr", func() error { SyncRadarr(); return nil }), Order: 1},
		"sonarr": {ID: "sonarr", Name: "Sync with Sonarr", Function: wrapWithQueue("sonarr", func() error { SyncSonarr(); return nil }), Order: 2},
		"extras": {ID: "extras", Name: "Search for Missing Extras", Function: wrapWithQueue("extras", func() error { handleExtrasDownloadLoop(context.Background()); return nil }), Order: 3},
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
	var arr []TaskState
	path := filepath.Join(TrailarrRoot, taskTimesFile)
	err := ReadJSONFile(path, &arr)
	states := make(TaskStates)
	// If file is missing or corrupt, always create/reset with default values
	if err != nil {
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
		_ = saveTaskStates(states)
		GlobalTaskStates = states
		return states, nil
	}
	// Map array entries to map by TaskId
	for _, t := range arr {
		if t.ID == "" {
			continue // skip invalid entries
		}
		// Ignore persisted Status, always set to "idle" on load
		t.Status = "idle"
		states[t.ID] = t
	}
	// Ensure all tasks exist
	for id := range tasksMeta {
		if _, ok := states[id]; !ok {
			states[id] = TaskState{ID: id}
		}
	}
	GlobalTaskStates = states
	return states, nil
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
	path := filepath.Join(TrailarrRoot, taskTimesFile)
	err := WriteJSONFile(path, arr)
	if err != nil {
		TrailarrLog(ERROR, "Tasks", "[saveTaskStates] Failed to write %s: %v", path, err)
	} else {
	}
	return err
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

func StartBackgroundTasks() {
	TrailarrLog(INFO, "Tasks", "StartBackgroundTasks called. PID=%d, time=%s", os.Getpid(), time.Now().Format(time.RFC3339Nano))
	states, err := LoadTaskStates()
	if err != nil {
		TrailarrLog(WARN, "Tasks", "Could not load last task times: %v", err)
	}

	type bgTask struct {
		id        TaskID
		started   *bool
		syncFunc  func()
		interval  time.Duration
		lastExec  time.Time
		logPrefix string
	}
	// Build taskList from tasksMeta
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

	// Native Go scheduler: one goroutine per task using time.Ticker
	for i := range taskList {
		task := &taskList[i]
		interval := task.interval
		if interval <= 0 {
			TrailarrLog(WARN, "Tasks", "Task %s has non-positive interval, skipping scheduling", task.logPrefix)
			continue
		}
		go func(t bgTask) {
			now := time.Now()
			initialDelay := t.lastExec.Add(interval).Sub(now)
			if initialDelay < 0 {
				initialDelay = 0
			}
			time.Sleep(initialDelay)
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				go runTaskAsync(TaskID(t.id), t.syncFunc)
				<-ticker.C
			}
		}(*task)
	}
	TrailarrLog(INFO, "Tasks", "Native Go scheduler started. Jobs will persist last execution times in %s", taskTimesFile)
}

func StartExtrasDownloadTask() {
	TrailarrLog(INFO, "Tasks", "[EXTRAS-TRIGGER] StartExtrasDownloadTask called. PID=%d, time=%s", os.Getpid(), time.Now().Format(time.RFC3339Nano))
	TrailarrLog(INFO, "Tasks", "[EXTRAS-TRIGGER] Call stack: %s", getStackTrace())
	TrailarrLog(INFO, "Tasks", "Starting extras download task (manual trigger)...")
	// Manual/forced run: runTaskAsync with the actual sync logic
	states := make(TaskStates)
	for k, v := range GlobalTaskStates {
		states[k] = v
	}
	go runTaskAsync("extras", func() { handleExtrasDownloadLoop(context.Background()) })
}

func handleExtrasDownloadLoop(ctx context.Context) bool {
	TrailarrLog(INFO, "Tasks", "[EXTRAS-TRIGGER] handleExtrasDownloadLoop entered. PID=%d, time=%s", os.Getpid(), time.Now().Format(time.RFC3339Nano))
	TrailarrLog(INFO, "Tasks", "[EXTRAS-TRIGGER] Call stack: %s", getStackTrace())
	interval := getExtrasInterval()
	processExtras(ctx)
	result := waitOrDone(ctx, time.Duration(interval)*time.Minute)
	return result
}

// getStackTrace returns a string with the current call stack for debugging triggers
func getStackTrace() string {
	buf := make([]byte, 2048)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

func waitOrDone(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		TrailarrLog(INFO, "Tasks", "[TASK] Extras download task stopped by cancel.")
		return true
	case <-time.After(d):
		return false
	}
}

func getExtrasInterval() int {
	interval := 360 // default 6 hours
	if v, ok := Timings["extras"]; ok {
		interval = v
	}
	return interval
}

func processExtras(ctx context.Context) {
	extraTypesCfg, err := GetExtraTypesConfig()
	CheckErrLog(WARN, "Tasks", "Could not load extra types config", err)
	TrailarrLog(INFO, "Tasks", "[TASK] Searching for missing movie extras...")
	downloadMissingExtrasWithTypeFilter(ctx, extraTypesCfg, MediaTypeMovie, MoviesJSONPath)
	TrailarrLog(INFO, "Tasks", "[TASK] Searching for missing series extras...")
	downloadMissingExtrasWithTypeFilter(ctx, extraTypesCfg, MediaTypeTV, SeriesJSONPath)
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

	for _, item := range items {
		if ctx != nil && ctx.Err() != nil {
			TrailarrLog(INFO, "Tasks", "Extras download cancelled before processing item.")
			break
		}
		// Skip items that have any extras of enabled types
		enabledTypes := GetEnabledCanonicalExtraTypes(cfg)
		if HasAnyEnabledExtras(item, enabledTypes) {
			continue // skip this item, it has at least one extra of an enabled type
		}
		mediaId, ok := parseMediaID(item["id"])
		if !ok {
			continue
		}
		extras, err := SearchExtras(mediaType, mediaId)
		if err != nil {
			TrailarrLog(WARN, "Tasks", "SearchExtras failed for mediaId=%v: %v", mediaId, err)
			continue
		}
		mediaPath, err := FindMediaPathByID(cacheFile, mediaId)
		if err != nil || mediaPath == "" {
			TrailarrLog(WARN, "Tasks", "FindMediaPathByID failed for mediaId=%v: %v", mediaId, err)
			continue
		}
		TrailarrLog(INFO, "Tasks", "Searching extras for %s %v: %s", mediaType, mediaId, item["title"])
		MarkDownloadedExtras(extras, mediaPath, "type", "title")
		// Defensive: mark rejected extras before any download
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
		// For each extra, download sequentially
		for _, extra := range extras {
			if ctx != nil && ctx.Err() != nil {
				TrailarrLog(INFO, "Tasks", "Extras download cancelled before processing extra.")
				break
			}
			typ := canonicalizeExtraType(extra.Type, extra.Type)
			if !isExtraTypeEnabled(cfg, typ) {
				continue
			}
			if extra.Status == "rejected" {
				continue
			}
			if extra.Status == "missing" && extra.YoutubeId != "" {
				err := handleTypeFilteredExtraDownload(mediaType, mediaId, extra)
				if err != nil {
					TrailarrLog(WARN, "Tasks", "[SEQ] Download failed: %v", err)
				}
			}
		}
	}
}

// Handles downloading a single extra and appending to history if successful
func handleTypeFilteredExtraDownload(mediaType MediaType, mediaId int, extra Extra) error {
	_, err := DownloadYouTubeExtra(mediaType, mediaId, extra.Type, extra.Title, extra.YoutubeId)
	if err != nil {
		TrailarrLog(WARN, "Tasks", "DownloadYouTubeExtra failed: %v", err)
		return err
	}
	// Add to history if download succeeded
	AppendHistoryEvent(HistoryEvent{
		Action: "download",

		Title:      getMediaTitleFromCache(mediaType, mediaId),
		MediaType:  mediaType,
		MediaId:    mediaId,
		ExtraType:  extra.Type,
		ExtraTitle: extra.Title,
		Date:       time.Now(),
	})
	return nil
}

// Helper to get media title from cache
func getMediaTitleFromCache(mediaType MediaType, mediaId int) string {
	cacheFile, _ := resolveCachePath(mediaType)
	if cacheFile != "" {
		items, _ := loadCache(cacheFile)
		for _, m := range items {
			idInt, ok := parseMediaID(m["id"])
			if ok && idInt == mediaId {
				if t, ok := m["title"].(string); ok {
					return t
				}
			}
		}
	}
	return ""
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
		// Add new queue item to file on start
		queued := time.Now()
		item := SyncQueueItem{
			TaskId:  string(taskId),
			Queued:  queued,
			Status:  "running",
			Started: queued,
		}
		var fileQueue []SyncQueueItem
		_ = ReadJSONFile(QueueFile, &fileQueue)
		fileQueue = append(fileQueue, item)
		_ = WriteJSONFile(QueueFile, fileQueue)

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
		// Update the last queue item for this task (by TaskId and Queued)
		_ = ReadJSONFile(QueueFile, &fileQueue)
		for i := len(fileQueue) - 1; i >= 0; i-- {
			if fileQueue[i].TaskId == string(taskId) && fileQueue[i].Queued.Equal(queued) {
				fileQueue[i].Status = status
				fileQueue[i].Started = queued
				fileQueue[i].Ended = ended
				fileQueue[i].Duration = duration
				if err != nil {
					fileQueue[i].Error = err.Error()
				} else {
					fileQueue[i].Error = ""
				}
				break
			}
		}
		_ = WriteJSONFile(QueueFile, fileQueue)
	}
}
