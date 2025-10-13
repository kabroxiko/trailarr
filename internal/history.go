package internal

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type HistoryEvent struct {
	Action     string    `json:"action"`
	MediaTitle string    `json:"mediaTitle"`
	MediaType  MediaType `json:"mediaType"`
	MediaId    int       `json:"mediaId"`
	ExtraType  string    `json:"extraType"`
	ExtraTitle string    `json:"extraTitle"`
	Date       time.Time `json:"date"`
}

var historyMutex sync.Mutex

func historyHandler(c *gin.Context) {
	events := LoadHistoryEvents()
	// Reverse events so newest is first
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}
	respondJSON(c, http.StatusOK, gin.H{"history": events})
}

var historyFile = HistoryFile

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
