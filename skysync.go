package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/NebulousLabs/go-skynet"
	"github.com/fishman/fsnotify"
	"github.com/sirupsen/logrus"
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
		filesToUpload map[string]struct{}
		opts          syncOpts
		root          string
		skyfiles      map[string]string
		watcher       *fsnotify.Watcher

		closeChan chan struct{}
	}
)

// contains is a helper that checks if a string exists in a []strings.
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
	// Get the absolute path of the directory
	abspath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	log.WithFields(logrus.Fields{
		"path":    path,
		"abspath": abspath,
	}).Debug("Path Provided and abspath found")

	// Initialize SkySync
	ss := &SkySync{
		root:          abspath,
		opts:          opts,
		filesToUpload: make(map[string]struct{}),
		skyfiles:      make(map[string]string),
		closeChan:     make(chan struct{}),
		watcher:       nil,
	}

	// Try and load persistence
	err = ss.load()
	if err != nil {
		log.WithFields(logrus.Fields{
			"err": err.Error(),
		}).Debug("error loading persistence")
		return nil, err
	}

	// watch for file changes
	if !ss.opts.syncOnly {
		watcher, err := newWatcher(ss.root)
		if err != nil {
			return nil, err
		}

		ss.watcher = watcher
	}

	// Save SkySync
	return ss, ss.save()
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

		// Add file to list of files to be uploaded
		log.WithFields(logrus.Fields{
			"file": walkpath,
		}).Debug("Adding file to files to be uploaded")
		ss.filesToUpload[walkpath] = struct{}{}
		return nil
	})
	if err != nil {
		return err
	}

	// Uploaded all pending files to SkyNet
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

// uploadNonExisting uploads any pending files to SkyNet.
func (ss *SkySync) uploadNonExisting() error {
	for file := range ss.filesToUpload {
		// Check if the file is good for upload
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

		// Upload file to SkyNet
		log.WithFields(logrus.Fields{
			"file": file,
		}).Info("uploading file to skynet")
		if dryRun {
			ss.skyfiles[file] = ""
			delete(ss.filesToUpload, file)
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
	// Persist to disk
	return ss.save()
}
