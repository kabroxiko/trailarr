package internal

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetTaskQueueFileHandler returns the queue directly from the file, not memory
func GetTaskQueueFileHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Load from store
		client := GetStoreClient()
		ctx := context.Background()
		vals, err := client.LRange(ctx, TaskQueueStoreKey, 0, -1)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read queue from bbolt", "detail": err.Error()})
			return
		}
		queues := make([]TaskStatus, 0, len(vals))
		for _, v := range vals {
			var qi SyncQueueItem
			if err := json.Unmarshal([]byte(v), &qi); err != nil {
				continue
			}
			queues = append(queues, TaskStatus{
				TaskId:   qi.TaskId,
				Queued:   qi.Queued,
				Started:  qi.Started,
				Ended:    qi.Ended,
				Duration: qi.Duration.Seconds(),
				Status:   qi.Status,
				Error:    qi.Error,
			})
		}
		sortTaskQueuesByQueuedDesc(queues)
		if len(queues) > 100 {
			queues = queues[:100]
		}
		c.JSON(http.StatusOK, gin.H{"queues": queues})
	}
}

// GetTaskQueueDebugHandler returns raw debug information about the stored task
// queue (count and raw serialized entries). This is useful for debugging
// whether items are being persisted/truncated in the underlying store.
func GetTaskQueueDebugHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		client := GetStoreClient()
		ctx := context.Background()
		vals, err := client.LRange(ctx, TaskQueueStoreKey, 0, -1)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read queue from store", "detail": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"count": len(vals),
			"raw":   vals,
		})
	}
}
