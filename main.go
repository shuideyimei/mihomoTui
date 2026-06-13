package main

import (
	"log"
	app "mihomoTui/internal"
	"mihomoTui/internal/api"
	"mihomoTui/internal/utils"
)

var (
	appName    = "mihomoTui"
	appVersion = "v0.0-Alpha"
)

func main() {
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
		app.Stop()
		log.Fatalf("Failed to run application: %v", err)
	}
}
