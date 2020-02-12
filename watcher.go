package main

import (
	"os"
	"path/filepath"

	"github.com/fishman/fsnotify"
	"github.com/sirupsen/logrus"
)

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
// performs the necessary upload/delete operations.
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
	closeEvent := event.Op&fsnotify.Close == fsnotify.Close
	updateEvent := event.Op&fsnotify.Update == fsnotify.Update
	writeEvent := event.Op&fsnotify.Write == fsnotify.Write
	log.WithFields(logrus.Fields{
		"closeEvent":  closeEvent,
		"updateEvent": updateEvent,
		"writeEvent":  writeEvent,
	}).Debug("fsnotify event")
	// Check for close and update events to know that the change is complete
	if !closeEvent && !updateEvent && !writeEvent {
		return
	}

	// TODO - filename is relative? do we need to make abs path?
	filename := filepath.Clean(event.Name)
	log.WithFields(logrus.Fields{
		"filename": filename,
	}).Info("file found by event watcher")
	f, err := os.Stat(filename)
	if err == nil && f.IsDir() {
		ss.watcher.Add(filename)
		return
	}
	goodForWrite, err := ss.checkFile(filename)
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Error with checkFile")
		return
	}
	if !goodForWrite {
		return
	}

	// Check if file path was previously uploaded as a skyfile
	if _, ok := ss.skyfiles[filename]; ok {
		log.WithFields(logrus.Fields{
			"filename": filename,
		}).Info("file already a skyfile")
		// TODO - figure out how to handle, should the skyfile be re-uploaded as any
		// changes to a file will result in the skyfile being different?
		//
		// Could try downloading the file and doing some sort of checksum? Might be
		// a V2 thing
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
