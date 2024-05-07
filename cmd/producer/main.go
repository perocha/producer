package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/perocha/goadapters/comms/httpadapter"
	"github.com/perocha/goadapters/messaging/eventhub"
	"github.com/perocha/goutils/pkg/telemetry"
	"github.com/perocha/producer/pkg/config"
	"github.com/perocha/producer/pkg/service"
)

const (
	SERVICE_NAME = "Producer"
)

func main() {
	// Load the configuration
	cfg, err := config.InitializeConfig()
	if err != nil {
		log.Fatalf("Main::Fatal error::Failed to load configuration %s\n", err.Error())
	}
	// Refresh the configuration with the latest values
	err = cfg.RefreshConfig()
	if err != nil {
		log.Fatalf("Main::Fatal error::Failed to refresh configuration %s\n", err.Error())
	}

	// Initialize telemetry package
	telemetryConfig := telemetry.NewXTelemetryConfig(cfg.AppInsightsInstrumentationKey, SERVICE_NAME, "debug", 1)
	xTelemetry, err := telemetry.NewXTelemetry(telemetryConfig)
	if err != nil {
		log.Fatalf("Main::Fatal error::Failed to initialize XTelemetry %s\n", err.Error())
	}
	// Add telemetry object to the context, so that it can be reused across the application
	ctx := context.WithValue(context.Background(), telemetry.TelemetryContextKey, xTelemetry)

	// Initialize EventHub
	eventHubInstance, err := eventhub.ProducerInitializer(ctx, cfg.EventHubName, cfg.EventHubConnectionString)
	if err != nil {
		xTelemetry.Error(ctx, "Main::Failed to initialize EventHub", telemetry.String("Error", err.Error()))
		panic(err)
	}

	// Initialize HTTP receiver
	httpReceiver, err := httpadapter.HTTPServerAdapterInit(ctx, cfg.HttpPortNumber)
	if err != nil {
		xTelemetry.Error(ctx, "Main::Failed to initialize HTTP receiver", telemetry.String("Error", err.Error()))
		panic(err)
	}

	// Create a channel to listen for termination signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// Start the service instance
	serviceInstance := service.Initialize(ctx, eventHubInstance, httpReceiver)
	if serviceInstance == nil {
		xTelemetry.Error(ctx, "Main::Failed to initialize service", telemetry.String("Error", "Failed to initialize service"))
		panic("Failed to initialize service")
	}
	xTelemetry.Info(ctx, "Main::Service initialized successfully")

	// Start the service
	go func() {
		err := serviceInstance.Start(ctx, signals)
		if err != nil {
			xTelemetry.Error(ctx, "Main::Failed to start service", telemetry.String("Error", err.Error()))
			panic(err)
		}
	}()

	// Infinite loop
	for {
		select {
		case <-signals:
			// Termination signal received
			xTelemetry.Info(ctx, "Main::Received termination signal")
			return
		case <-time.After(2 * time.Minute):
			// Do nothing
			xTelemetry.Debug(ctx, "Main::Waiting for termination signal")
		}
	}
}
