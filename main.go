package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/sirupsen/logrus"
)

// TODO
//  - Current the default skynet upload options are used. This could be updated
//  to accept custom skynet upload options

var (
	debug    bool
	dryRun   bool
	exclude  string
	include  string
	siaDir   string
	syncOnly bool
)

// log is the logger for outputting info to the terminal
var log *logrus.Logger

// initLogger initializes the logger
func initLogger(debug bool) {
	log = logrus.New()

	// Define logger level
	if debug {
		log.SetLevel(logrus.DebugLevel)
	} else {
		log.SetLevel(logrus.InfoLevel)
	}

	// Print out file names and line numbers
	log.SetReportCaller(true)
}

// Usage prints out an example usage command and the defaults for the flags
func Usage() {
	fmt.Printf(`usage: skysync <flags> <directory-to-sync>
  for example: ./skysync --dry-run=true /tmp/sync/to/skynet

`)
	flag.PrintDefaults()
}

func main() {
	flag.Usage = Usage
	flag.BoolVar(&debug, "debug", false, "Enable debug mode. Warning: generates a lot of output.")
	flag.BoolVar(&dryRun, "dry-run", false, "Show what would have been uploaded without changing files in Sia")
	flag.StringVar(&exclude, "exclude", "", "Comma separated list of file extensions to skip, all other files will be copied.")
	flag.StringVar(&include, "include", "", "Comma separated list of file extensions to copy, all other files will be ignored.")
	flag.BoolVar(&syncOnly, "sync-only", false, "Sync, don't monitor directory for changes")

	flag.Parse()

	// Init the logger
	initLogger(debug)

	directory := os.Args[len(os.Args)-1]

	// Log the parse flags
	log.WithFields(logrus.Fields{
		"debug":     debug,
		"dryrun":    dryRun,
		"exclude":   exclude,
		"include":   include,
		"syncOnly":  syncOnly,
		"directory": directory,
	}).Debug("flags parsed")

	// Set the parsed options
	opts := syncOpts{
		syncOnly: syncOnly,
	}
	if exclude != "" {
		opts.excludeExtensions = strings.Split(exclude, ",")
	}
	if include != "" {
		opts.includeExtensions = strings.Split(include, ",")
	}

	log.Info("Creating SkySync")
	ss, err := NewSkySync(directory, opts)
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("Could not create new SkySync")
	}
	defer ss.Close()

	// Upload Directory
	err = ss.UploadDir()
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("Could not upload directory")
		return
	}

	if !syncOnly {
		// Launch event watcher
		go ss.eventWatcher()

		log.WithFields(logrus.Fields{
			"directory": directory,
		}).Info("Watching Directory for changes")

		done := make(chan os.Signal)
		signal.Notify(done, os.Interrupt)
		<-done
		log.Error("caught quit signal, exiting...")
	}
	log.Info("Done")
}
