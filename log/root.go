package log

import (
	"fmt"
	"os"
	"time"
)

// this is an dummy log impl

func Trace(msg string, ctx ...interface{}) {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05")+"[T]"+msg, ctx)
}

func Debug(msg string, ctx ...interface{}) {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05")+"[D]"+msg, ctx)
}

func Info(msg string, ctx ...interface{}) {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05")+"[I]"+msg, ctx)
}

func Warn(msg string, ctx ...interface{}) {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05")+"[W]"+msg, ctx)
}

func Error(msg string, ctx ...interface{}) {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05")+"[E]"+msg, ctx)
}

func Fatal(msg string, ctx ...interface{}) {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05")+"[F]"+msg, ctx)
	os.Exit(1)
}

func Crit(msg string, ctx ...interface{}) {
	fmt.Println(time.Now().Format("2006-01-02 15:04:05")+"[C]"+msg, ctx)
	os.Exit(1)
}
