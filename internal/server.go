package internal

import (
	"os"
	"path/filepath"

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
	// List subfolders
	entries, err := os.ReadDir(reqPath)
	if cerr := CheckErrLog(ERROR, "Server", "ReadDir failed", err); cerr != nil {
		respondError(c, 500, cerr.Error())
		return
	}
	var folders []string
	for _, entry := range entries {
		if entry.IsDir() {
			folders = append(folders, filepath.Join(reqPath, entry.Name()))
		}
	}
	respondJSON(c, 200, gin.H{"folders": folders})
}
