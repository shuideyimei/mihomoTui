package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	app "mihomoTui/internal"
	"mihomoTui/internal/api"
	"mihomoTui/internal/utils"
)

var (
	appName    = "mihomoTui"
	appVersion = "v0.1.0"
)

func main() {
	versionFlag := flag.Bool("version", false, "Print application version and exit")
	shortVersionFlag := flag.Bool("v", false, "Print application version and exit (short)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s - Terminal UI client for Mihomo proxy\n\n", appName)
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *versionFlag || *shortVersionFlag {
		fmt.Printf("%s %s\n", appName, utils.GetEnvWithDefault("APP_VERSION", appVersion))
		return
	}

	// Initialize application logging
	shutdownLog := api.InitLogging()
	defer shutdownLog()

	// Create new application with build info
	app := app.NewApp(
		utils.GetEnvWithDefault("APP_NAME", appName),
		utils.GetEnvWithDefault("APP_VERSION", appVersion),
	)

	// Initialize the application
	if err := app.Initialize(); err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// Run the application
	if err := app.Run(); err != nil {
		log.Fatalf("Failed to run application: %v", err)
	}
}
