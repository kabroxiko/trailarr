package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestListServerFoldersHandlerRootsAndInvalid(t *testing.T) {
	// Setup a router with the handler
	r := gin.New()
	r.GET("/api/files/list", ListServerFoldersHandler)

	// Request without path should return allowed roots
	req := httptest.NewRequest(http.MethodGet, "/api/files/list", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 when listing roots, got %d", w.Code)
	}
	var resp map[string][]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse roots response: %v", err)
	}
	if _, ok := resp["folders"]; !ok {
		t.Fatalf("expected folders key in response: %v", resp)
	}

	// Invalid path outside allowed roots should return 400
	req2 := httptest.NewRequest(http.MethodGet, "/api/files/list?path=/etc/passwd", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != 400 {
		t.Fatalf("expected 400 for invalid path, got %d", w2.Code)
	}
}

func TestCalcNextAndBuildSchedules(t *testing.T) {
	// Prepare a fake tasksMeta and timings
	// Keep original values to restore
	origTasksMeta := tasksMeta
	origTimings := Timings
	defer func() { tasksMeta = origTasksMeta; Timings = origTimings }()

	tasksMeta = map[TaskID]TaskMeta{
		"a": {ID: "a", Name: "A", Order: 2},
		"b": {ID: "b", Name: "B", Order: 1},
	}
	Timings = map[string]int{"a": 10, "b": 5}

	// Case 1: LastExecution zero should produce NextExecution ~ now + interval
	states := make(TaskStates)
	states["a"] = TaskState{ID: "a", LastExecution: time.Time{}, LastDuration: 0}
	states["b"] = TaskState{ID: "b", LastExecution: time.Now(), LastDuration: 1}

	schedules := buildSchedules(states)
	if len(schedules) != 2 {
		t.Fatalf("expected 2 schedules, got %d", len(schedules))
	}
	// Because order sorts by TaskMeta.Order, first should be B then A
	if schedules[0].Name != "B" || schedules[1].Name != "A" {
		t.Fatalf("unexpected order in schedules: %+v", schedules)
	}

	// Check NextExecution roughly matches calcNext behavior
	for _, s := range schedules {
		if s.LastExecution.IsZero() {
			// NextExecution should be approx now + interval
			expected := time.Now().Add(time.Duration(Timings[string(s.TaskID)]) * time.Minute)
			if s.NextExecution.Before(time.Now()) || s.NextExecution.After(expected.Add(2*time.Second)) {
				t.Fatalf("NextExecution for zero LastExecution out of expected range: got %v, expected around %v", s.NextExecution, expected)
			}
		}
	}
}
