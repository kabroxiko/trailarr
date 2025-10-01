package internal

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	r.GET("/api/extras/search", searchExtrasHandler)
	r.POST("/api/extras/download", downloadExtraHandler)
	r.GET("/api/plex", plexItemsHandler)
}

// Handler for Plex items
func plexItemsHandler(c *gin.Context) {
	items, err := FetchPlexLibrary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func searchExtrasHandler(c *gin.Context) {
	movie := c.Query("movie")
	results, _ := SearchExtras(movie)
	c.JSON(http.StatusOK, gin.H{"extras": results})
}

func downloadExtraHandler(c *gin.Context) {
	var req struct {
		URL string `json:"url"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	_ = DownloadExtra(req.URL)
	c.JSON(http.StatusOK, gin.H{"status": "downloading"})
}
