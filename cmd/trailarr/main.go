package main

import (
	"os"
	"trailarr/internal"

	"github.com/gin-gonic/gin"
)

var timings map[string]int

func main() {
	// Only log backend/server logs to file. Gin (frontend HTTP) logs go to stdout only.
	logDir := internal.TrailarrRoot + "/logs"
	logFile := logDir + "/trailarr.txt"
	_ = os.MkdirAll(logDir, 0775)
	internal.InitTrailarrLogWriter(logFile)
	gin.DefaultWriter = os.Stdout
	gin.DefaultErrorWriter = os.Stderr

	var err error
	timings, err = internal.EnsureSyncTimingsConfig()
	if err != nil {
		internal.TrailarrLog("Warn", "Startup", "Could not load sync timings: %v", err)
	}
	internal.Timings = timings
	internal.TrailarrLog("Info", "Startup", "Sync timings: %v", timings)
	r := gin.Default()
	internal.RegisterRoutes(r)
	internal.StartBackgroundTasks()
	r.Run(":8080")
}
