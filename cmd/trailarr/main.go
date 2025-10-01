package main

import (
	"os"
	"log"
	"io"
	"trailarr/internal"
	"github.com/gin-gonic/gin"
	"gopkg.in/natefinch/lumberjack.v2"
)

var timings map[string]int

func main() {
	// Setup log rotation
	logDir := internal.TrailarrRoot + "/logs"
	logFile := logDir + "/trailarr.txt"
	_ = os.MkdirAll(logDir, 0775)
	lumberjackLogger := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    1,    // megabytes
		MaxBackups: 50,
		MaxAge:     0,    // days, 0 means keep forever
		Compress:   false,
	}
	mw := io.MultiWriter(os.Stdout, lumberjackLogger)
	log.SetOutput(mw)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// Redirect fmt.Print and Gin logs as well
	gin.DefaultWriter = mw
	gin.DefaultErrorWriter = mw
	// Note: Cannot assign lumberjackLogger to os.Stdout/os.Stderr (not *os.File), so fmt.Print will still go to the original stdout/stderr.

	var err error
	timings, err = internal.EnsureSyncTimingsConfig()
	if err != nil {
		log.Printf("[WARN] Could not load sync timings: %v\n", err)
	}
	internal.Timings = timings
	log.Printf("[INFO] Sync timings: %v\n", timings)
	r := gin.Default()
	internal.RegisterRoutes(r)
	internal.StartBackgroundTasks()
	r.Run(":8080")
}
