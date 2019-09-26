package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"gitlab.com/NebulousLabs/Sia/modules"

	sia "gitlab.com/NebulousLabs/Sia/node/api/client"
)

// SiaFolder is a folder that is synchronized to a Sia node.
type SiaFolder struct {
	path    string
	client  *sia.Client
	archive bool
	prefix  string
	watcher *fsnotify.Watcher

	files map[string]string // files is a map of file paths to SHA256 checksums, used to reconcile file changes

	closeChan chan struct{}
}

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
func checkFile(path string) (bool, error) {
	if include != "" {
		if contains(includeExtensions, strings.TrimLeft(filepath.Ext(path), ".")) {
			if debug {
				fmt.Println("[DEBUG] Found extension in include flag")
			}
			return true, nil
		}
		if debug {
			fmt.Println("[DEBUG] Extension not found in include flag")
		}
		return false, nil

	}

	if exclude != "" {
		if contains(excludeExtensions, strings.TrimLeft(filepath.Ext(path), ".")) {
			return false, nil
		}
		return true, nil
	}
	return true, nil
}

// NewSiafolder creates a new SiaFolder using the provided path and api
// address.
func NewSiafolder(path string, client *sia.Client) (*SiaFolder, error) {
	sf := &SiaFolder{}

	abspath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	sf.path = abspath
	sf.files = make(map[string]string)
	sf.closeChan = make(chan struct{})
	sf.client = client
	sf.archive = archive
	sf.prefix = prefix

	// watch for file changes
	sf.watcher = nil
	if !syncOnly {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return nil, err
		}
		err = watcher.Add(abspath)
		if err != nil {
			return nil, err
		}

		sf.watcher = watcher
	}

	// walk the provided path, accumulating a slice of files to potentially
	// upload and adding any subdirectories to the watcher.
	err = filepath.Walk(abspath, func(walkpath string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if walkpath == path {
			return nil
		}

		// Check if a Directory was found
		if f.IsDir() {
			// subdirectories must be added to the watcher.
			if sf.watcher != nil {
				sf.watcher.Add(walkpath)
			}
			return nil
		}

		// File Found
		if debug {
			log.Printf("[DEBUG] Calculating checksum for: %v\n", walkpath)
		}
		checksum, err := checksumFile(walkpath)
		if err != nil {
			return err
		}
		sf.files[walkpath] = checksum
		return nil
	})
	if err != nil {
		return nil, err
	}

	log.Println("Uploading files missing from Sia")
	err = sf.uploadNonExisting()
	if err != nil {
		return nil, err
	}

	// remove files that are in Sia but not in local directory
	if !archive {
		log.Println("Removing files missing from local directory")
		err = sf.removeDeleted()
		if err != nil {
			return nil, err
		}
	}

	// since there is no simple way to retrieve a sha256 checksum of a remote
	// file, this only works in size-only mode
	if sizeOnly {
		log.Println("Uploading changed files")
		err = sf.uploadChanged()
	}
	if err != nil {
		return nil, err
	}

	go sf.eventWatcher()

	return sf, nil
}

// newSiaPath is a wrapper for modules.NewSiaPath that just panics if there is
// an error
func newSiaPath(path string) (siaPath modules.SiaPath) {
	siaPath, err := modules.NewSiaPath(path)
	if err != nil {
		panic(err)
	}
	return siaPath
}

// checksumFile returns a sha256 checksum or size of a given file on disk
// depending on a options provided
func checksumFile(path string) (string, error) {
	var checksum string
	var err error

	if sizeOnly {
		checksum, err = sizeFile(path)
	} else {
		checksum, err = sha256File(path)
	}
	if err != nil {
		return "", err
	}
	return checksum, nil
}

// sha256File returns a sha256 checksum of a given file on disk.
func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return string(h.Sum(nil)), nil
}

// sizeFile returns the file size
func sizeFile(path string) (string, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	size := stat.Size()

	return strconv.FormatInt(size, 10), nil
}

// eventWatcher continuously listens on the SiaFolder's watcher channels and
// performs the necessary upload/delete operations.
func (sf *SiaFolder) eventWatcher() {
	if sf.watcher == nil {
		return
	}

	for {
		select {
		case <-sf.closeChan:
			return
		case event := <-sf.watcher.Events:
			filename := filepath.Clean(event.Name)
			f, err := os.Stat(filename)
			if err == nil && f.IsDir() {
				sf.watcher.Add(filename)
				continue
			}
			goodForWrite, err := checkFile(filename)
			if err != nil {
				log.Println(err)
			}
			if !goodForWrite {
				continue
			}

			// WRITE event, checksum the file and re-upload it if it has changed
			if event.Op&fsnotify.Write == fsnotify.Write {
				err = sf.handleFileWrite(filename)
				if err != nil {
					log.Println(err)
				}
			}

			// REMOVE event
			if event.Op&fsnotify.Remove == fsnotify.Remove && !sf.archive {
				log.Println("file removal detected, removing", filename)
				err = sf.handleRemove(filename)
				if err != nil {
					log.Println(err)
				}
			}

			// CREATE event
			if event.Op&fsnotify.Create == fsnotify.Create {
				log.Println("file creation detected, uploading", filename)
				uploadRetry(sf, filename)
			}
		case err := <-sf.watcher.Errors:
			if err != nil {
				log.Println("fsevents error:", err)
			}
		}
	}
}

// uploadRetry attempts to reupload a file to Sia
func uploadRetry(sf *SiaFolder, filename string) {
	err := sf.handleCreate(filename)
	if err == nil {
		return
	}

	// If there was an error returned from handleCreate, sleep for 10s and then
	// remove and try again
	time.Sleep(10 * time.Second)

	// check if we have received create event for a file that is already in sia
	exists, err := sf.isFile(filename)
	if err != nil {
		log.Println(err)
	}
	if exists && !archive {
		err := sf.handleRemove(filename)
		if err != nil {
			log.Println(err)
		}
	}

	err2 := sf.handleCreate(filename)
	if err2 != nil {
		log.Println(err2)
	}
}

// isFile checks to see if the file exists on Sia
func (sf *SiaFolder) isFile(file string) (bool, error) {
	relpath, err := filepath.Rel(sf.path, file)
	if err != nil {
		return false, fmt.Errorf("error getting relative path: %v\n", err)
	}

	_, err = sf.client.RenterFileGet(newSiaPath(relpath))
	exists := true
	if err != nil && strings.Contains(err.Error(), "no file known") {
		exists = false
	}
	return exists, nil
}

// handleFileWrite handles a WRITE fsevent.
func (sf *SiaFolder) handleFileWrite(file string) error {
	checksum, err := checksumFile(file)
	if err != nil {
		return err
	}

	oldChecksum, exists := sf.files[file]
	if exists && oldChecksum != checksum {
		log.Printf("change in %v detected, reuploading..\n", file)
		sf.files[file] = checksum
		if !sf.archive {
			err = sf.handleRemove(file)
			if err != nil {
				return err
			}
		}
		err = sf.handleCreate(file)
		if err != nil {
			return err
		}
	}

	return nil
}

// Close releases any resources allocated by a SiaFolder.
func (sf *SiaFolder) Close() error {
	close(sf.closeChan)
	if sf.watcher != nil {
		return sf.watcher.Close()
	}
	return nil
}

// getSiaPath returns a SiaPath for relative file name with prefix appended
func getSiaPath(relpath string) modules.SiaPath {
	return newSiaPath(filepath.Join(prefix, relpath))
}

// handleCreate handles a file creation event. `file` is a relative path to the
// file on disk.
func (sf *SiaFolder) handleCreate(file string) error {
	abspath, err := filepath.Abs(file)
	if err != nil {
		return fmt.Errorf("error getting absolute path to upload: %v\n:", err)
	}
	relpath, err := filepath.Rel(sf.path, file)
	if err != nil {
		return fmt.Errorf("error getting relative path to upload: %v\n", err)
	}

	if debug {
		log.Printf("[DEBUG] Uploading: %v\n", abspath)
	}

	if !dryRun {
		err = sf.client.RenterUploadPost(abspath, getSiaPath(relpath), dataPieces, parityPieces)
		if err != nil {
			return fmt.Errorf("error uploading %v: %v\n", file, err)
		}
	}

	checksum, err := checksumFile(file)
	if err != nil {
		return err
	}
	sf.files[file] = checksum
	return nil
}

// handleRemove handles a file removal event.
func (sf *SiaFolder) handleRemove(file string) error {
	relpath, err := filepath.Rel(sf.path, file)
	if err != nil {
		return fmt.Errorf("error getting relative path to remove: %v\n", err)
	}

	if debug {
		log.Printf("[DEBUG] Deleting: %v", file)
	}

	if !dryRun {
		err = sf.client.RenterDeletePost(getSiaPath(relpath))
		if err != nil {
			return fmt.Errorf("error removing %v: %v\n", file, err)
		}
	}

	delete(sf.files, file)
	return nil
}

// uploadNonExisting runs once and performs any uploads required to ensure
// every file in files is uploaded to the Sia node.
func (sf *SiaFolder) uploadNonExisting() error {
	renterFiles, err := sf.getSiaFiles()
	if err != nil {
		return err
	}

	for file := range sf.files {
		goodForWrite, err := checkFile(filepath.Clean(file))
		if err != nil {
			log.Println(err)
		}
		if !goodForWrite {
			continue
		}

		relpath, err := filepath.Rel(sf.path, file)
		if err != nil {
			return err
		}
		if _, ok := renterFiles[getSiaPath(relpath)]; !ok {
			err := sf.handleCreate(file)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// uploadChanged runs once and performs any uploads of files where file size in
// Sia is different from local file
func (sf *SiaFolder) uploadChanged() error {
	renterFiles, err := sf.getSiaFiles()
	if err != nil {
		return err
	}

	for file := range sf.files {
		goodForWrite, err := checkFile(filepath.Clean(file))
		if err != nil {
			log.Println(err)
		}
		if !goodForWrite {
			continue
		}

		relpath, err := filepath.Rel(sf.path, file)
		if err != nil {
			return err
		}
		if siafile, ok := renterFiles[getSiaPath(relpath)]; ok {
			sf.files[file] = strconv.FormatInt(int64(siafile.Filesize), 10)
			// set file size to size in Sia and call handleFileWrite
			// if local file has different size it will reload file to Sia
			err := sf.handleFileWrite(file)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// removeDeleted runs once and removes any files from Sia that don't exist in
// local directory anymore
func (sf *SiaFolder) removeDeleted() error {
	renterFiles, err := sf.getSiaFiles()
	if err != nil {
		return err
	}

	for siapath, siafile := range renterFiles {
		goodForWrite, err := checkFile(filepath.Clean(siafile.SiaPath.Path))
		if err != nil {
			log.Println(err)
		}
		if !goodForWrite {
			continue
		}

		filePath := filepath.Join(sf.path, siapath.String())
		if _, ok := sf.files[filePath]; !ok {
			err = sf.handleRemove(filePath)
			if err != nil {
				log.Println(err)
			}
		}
	}

	return nil
}

// filters Sia remote files, only files that match prefix parameter are returned
func (sf *SiaFolder) getSiaFiles() (map[modules.SiaPath]modules.FileInfo, error) {
	siaSyncDir, err := sf.client.RenterGetDir(getSiaPath(prefix))
	if err != nil {
		return nil, err
	}
	siaFiles := make(map[modules.SiaPath]modules.FileInfo)
	for _, file := range siaSyncDir.Files {
		siaFiles[file.SiaPath] = file
	}
	return siaFiles, nil
}
