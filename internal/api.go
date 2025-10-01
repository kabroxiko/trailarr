package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all API endpoints to the Gin router
func RegisterRoutes(r *gin.Engine) {
	// Serve static files for movie posters
	r.Static("/mediacover", "/mnt/unionfs/extrazarr/MediaCover")
	r.GET("/api/radarr/movies", getRadarrMoviesHandler)
	r.POST("/api/settings/radarr", saveRadarrSettingsHandler)
	r.GET("/api/settings/radarr", getRadarrSettingsHandler)
	r.GET("/api/extras/search", searchExtrasHandler)
	r.POST("/api/extras/download", downloadExtraHandler)
	r.GET("/api/plex", plexItemsHandler)
}

// Handler to fetch movies from Radarr
func getRadarrMoviesHandler(c *gin.Context) {
	// Serve movies from cache (only movies with downloaded posters)
	cachePath := "/mnt/unionfs/extrazarr/movies_cache.json"
	cacheData, err := os.ReadFile(cachePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Movie cache not found"})
		return
	}
	var movies []map[string]interface{}
	if err := json.Unmarshal(cacheData, &movies); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode movie cache"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"movies": movies})
}

// Handler to get Radarr settings
func getRadarrSettingsHandler(c *gin.Context) {
	data, err := os.ReadFile("radarr.json")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"url": "", "apiKey": ""})
		return
	}
	var settings struct {
		URL    string `json:"url"`
		APIKey string `json:"apiKey"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		c.JSON(http.StatusOK, gin.H{"url": "", "apiKey": ""})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": settings.URL, "apiKey": settings.APIKey})
}

// Handler to save Radarr settings
func saveRadarrSettingsHandler(c *gin.Context) {
	var req struct {
		URL    string `json:"url"`
		APIKey string `json:"apiKey"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	// Save to a config file (radarr.json)
	data := []byte(fmt.Sprintf(`{"url": "%s", "apiKey": "%s"}`, req.URL, req.APIKey))
	err := os.WriteFile("radarr.json", data, 0644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "saved"})
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

// SyncRadarrMoviesAndMediaCover syncs Radarr movie list and MediaCover folder
func SyncRadarrMoviesAndMediaCover() {
	// Load Radarr settings
	data, err := os.ReadFile("radarr.json")
	if err != nil {
		fmt.Println("[Sync] Radarr settings not found")
		return
	}
	var settings struct {
		URL    string `json:"url"`
		APIKey string `json:"apiKey"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		fmt.Println("[Sync] Invalid Radarr settings")
		return
	}
	// Read movies from cache file
	cachePath := "/mnt/unionfs/extrazarr/movies_cache.json"
	cacheData, err := os.ReadFile(cachePath)
	var movies []map[string]interface{}
	if err != nil {
		fmt.Println("[Sync] Movie cache not found, fetching from Radarr:", err)
		// Fetch movies from Radarr
		req, err := http.NewRequest("GET", settings.URL+"/api/v3/movie", nil)
		if err != nil {
			fmt.Println("[Sync] Error creating request:", err)
			return
		}
		req.Header.Set("X-Api-Key", settings.APIKey)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("[Sync] Error fetching movies:", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			fmt.Println("[Sync] Radarr API error:", resp.StatusCode)
			return
		}
		var allMovies []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&allMovies); err != nil {
			fmt.Println("[Sync] Failed to decode Radarr response:", err)
			return
		}
		// Filter only downloaded movies (hasFile == true)
		movies = make([]map[string]interface{}, 0)
		for _, m := range allMovies {
			if hasFile, ok := m["hasFile"].(bool); ok && hasFile {
				movies = append(movies, m)
			}
		}
		// Save movies to cache file
		cacheData, _ = json.MarshalIndent(movies, "", "  ")
		_ = os.WriteFile(cachePath, cacheData, 0644)
		fmt.Println("[Sync] Synced", len(movies), "downloaded movies to cache.")
	} else {
		if err := json.Unmarshal(cacheData, &movies); err != nil {
			fmt.Println("[Sync] Failed to decode movie cache:", err)
			return
		}
		fmt.Println("[Sync] Loaded", len(movies), "movies from cache.")
	}

	// Download poster images from Radarr and cache only movies with posters
	client := &http.Client{}
	downloadedMovies := make([]map[string]interface{}, 0)
	for _, movie := range movies {
		hasFile, ok := movie["hasFile"].(bool)
		if !ok || !hasFile {
			continue
		}
		id, ok := movie["id"].(float64)
		if !ok {
			continue
		}
		idStr := fmt.Sprintf("%d", int(id))
		posterUrl := fmt.Sprintf("%s/MediaCover/%s/poster-500.jpg", settings.URL, idStr)
		localPath := fmt.Sprintf("/mnt/unionfs/extrazarr/MediaCover/%s/poster-500.jpg", idStr)

		os.MkdirAll(fmt.Sprintf("/mnt/unionfs/extrazarr/MediaCover/%s", idStr), 0755)

		resp, err := client.Get(posterUrl)
		if err != nil {
			fmt.Println("[Sync] Failed to download poster for movie", idStr, err)
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			out, err := os.Create(localPath)
			if err == nil {
				_, err = io.Copy(out, resp.Body)
				out.Close()
				if err == nil {
					// Only add movie to cache if poster was saved successfully and hasFile == true
					downloadedMovies = append(downloadedMovies, movie)
				} else {
					fmt.Println("[Sync] Error saving poster for movie", idStr, err)
				}
			} else {
				fmt.Println("[Sync] Error creating file for poster", localPath, err)
			}
		} else {
			fmt.Println("[Sync] Poster not found for movie", idStr)
		}
	}

	// Save only movies with downloaded posters to cache
	cachePath = "/mnt/unionfs/extrazarr/movies_cache.json"
	cacheData, _ = json.MarshalIndent(downloadedMovies, "", "  ")
	_ = os.WriteFile(cachePath, cacheData, 0644)
	fmt.Println("[Sync] Cached", len(downloadedMovies), "movies with posters.")
}
