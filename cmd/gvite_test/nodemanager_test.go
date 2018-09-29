package main

import (
	"fmt"
	"gopkg.in/urfave/cli.v1"
	"testing"
)

func TestNewApp(t *testing.T) {
	fmt.Println("sss")
}

func TestMergeFlags(t *testing.T) {

	// Network Settings
	OneFlag := cli.StringFlag{
		Name:  "identity", //mapping:p2p.Name
		Usage: "Custom node name",
	}

	// Network Settings
	TwoFlag := cli.StringFlag{
		Name:  "identity", //mapping:p2p.Name
		Usage: "Custom node name",
	}

	fmt.Println(fmt.Sprintf("merge flags: %v", len(MergeFlags([]cli.Flag{OneFlag, TwoFlag}, []cli.Flag{OneFlag, TwoFlag}))))
}

func MergeFlags(flagsSet ...[]cli.Flag) []cli.Flag {

	mergeFlags := []cli.Flag{}

	for _, flags := range flagsSet {

		mergeFlags = append(mergeFlags, flags...)
	}
	return mergeFlags
}
