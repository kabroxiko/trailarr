package internal

import (
	"os"
	"testing"
	"time"
)

// TestMain sets environment needed for tests.
// It ensures embedded Redis is skipped during the test run; tests that need Redis
// should rely on a running Redis instance or mock the interface.
func TestMain(m *testing.M) {
	// Use the fake runner for yt-dlp to avoid launching external processes in tests
	oldRunner := ytDlpRunner
	ytDlpRunner = &fakeRunner{}
	defer func() { ytDlpRunner = oldRunner }()

	// Shorten queue-related delays for faster tests
	QueueItemRemoveDelay = 10 * time.Millisecond
	QueuePollInterval = 10 * time.Millisecond
	// Shorten other package-level sleeps so tests run quickly
	DownloadQueueWatcherInterval = 5 * time.Millisecond
	TooManyRequestsPauseDuration = 100 * time.Millisecond
	TooManyRequestsPauseLogInterval = 10 * time.Millisecond
	TasksDepsWaitInterval = 10 * time.Millisecond
	TasksInitialDelay = 10 * time.Millisecond
	os.Exit(m.Run())
}
