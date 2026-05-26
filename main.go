package main

import (
	"log"
	app "mihomoTui/internal"
	"mihomoTui/internal/api"
	"mihomoTui/internal/utils"
)

func main() {
	// Initialize application logging
	shutdownLog := api.InitLogging()
	defer shutdownLog()

	// Create new application with build info
	app := app.NewApp(
		utils.GetEnvWithDefault("APP_NAME", "mihomoTui"),
		utils.GetEnvWithDefault("APP_VERSION", "v0.0-Alpha"),
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
