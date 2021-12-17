package common

import (
	"io"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/vitelabs/go-vite/v2/log15"
)

func makeDefaultLogger(absFilePath string) io.Writer {
	return &lumberjack.Logger{
		Filename:   absFilePath,
		MaxSize:    100,
		MaxBackups: 14,
		MaxAge:     14,
		Compress:   true,
		LocalTime:  true,
	}
}

func LogHandler(path, subDir, filename, lvl string) log15.Handler {
	logLevel, err := log15.LvlFromString(lvl)
	if err != nil {
		logLevel = log15.LvlInfo
	}
	absFilename := filepath.Join(path, subDir, filename)
	out := makeDefaultLogger(absFilename)
	return log15.LvlFilterHandler(logLevel, log15.StreamHandler(out, log15.LogfmtFormat()))
}
