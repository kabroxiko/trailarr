package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetMediaHandlerListAndFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmp := t.TempDir()
	cache := filepath.Join(tmp, "cache.json")
	items := []map[string]interface{}{{"id": 1, "title": "A"}, {"id": 2, "title": "B"}}
	if err := WriteJSONFile(cache, items); err != nil {
		t.Fatalf("failed to write cache: %v", err)
	}

	// GET all
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/?", nil)
	c.Request = req
	handler := GetMediaHandler(cache, "id")
	handler(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string][]map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if len(resp["items"]) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp["items"]))
	}

	// GET with id filter
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	req2 := httptest.NewRequest("GET", "/?id=2", nil)
	c2.Request = req2
	handler(c2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200 on filtered, got %d", w2.Code)
	}
	var resp2 map[string][]map[string]interface{}
	if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("invalid json response filtered: %v", err)
	}
	if len(resp2["items"]) != 1 {
		t.Fatalf("expected 1 item after filter, got %d", len(resp2["items"]))
	}
}

func TestGetMediaByIdHandlerFoundAndNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmp := t.TempDir()
	cache := filepath.Join(tmp, "cache2.json")
	items := []map[string]interface{}{{"id": 10, "title": "X"}}
	if err := WriteJSONFile(cache, items); err != nil {
		t.Fatalf("failed to write cache: %v", err)
	}

	handler := GetMediaByIdHandler(cache, "id")

	// found
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/items/10", nil)
	c.Request = req
	c.Params = gin.Params{gin.Param{Key: "id", Value: "10"}}
	handler(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for found item, got %d", w.Code)
	}

	// not found
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	req2 := httptest.NewRequest("GET", "/items/20", nil)
	c2.Request = req2
	c2.Params = gin.Params{gin.Param{Key: "id", Value: "20"}}
	handler(c2)
	if w2.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing item, got %d", w2.Code)
	}
}
