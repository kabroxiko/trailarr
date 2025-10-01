package internal

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

type HistoryEvent struct {
	Action     string    `json:"action"`
	Title      string    `json:"title"`
	MediaType  string    `json:"mediaType"`
	ExtraType  string    `json:"extraType"`
	ExtraTitle string    `json:"extraTitle"`
	Date       time.Time `json:"date"`
}

var historyMutex sync.Mutex
var historyFile = "/var/lib/extrazarr/history.json"

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
