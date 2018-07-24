package log

import (
	"fmt"
	"os"
)

// this is an dummy log impl

func Trace(msg string, ctx ...interface{}) {
	fmt.Println(msg, ctx)
}

func Debug(msg string, ctx ...interface{}) {
	fmt.Println(msg, ctx)
}

func Info(msg string, ctx ...interface{}) {
	fmt.Println(msg, ctx)
}

func Warn(msg string, ctx ...interface{}) {
	fmt.Println(msg, ctx)
}

func Error(msg string, ctx ...interface{}) {
	fmt.Println(msg, ctx)
}

func Fatal(msg string, ctx ...interface{}) {
	fmt.Println(msg, ctx)
	os.Exit(1)
}

func Crit(msg string, ctx ...interface{}) {
	fmt.Println(msg, ctx)
	os.Exit(1)
}
