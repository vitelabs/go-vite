package vite

import (
	"archive/zip"
	"github.com/vitelabs/go-vite/log15"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

var donwloadLedgerLog = log15.New("module", "donwloadLedgerLog")

const downloadTryTimes = 5

func downloadLedger(isDownload bool, dataDir string) {
	if !isDownload {
		return
	}

	filePath := filepath.Join(dataDir, "./ledger")

	modTime := getFileModTime(filePath)
	if time.Now().Unix()-modTime < 8*3600 {
		return
	}

	var res *http.Response
	iterIndex := 0
	for ; iterIndex < downloadTryTimes; iterIndex++ {
		var getErr error
		res, getErr = http.Get("https://testnet.vite.net/ledger")

		if getErr != nil || res == nil || res.StatusCode != 200 {
			if getErr != nil {
				donwloadLedgerLog.Error(getErr.Error())
			}
			donwloadLedgerLog.Info("Retry download, index is " + strconv.Itoa(iterIndex))
			time.Sleep(time.Second) // sleep one second
		} else {
			break
		}
	}

	if iterIndex >= downloadTryTimes {
		return
	}

	zipPath := filepath.Join(dataDir, "./ledger.zip")

	os.Remove(zipPath)
	zipFile, createErr := os.Create(zipPath)
	if createErr != nil {
		donwloadLedgerLog.Error(createErr.Error())
		return
	}

	_, copyErr := io.Copy(zipFile, res.Body)

	if copyErr != nil {
		donwloadLedgerLog.Error(copyErr.Error())
		return
	}

	os.Remove(filePath)
	deCompressByPath(zipPath, dataDir)
}

func getFileModTime(path string) int64 {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return 0
	}

	return fi.ModTime().Unix()
}

func deCompressByPath(tarFile, dest string) error {
	srcFile, err := os.Open(tarFile)
	if err != nil {
		donwloadLedgerLog.Error(err.Error())
		return err
	}
	defer srcFile.Close()

	return deCompress(srcFile, dest)
}

func deCompress(srcFile *os.File, dest string) error {
	zipFile, err := zip.OpenReader(srcFile.Name())
	if err != nil {
		donwloadLedgerLog.Error("Unzip File Error：", err.Error())
		return err
	}
	defer zipFile.Close()
	for _, innerFile := range zipFile.File {
		info := innerFile.FileInfo()
		if info.IsDir() {
			err = os.MkdirAll(filepath.Join(dest, innerFile.Name), os.ModePerm)
			if err != nil {
				donwloadLedgerLog.Error("Unzip File Error : " + err.Error())
				return err
			}
			continue
		}
		srcFile, err := innerFile.Open()
		if err != nil {
			donwloadLedgerLog.Error("Unzip File Error : " + err.Error())
			continue
		}
		defer srcFile.Close()
		newFile, err := os.Create(filepath.Join(dest, innerFile.Name))
		if err != nil {
			donwloadLedgerLog.Error("Unzip File Error : " + err.Error())
			continue
		}
		io.Copy(newFile, srcFile)
		newFile.Close()
	}
	return nil
}
