package internal

import (
	"fmt"
	"os"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logWriter     *lumberjack.Logger
	logWriterOnce sync.Once
)

// Call this once at startup to set up the log file writer
func InitTrailarrLogWriter(logPath string) {
	logWriterOnce.Do(func() {
		logWriter = &lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    1, // megabytes
			MaxBackups: 50,
			MaxAge:     0, // days, 0 means keep forever
			Compress:   false,
		}
	})
}

// LogLevel can be Info, Warn, Error, Debug, etc.
func TrailarrLog(level, component, message string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05.0")
	msg := fmt.Sprintf(message, args...)
	logLine := fmt.Sprintf("%s|%s|%s|%s\n", timestamp, level, component, msg)
	fmt.Fprint(os.Stdout, logLine)
	if logWriter != nil {
		logWriter.Write([]byte(logLine))
	}
}

// CheckErrLog logs the error with context and returns it (for propagation)
func CheckErrLog(level, component, context string, err error) error {
	if err != nil {
		TrailarrLog(level, component, "%s: %v", context, err)
	}
	return err
}
