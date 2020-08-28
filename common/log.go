package common

import (
	"path/filepath"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/vitelabs/go-vite/log15"
)

func makeDefaultLogger(absFilePath string) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   absFilePath,
		MaxSize:    100,
		MaxBackups: 14,
		MaxAge:     14,
		Compress:   true,
		LocalTime:  true,
	}
}

func LogHandler(baseDir string, subDir string, filename string, lvl string) log15.Handler {
	logLevel, err := log15.LvlFromString(lvl)
	if err != nil {
		logLevel = log15.LvlInfo
	}
	path := filepath.Join(baseDir, subDir, time.Now().Format("2006-01-02T15-04"))

	absFilename := filepath.Join(path, filename)
	return log15.LvlFilterHandler(logLevel, log15.StreamHandler(makeDefaultLogger(absFilename), log15.LogfmtFormat()))
}
