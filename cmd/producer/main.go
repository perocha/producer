package main

import (
	"context"
	"log"

	"github.com/perocha/producer/pkg/appcontext"
	"github.com/perocha/producer/pkg/config"
	"github.com/perocha/producer/pkg/infrastructure/telemetry"
)

const (
	SERVICE_NAME = "Producer"
)

func main() {
	// Initialize the configuration
	cfg := config.InitializeConfig()
	if cfg == nil {
		// Print error
		log.Println("Main::Fatal error::Failed to load configuration")
		panic("Main::Failed to load configuration")
	}

	// Initialize App Insights
	telemetryClient, err := telemetry.Initialize(cfg.AppInsightsInstrumentationKey, SERVICE_NAME)
	if err != nil {
		log.Printf("Main::Fatal error::Failed to initialize App Insights %s\n", err.Error())
		panic("Main::Failed to initialize App Insights")
	}
	// Add telemetry object to the context, so that it can be reused across the application
	ctx := context.WithValue(context.Background(), appcontext.TelemetryContextKey, telemetryClient)

	telemetryClient.TrackTrace(ctx, "Producer started", telemetry.Information, nil, true)
}
