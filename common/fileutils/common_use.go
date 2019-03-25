package fileutils

import (
	"errors"
	"fmt"
	"os"
)

func OpenOrCreateFd(dirName string) (*os.File, error) {
	var dirFd *os.File
	for dirFd == nil {
		var openErr error
		dirFd, openErr = os.Open(dirName)
		if openErr != nil {
			if os.IsNotExist(openErr) {
				var cErr error
				cErr = os.MkdirAll(dirName, 0744)

				if cErr != nil {
					return nil, errors.New(fmt.Sprintf("Create %s failed, error is %s", dirName, cErr.Error()))
				}
			} else {
				return nil, errors.New(fmt.Sprintf("os.Open %s failed, error is %s", dirName, openErr.Error()))
			}
		}
	}

	return dirFd, nil
}

func FileSize(fd *os.File) (int64, error) {
	fileInfo, err := fd.Stat()
	if err != nil {
		return 0, errors.New(fmt.Sprintf("fd.Stat() failed, error is %s", err.Error()))
	}

	return fileInfo.Size(), nil
}
