package logger

import (
	"log"
	"os"
	"sync"
)

type Level string

const (
	LevelError Level = "ERROR"
	LevelWarn  Level = "WARN"
	LevelInfo  Level = "INFO"
)

var (
	mu     sync.Mutex
	stdLog *log.Logger
)

func ensureLogger() {
	if stdLog != nil {
		return
	}

	f, err := os.OpenFile("local.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		stdLog = log.New(os.Stderr, "", log.LstdFlags|log.Lmicroseconds)
		return
	}
	stdLog = log.New(f, "", log.LstdFlags|log.Lmicroseconds)
}

func logf(level Level, component, format string, args ...any) {
	mu.Lock()
	defer mu.Unlock()

	if stdLog == nil {
		ensureLogger()
	}

	prefix := "[" + string(level) + "][" + component + "] "
	stdLog.Printf(prefix+format, args...)
}

func Error(component, message string, err error) {
	if err != nil {
		logf(LevelError, component, "%s: %v", message, err)
		return
	}
	logf(LevelError, component, "%s", message)
}

func Warn(component, message string) {
	logf(LevelWarn, component, "%s", message)
}

func Info(component, message string) {
	logf(LevelInfo, component, "%s", message)
}
