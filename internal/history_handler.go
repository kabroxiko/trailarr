package internal

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func historyHandler(c *gin.Context) {
	events := LoadHistoryEvents()
	c.JSON(http.StatusOK, gin.H{"history": events})
}
