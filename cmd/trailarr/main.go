package main

import (
	"fmt"

	"trailarr/internal"

	"github.com/gin-gonic/gin"
)

var timings map[string]int

func main() {
	var err error
	timings, err = internal.EnsureSyncTimingsConfig()
	if err != nil {
		fmt.Printf("[WARN] Could not load sync timings: %v\n", err)
	}
	internal.Timings = timings
	fmt.Printf("[INFO] Sync timings: %v\n", timings)
	r := gin.Default()
	internal.RegisterRoutes(r)
	internal.StartBackgroundTasks()
	r.Run(":8080")
}
