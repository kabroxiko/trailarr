package internal

import (
	"encoding/json"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type HistoryEvent struct {
	Action     string    `json:"action"`
	Title      string    `json:"title"`
	MediaType  MediaType `json:"mediaType"`
	ExtraType  string    `json:"extraType"`
	ExtraTitle string    `json:"extraTitle"`
	Date       time.Time `json:"date"`
}

var historyMutex sync.Mutex

func historyHandler(c *gin.Context) {
	events := LoadHistoryEvents()
	c.JSON(http.StatusOK, gin.H{"history": events})
}

var historyFile = TrailarrRoot + "/history.json"

func AppendHistoryEvent(event HistoryEvent) error {
	historyMutex.Lock()
	defer historyMutex.Unlock()
	events := LoadHistoryEvents()
	events = append(events, event)
	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(historyFile, data, 0644)
}

func LoadHistoryEvents() []HistoryEvent {
	var events []HistoryEvent
	data, err := os.ReadFile(historyFile)
	if err == nil {
		_ = json.Unmarshal(data, &events)
	}
	return events
}
