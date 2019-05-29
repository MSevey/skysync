package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"

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
)

func Usage() {
	fmt.Printf(`usage: siasync <flags> <directory-to-sync>
  for example: ./siasync -password abcd123 /tmp/sync/to/sia

`)
	flag.PrintDefaults()
}

// findApiPassword looks for the API password via a flag, env variable, or the default apipassword file
func findApiPassword() string {
	// password from cli -password flag
	if password != "" {
		return password
	} else {
		// password from environment variable
		envPassword := os.Getenv("SIA_API_PASSWORD")
		if envPassword != "" {
			return envPassword
		} else {
			// password from apipassword file
			APIPasswordFile, err := ioutil.ReadFile(build.APIPasswordFile(build.DefaultSiaDir()))
			if err != nil {
				fmt.Println("Could not read API password file:", err)
			}
			return strings.TrimSpace(string(APIPasswordFile))
		}
	}

}

func testConnection(sc *sia.Client) {
	// Get siad Version
	version, err := sc.DaemonVersionGet()
	if err != nil {
		panic(err)
	}
	log.Println("Connected to Sia ", version.Version)

	// Check Allowance
	rg, err := sc.RenterGet()
	if err != nil {
		log.Fatal("Could not get renter info:", err)
	}
	if rg.Settings.Allowance.Funds.IsZero() {
		log.Fatal("Cannot upload: No allowance available")
	}

	// Check Contracts
	rc, err := sc.RenterDisabledContractsGet()
	if err != nil {
		log.Fatal("Could not get renter contracts", err)
	}
	if len(rc.ActiveContracts) == 0 {
		log.Fatal("No active contracts")
	}
	var GoodForUpload = 0
	for _, c := range rc.ActiveContracts {
		if c.GoodForUpload {
			GoodForUpload += 1
		}
	}
	log.Println(GoodForUpload, " contracts are ready for upload")

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
	flag.StringVar(&include, "exclude", "", "Comma separated list of file extensions to skip, all other files will be copied.")

	flag.Parse()

	sc := sia.New(*address)
	sc.Password = findApiPassword()
	sc.UserAgent = *agent
	directory := os.Args[len(os.Args)-1]

	// Verify that we can talk to Sia and have valid contracts.
	testConnection(sc)

	includeExtensions = strings.Split(include, ",")
	excludeExtensions = strings.Split(exclude, ",")

	sf, err := NewSiafolder(directory, sc)
	if err != nil {
		log.Fatal(err)
	}
	defer sf.Close()

	log.Println("watching for changes to ", directory)

	done := make(chan os.Signal)
	signal.Notify(done, os.Interrupt)
	<-done
	fmt.Println("caught quit signal, exiting...")
}
