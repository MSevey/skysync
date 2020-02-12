package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/NebulousLabs/go-skynet"
	"github.com/fishman/fsnotify"
	"github.com/sirupsen/logrus"
)

var (
// TODO
// - create persistence filenames and structure.
// - persistence should be in the root directory
)

type (
	// syncOpts contains the options for syncing
	syncOpts struct {
		excludeExtensions []string
		includeExtensions []string
		portalURL         string
		syncOnly          bool
	}

	// SkySync is a struct that contains information needed to ensure a directory
	// is synced with Skynet
	SkySync struct {
		root    string
		opts    syncOpts
		watcher *fsnotify.Watcher

		filesToUpload map[string]struct{}
		// skyfiles is a map of file path to sky link
		skyfiles  map[string]string
		closeChan chan struct{}
	}
)

// contains checks if a string exists in a []strings.
func contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

// checkFile checks if a file's extension is included or excluded
// included takes precedence over excluded.
func (ss *SkySync) checkFile(path string) (bool, error) {
	if len(ss.opts.includeExtensions) > 0 {
		if contains(ss.opts.includeExtensions, strings.TrimLeft(filepath.Ext(path), ".")) {
			return true, nil
		}
		log.Debug("Extension not found in included extensions")
		return false, nil

	}

	if len(ss.opts.excludeExtensions) > 0 {
		if contains(ss.opts.excludeExtensions, strings.TrimLeft(filepath.Ext(path), ".")) {
			log.Debug("Found extension in excluded extensions")
			return false, nil
		}
		return true, nil
	}
	return true, nil
}

// NewSkySync creates a new SkySync
func NewSkySync(path string, opts syncOpts) (*SkySync, error) {
	abspath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	log.WithFields(logrus.Fields{
		"path":    path,
		"abspath": abspath,
	}).Debug("Path Provided and abspath found")

	ss := &SkySync{
		root:          abspath,
		opts:          opts,
		filesToUpload: make(map[string]struct{}),
		skyfiles:      make(map[string]string),
		closeChan:     make(chan struct{}),
		watcher:       nil,
	}

	// watch for file changes
	if !ss.opts.syncOnly {
		watcher, err := newWatcher(ss.root)
		if err != nil {
			return nil, err
		}

		ss.watcher = watcher
	}
	return ss, nil
}

// UploadDir recursively walks the directory and uploads files to skynet
func (ss *SkySync) UploadDir() error {
	// walk the provided path, accumulating a slice of files to potentially
	// upload and adding any subdirectories to the watcher.
	log.WithFields(logrus.Fields{
		"directory": ss.root,
	}).Debug("Recursively walking over directory to find files")
	err := filepath.Walk(ss.root, func(walkpath string, f os.FileInfo, err error) error {
		log.WithFields(logrus.Fields{
			"walkpath": walkpath,
		}).Debug("walking")
		if err != nil {
			return err
		}

		// Check if a Directory was found
		if f.IsDir() {
			// subdirectories must be added to the watcher.
			if ss.watcher != nil {
				log.WithFields(logrus.Fields{
					"sub directory": walkpath,
				}).Debug("found sub directory, adding to watcher")
				ss.watcher.Add(walkpath)
			}
			return nil
		}

		// File Found
		log.WithFields(logrus.Fields{
			"file": walkpath,
		}).Debug("Checking if file has been uploaded")
		if _, ok := ss.skyfiles[walkpath]; ok {
			return nil
		}
		log.WithFields(logrus.Fields{
			"file": walkpath,
		}).Debug("Adding file to files to be uploaded")
		ss.filesToUpload[walkpath] = struct{}{}
		return nil
	})
	if err != nil {
		return err
	}

	log.Info("Uploading files To Skynet")
	err = ss.uploadNonExisting()
	if err != nil {
		return err
	}

	return nil
}

// Close releases any resources allocated by a SkySync.
func (ss *SkySync) Close() error {
	close(ss.closeChan)
	if ss.watcher != nil {
		return ss.watcher.Close()
	}
	return nil
}

// uploadNonExisting runs once and performs any uploads required to ensure
// every file in files is uploaded to the Sia node.
func (ss *SkySync) uploadNonExisting() error {
	for file := range ss.filesToUpload {
		goodForUpload, err := ss.checkFile(filepath.Clean(file))
		if err != nil {
			log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Error("Error with checkFile")
		}
		if !goodForUpload {
			log.WithFields(logrus.Fields{
				"file": file,
			}).Debug("File is not good for upload")
			continue
		}

		log.WithFields(logrus.Fields{
			"file": file,
		}).Info("uploading file to skynet")
		if dryRun {
			continue
		}
		skylink, err := skynet.UploadFile(file, skynet.DefaultUploadOptions)
		if err != nil {
			log.WithFields(logrus.Fields{
				"error": err.Error(),
			}).Debug("File skynet upload failed")
			continue
		}

		// Add file to map and remove from slice of files to upload
		ss.skyfiles[file] = skylink
		delete(ss.filesToUpload, file)
	}
	// TODO - persist to disk
	return nil
}
