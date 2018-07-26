package log

import (
	"fmt"
	"os"
)

// this is an dummy log impl

func Trace(msg string, ctx ...interface{}) {
	fmt.Println("[T]"+msg, ctx)
}

func Debug(msg string, ctx ...interface{}) {
	fmt.Println("[D]"+msg, ctx)
}

func Info(msg string, ctx ...interface{}) {
	fmt.Println("[I]"+msg, ctx)
}

func Warn(msg string, ctx ...interface{}) {
	fmt.Println("[W]"+msg, ctx)
}

func Error(msg string, ctx ...interface{}) {
	fmt.Println("[E]"+msg, ctx)
}

func Fatal(msg string, ctx ...interface{}) {
	fmt.Println("[F]"+msg, ctx)
	os.Exit(1)
}

func Crit(msg string, ctx ...interface{}) {
	fmt.Println("[C]"+msg, ctx)
	os.Exit(1)
}
