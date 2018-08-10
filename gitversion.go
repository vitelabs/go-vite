package govite

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
)

func PrintBuildVersion() {
	_, filename, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(filename), "gitversion")
	bytes, e := ioutil.ReadFile(path)
	gitversion := string(bytes)
	if e == nil || gitversion != "" {
		fmt.Println("this vite node`s git version is ", gitversion)
	} else {
		println("can not read gitversion file please use Make to build Vite ")
	}
}
