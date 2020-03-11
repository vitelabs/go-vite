package govite

import "fmt"

// For "govendor install"

func PrintBuildVersion() {
	if VITE_VERSION != "" {
		fmt.Println("this vite node`s git GO version is ", VITE_VERSION)
	} else {
		fmt.Println("can not read gitversion file please use Make to build Vite ")
	}
}
