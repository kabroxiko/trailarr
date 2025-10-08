package internal

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetTaskQueueFileHandler returns the queue directly from the file, not memory
func GetTaskQueueFileHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var fileQueue []SyncQueueItem
		err := ReadJSONFile(QueueFile, &fileQueue)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read queue file", "detail": err.Error()})
			return
		}
		queues := make([]map[string]interface{}, 0, len(fileQueue))
		for _, item := range fileQueue {
			queues = append(queues, NewQueueStatusMap(item))
		}
		sortTaskQueuesByQueuedDesc(queues)
		// Show only the first 12 records (most recent)
		if len(queues) > 12 {
			queues = queues[:12]
		}
		c.JSON(http.StatusOK, gin.H{"queues": queues})
	}
}
