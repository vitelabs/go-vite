package fileutils

import (
	"github.com/deckarep/golang-set"
	"io/ioutil"
	"path/filepath"
	"sync"
	"time"
	"os"
)

// this is an file system cache
type FileChangeRecord struct {
	AllCached     mapset.Set                              // all cached file
	FileFilter    func(dir string, file os.FileInfo) bool //  if cb return true  represents cb digest the file
	mutex         sync.RWMutex
	latestModTime time.Time // the latest modified file`s modify time
}

func (fc *FileChangeRecord) RefreshCache(keyDir string) (creates mapset.Set, deletes mapset.Set, updates mapset.Set, err error) {

	files, err := ioutil.ReadDir(keyDir)
	if err != nil {
		return nil, nil, nil, err
	}

	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	all := mapset.NewThreadUnsafeSet()
	mods := mapset.NewThreadUnsafeSet()

	var latestModTime time.Time
	for _, f := range files {
		path := filepath.Join(keyDir, f.Name())
		if fc.FileFilter(keyDir, f) {
			continue
		}

		modTime := f.ModTime()
		if modTime.After(fc.latestModTime) {
			mods.Add(path)
		}
		if modTime.After(latestModTime) {
			latestModTime = modTime
		}

		all.Add(path)
	}
	fc.latestModTime = latestModTime

	creates = all.Difference(fc.AllCached) // the existing - the oldcache = creates
	deletes = fc.AllCached.Difference(all) // The oldcache - the existing = deletes
	updates = mods.Difference(creates)     // all modified file - new files = those old modified files

	fc.AllCached = all

	return creates, deletes, updates, nil
}
