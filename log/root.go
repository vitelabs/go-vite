package log

import (
	"os"
)

// this is an dummy log impl

func Trace(msg string, ctx ...interface{}) {
	println(msg, ctx)
}

func Debug(msg string, ctx ...interface{}) {
	println(msg, ctx)
}

func Info(msg string, ctx ...interface{}) {
	println(msg, ctx)
}

func Warn(msg string, ctx ...interface{}) {
	println(msg, ctx)
}

func Error(msg string, ctx ...interface{}) {
	println(msg, ctx)
}

func Crit(msg string, ctx ...interface{}) {
	println(msg, ctx)
	os.Exit(1)
}
