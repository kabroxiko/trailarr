package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// Handler to list existing extras for a movie path
func existingExtrasHandler(c *gin.Context) {
	moviePath := c.Query("moviePath")
	if moviePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "moviePath required"})
		return
	}
	// Scan subfolders for .mp4 files and their metadata
	var existing []map[string]interface{}
	entries, err := os.ReadDir(moviePath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"existing": []map[string]interface{}{}})
		return
	}
	// Track duplicate index for each type/title
	dupCount := make(map[string]int)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subdir := moviePath + "/" + entry.Name()
		files, _ := os.ReadDir(subdir)
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".mp4") {
				metaFile := subdir + "/" + strings.TrimSuffix(f.Name(), ".mp4") + ".mp4.json"
				var meta struct {
					Type      string `json:"type"`
					Title     string `json:"title"`
					YouTubeID string `json:"youtube_id"`
				}
				if metaBytes, err := os.ReadFile(metaFile); err == nil {
					_ = json.Unmarshal(metaBytes, &meta)
				}
				key := entry.Name() + "|" + meta.Title
				dupCount[key]++
				existing = append(existing, map[string]interface{}{
					"type":       entry.Name(),
					"title":      meta.Title,
					"youtube_id": meta.YouTubeID,
					"_dupIndex":  dupCount[key],
				})
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"existing": existing})
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
	mediaType := c.Query("mediaType")
	idStr := c.Query("id")
	var id int
	fmt.Sscanf(idStr, "%d", &id)
	results, _ := SearchExtras(mediaType, id)
	c.JSON(http.StatusOK, gin.H{"extras": results})
}

func downloadExtraHandler(c *gin.Context) {
	var req struct {
		MoviePath  string `json:"moviePath"`
		ExtraType  string `json:"extraType"`
		ExtraTitle string `json:"extraTitle"`
		URL        string `json:"url"`
	}
	if err := c.BindJSON(&req); err != nil {
		fmt.Printf("[downloadExtraHandler] Invalid request: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRequest})
		return
	}
	fmt.Printf("[downloadExtraHandler] Download request: moviePath=%s, extraType=%s, extraTitle=%s, url=%s\n", req.MoviePath, req.ExtraType, req.ExtraTitle, req.URL)
	meta, err := DownloadYouTubeExtra(req.MoviePath, req.ExtraType, req.ExtraTitle, req.URL)
	if err != nil {
		fmt.Printf("[downloadExtraHandler] Download error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "downloaded", "meta": meta})
}
