package internal

import (
	"context"
	"encoding/json"
	"net/http"
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

func historyHandler(c *gin.Context) {
	events, err := LoadHistoryEvents()
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	// events are stored newest-last, reverse for newest-first
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}
	respondJSON(c, http.StatusOK, gin.H{"history": events})
}

func AppendHistoryEvent(event HistoryEvent) error {
	client := GetStoreClient()
	ctx := context.Background()
	b, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if err := client.RPush(ctx, HistoryStoreKey, b); err != nil {
		return err
	}
	// Trim list to maximum length
	return client.LTrim(ctx, HistoryStoreKey, -HistoryMaxLen, -1)
}

func LoadHistoryEvents() ([]HistoryEvent, error) {
	client := GetStoreClient()
	ctx := context.Background()
	vals, err := client.LRange(ctx, HistoryStoreKey, 0, -1)
	if err != nil {
		return nil, err
	}
	events := make([]HistoryEvent, 0, len(vals))
	for _, v := range vals {
		var e HistoryEvent
		if err := json.Unmarshal([]byte(v), &e); err == nil {
			events = append(events, e)
		}
	}
	return events, nil
}
