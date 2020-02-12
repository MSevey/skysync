package main

import (
	"os"
	"path/filepath"

	"github.com/fishman/fsnotify"
	"github.com/sirupsen/logrus"
)

// newWatcher creates and new fsnotify.Watcher for the path provided
func newWatcher(path string) (*fsnotify.Watcher, error) {
	log.WithFields(logrus.Fields{
		"directory": path,
	}).Info("creating watcher for directory")
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	err = watcher.Add(path)
	if err != nil {
		return nil, err
	}
	return watcher, nil
}

// eventWatcher continuously listens on the SkySync's watcher channels and
// handles events.
func (ss *SkySync) eventWatcher() {
	if ss.watcher == nil {
		return
	}

	for {
		select {
		case <-ss.closeChan:
			return
		case event := <-ss.watcher.Events:
			ss.handleWatchEvent(event)
		case err := <-ss.watcher.Errors:
			if err != nil {
				log.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("watcher error chan returned")
			}
		}
	}
}

// handleWatchEvent handles the event returned from the watcher Events Chan
func (ss *SkySync) handleWatchEvent(event fsnotify.Event) {
	// Check to see if it is a close, update, or write event
	closeEvent := event.Op&fsnotify.Close == fsnotify.Close
	updateEvent := event.Op&fsnotify.Update == fsnotify.Update
	writeEvent := event.Op&fsnotify.Write == fsnotify.Write
	log.WithFields(logrus.Fields{
		"closeEvent":  closeEvent,
		"updateEvent": updateEvent,
		"writeEvent":  writeEvent,
	}).Debug("fsnotify event")

	// If it is not a close, update, or write event there is nothing to do
	if !closeEvent && !updateEvent && !writeEvent {
		return
	}

	// Get the filename from the event
	filename := filepath.Clean(event.Name)
	log.WithFields(logrus.Fields{
		"filename": filename,
	}).Info("file found by event watcher")

	// If it is a directory, add it to the watcher
	f, err := os.Stat(filename)
	if err == nil && f.IsDir() {
		ss.watcher.Add(filename)
		return
	}

	// Check to see if the file is good to upload
	goodForUpload, err := ss.checkFile(filename)
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Error with checkFile")
		return
	}
	if !goodForUpload {
		return
	}

	// Check if file path was previously uploaded as a skyfile
	if _, ok := ss.skyfiles[filename]; ok {
		log.WithFields(logrus.Fields{
			"filename": filename,
		}).Info("file already a skyfile")
		// TODO - once SkyNet has better support for versioning we could updated
		// the file
		return
	}

	// Add to filesToUpload
	ss.filesToUpload[filename] = struct{}{}

	// Submit upload
	err = ss.uploadNonExisting()
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Error uploading files")
		return
	}
	return
}
