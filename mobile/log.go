package mobile

import (
	"github.com/vitelabs/go-vite/log15"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"path/filepath"
)

var mobileLog = log15.Root().New()

func InitLog(dir string, needDebug bool) error {
	filename := "vite.log"
	{
		errAbsFilePath := filepath.Join(dir, "glog", "error")
		if err := os.MkdirAll(errAbsFilePath, 0555); err != nil {
			return err
		}
		infoAbsFilePath := filepath.Join(dir, "glog")
		if err := os.MkdirAll(infoAbsFilePath, 0555); err != nil {
			return err
		}
		log15.Root().SetHandler(getLogLvlFilter(needDebug, filepath.Join(errAbsFilePath, filename),
			filepath.Join(errAbsFilePath, filename)))
	}

	{
		errAbsFilePath := filepath.Join(dir, "mlog", "error")
		if err := os.MkdirAll(errAbsFilePath, 0555); err != nil {
			return err
		}
		infoAbsFilePath := filepath.Join(dir, "mlog")
		if err := os.MkdirAll(infoAbsFilePath, 0555); err != nil {
			return err
		}
		mobileLog.SetHandler(getLogLvlFilter(needDebug, filepath.Join(errAbsFilePath, filename),
			filepath.Join(errAbsFilePath, filename)))
	}

	return nil
}

func makeDefaultLogger(absFilePath string) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   absFilePath,
		MaxSize:    1,
		MaxBackups: 10,
		MaxAge:     7,
		Compress:   true,
		LocalTime:  true,
	}
}

func getLogLvlFilter(needDebug bool, infoAbsFilePath, errAbsFilePath string) log15.Handler {
	return log15.MultiHandler(
		log15.FilterHandler(func(r *log15.Record) (pass bool) {
			return needDebug
		}, log15.StdoutHandler),

		log15.FilterHandler(func(r *log15.Record) (pass bool) {
			maxLevel := log15.LvlInfo
			if needDebug {
				maxLevel = log15.LvlDebug
			}
			return log15.LvlWarn < r.Lvl || r.Lvl <= maxLevel
		}, log15.StreamHandler(makeDefaultLogger(infoAbsFilePath), log15.LogfmtFormat())),

		log15.FilterHandler(func(r *log15.Record) (pass bool) {
			return r.Lvl <= log15.LvlError
		}, log15.StreamHandler(makeDefaultLogger(errAbsFilePath), log15.LogfmtFormat())),
	)
}

func LogD(msg string) {
	mobileLog.Debug(msg)
}

func LogI(msg string) {
	mobileLog.Info(msg)
}

func LogW(msg string) {
	mobileLog.Warn(msg)
}

func LogE(msg string) {
	mobileLog.Error(msg)
}
