package internal

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetTaskQueueFileHandler returns the queue directly from the file, not memory
func GetTaskQueueFileHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var fileQueue []TaskStatus
		err := ReadJSONFile(QueueFile, &fileQueue)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read queue file", "detail": err.Error()})
			return
		}
		sortTaskQueuesByQueuedDesc(fileQueue)
		queues := fileQueue
		// Show only the first 100 records (most recent)
		if len(queues) > 100 {
			queues = queues[:100]
		}
		c.JSON(http.StatusOK, gin.H{"queues": queues})
	}
}
