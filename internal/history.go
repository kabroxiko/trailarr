package internal

import (
	"net/http"
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
	respondJSON(c, http.StatusOK, gin.H{"history": events})
}

var historyFile = TrailarrRoot + "/history.json"

func AppendHistoryEvent(event HistoryEvent) error {
	historyMutex.Lock()
	defer historyMutex.Unlock()
	events := LoadHistoryEvents()
	events = append(events, event)
	return WriteJSONFile(historyFile, events)
}

func LoadHistoryEvents() []HistoryEvent {
	var events []HistoryEvent
	_ = ReadJSONFile(historyFile, &events)
	return events
}
