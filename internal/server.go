package internal

import (
	"github.com/gin-gonic/gin"
)

// ListServerFoldersHandler handles GET /api/files/list and returns subfolders for a given path
func ListServerFoldersHandler(c *gin.Context) {
	// Only allow browsing from allowed roots
	allowedRoots := []string{"/mnt", TrailarrRoot}
	reqPath := c.Query("path")
	if reqPath == "" {
		// If no path, return allowed roots
		respondJSON(c, 200, gin.H{"folders": allowedRoots})
		return
	}
	// Security: ensure reqPath is under allowed roots
	valid := false
	for _, root := range allowedRoots {
		if reqPath == root || (len(reqPath) > len(root) && reqPath[:len(root)] == root) {
			valid = true
			break
		}
	}
	if !valid {
		respondError(c, 400, "Invalid path")
		return
	}
	folders, err := ListSubdirectories(reqPath)
	if err != nil {
		respondError(c, 500, err.Error())
		return
	}
	respondJSON(c, 200, gin.H{"folders": folders})
}
