package net

import (
	"testing"
)

func create() (fs *fileServer, fc *fileClient, err error) {
	fs, err = newFileServer(8488, nil)
	if err != nil {
		return
	}

	fc = newFileClient(nil)

	return
}

func TestException(t *testing.T) {
	fs, fc, err := create()
	if err != nil {
		t.Error(err)
	}

	fs.start()
	defer fs.stop()
	fc.start()
	defer fc.stop()
}
