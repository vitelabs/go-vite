package version

import "fmt"

// For "govendor install"

func PrintBuildVersion() {
	if VITE_COMMIT_VERSION != "" {
		fmt.Printf("this vite node`s version(%s), git commit %s, ", VITE_BUILD_VERSION, VITE_COMMIT_VERSION)
	} else {
		fmt.Println("can not read gitversion file please use Make to build Vite ")
	}
}
