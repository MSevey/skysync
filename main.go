package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"

	"github.com/sirupsen/logrus"
	"gitlab.com/NebulousLabs/Sia/build"
	sia "gitlab.com/NebulousLabs/Sia/node/api/client"
)

var (
	archive           bool
	debug             bool
	password          string
	prefix            string
	include           string
	includeExtensions []string
	exclude           string
	excludeExtensions []string
	siaDir            string
	dataPieces        uint64
	parityPieces      uint64
	sizeOnly          bool
	syncOnly          bool
	dryRun            bool
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
	fmt.Printf(`usage: siasync <flags> <directory-to-sync>
  for example: ./siasync -password abcd123 /tmp/sync/to/sia

`)
	flag.PrintDefaults()
}

// findAPIPassword looks for the API password via a flag, env variable, or the
// default apipassword file
func findAPIPassword() string {
	// password from cli -password flag
	if password != "" {
		log.Info("Using API password submitted by user")
		return password
	}

	// password from environment variable
	envPassword := os.Getenv("SIA_API_PASSWORD")
	if envPassword != "" {
		log.Info("Using Environnement Variable API password")
		return envPassword
	}

	// password from apipassword file
	APIPasswordFile, err := ioutil.ReadFile(build.APIPasswordFile(build.DefaultSiaDir()))
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Error("Could not read API password file")
	}
	return strings.TrimSpace(string(APIPasswordFile))
}

// testConnection test the connection to the sia network
func testConnection(sc *sia.Client) {
	// Get siad Version
	version, err := sc.DaemonVersionGet()
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Panic("Could not get the Daemon version")
	}
	log.WithFields(logrus.Fields{
		"version": version.Version,
	}).Info("Connected to Sia")

	// Check Allowance
	rg, err := sc.RenterGet()
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("Could not get renter info")
	}
	if rg.Settings.Allowance.Funds.IsZero() {
		log.Fatal("Cannot upload: No allowance available")
	}

	// Check Contracts
	rc, err := sc.RenterDisabledContractsGet()
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("Could not get renter contracts")
	}
	if len(rc.ActiveContracts) == 0 {
		log.Fatal("No active contracts")
	}
	log.WithFields(logrus.Fields{
		"contracts": len(rc.ActiveContracts),
	}).Info("contracts are ready for upload")

}

func main() {
	flag.Usage = Usage
	address := flag.String("address", "127.0.0.1:9980", "Sia's API address")
	flag.StringVar(&password, "password", "", "Sia's API password")
	agent := flag.String("agent", "Sia-Agent", "Sia agent")
	flag.BoolVar(&archive, "archive", false, "Files will not be removed from Sia, even if they are deleted locally")
	flag.BoolVar(&debug, "debug", false, "Enable debug mode. Warning: generates a lot of output.")
	flag.StringVar(&prefix, "subfolder", "siasync", "Folder on Sia to sync files too")
	flag.StringVar(&include, "include", "", "Comma separated list of file extensions to copy, all other files will be ignored.")
	flag.StringVar(&exclude, "exclude", "", "Comma separated list of file extensions to skip, all other files will be copied.")
	flag.Uint64Var(&dataPieces, "data-pieces", 10, "Number of data pieces in erasure code")
	flag.Uint64Var(&parityPieces, "parity-pieces", 30, "Number of parity pieces in erasure code")
	flag.BoolVar(&sizeOnly, "size-only", false, "Compare only based on file size and not on checksum")
	flag.BoolVar(&syncOnly, "sync-only", false, "Sync, don't monitor directory for changes")
	flag.BoolVar(&dryRun, "dry-run", false, "Show what would have been uploaded without changing files in Sia")

	flag.Parse()

	// Init the logger
	initLogger(debug)

	sc := sia.New(*address)
	sc.Password = findAPIPassword()
	sc.UserAgent = *agent
	directory := os.Args[len(os.Args)-1]

	// Verify that we can talk to Sia and have valid contracts.
	testConnection(sc)

	includeExtensions = strings.Split(include, ",")
	excludeExtensions = strings.Split(exclude, ",")

	sf, err := NewSiafolder(directory, sc)
	if err != nil {
		log.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("Could not create new Siafolder")
	}
	defer sf.Close()

	if !syncOnly {
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
