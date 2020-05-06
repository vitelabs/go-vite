package log

import (
	"log"
	"math/rand"
	"os"
	"os/user"
	"path"
	"strconv"
	"time"
)

func InitPath() {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}

	rand.Seed(time.Now().Unix())

	dir := path.Join(usr.HomeDir, "go_vite", "log")
	os.MkdirAll(dir, os.ModePerm)
	fileName := path.Join(dir, "testlogfile."+strconv.Itoa(rand.Intn(10000)))
	f, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	} else {
		log.Printf("log file:%s\n", fileName)
	}
	log.SetOutput(f)
}

func Debug(msg string, ctx ...interface{}) {
	log.Printf("[Debug]"+msg+"\n", ctx...)
}

func Info(msg string, ctx ...interface{}) {
	log.Printf("[Info]"+msg+"\n", ctx...)
}

func Warn(msg string, ctx ...interface{}) {
	log.Printf("[Warn]"+msg+"\n", ctx...)
}

func Error(msg string, ctx ...interface{}) {
	log.Printf("[Error]"+msg+"\n", ctx...)

}

func Fatal(msg string, ctx ...interface{}) {
	log.Printf("[Fatal]"+msg+"\n", ctx...)
	os.Exit(1)
}
