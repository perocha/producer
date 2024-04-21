package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/perocha/goutils/pkg/telemetry"
	"github.com/perocha/producer/pkg/config"
	"github.com/perocha/producer/pkg/domain/event"
	"github.com/perocha/producer/pkg/domain/order"
	"github.com/perocha/producer/pkg/infrastructure/adapter/messaging/eventhub"
	"github.com/perocha/producer/pkg/service"
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
	ctx := context.WithValue(context.Background(), telemetry.TelemetryContextKey, telemetryClient)

	// Initialize EventHub
	eventHubInstance, err := eventhub.ProducerInit(ctx, cfg.EventHubConnectionString, cfg.EventHubName)
	if err != nil {
		telemetryClient.TrackException(ctx, "Main::Failed to initialize EventHub", err, telemetry.Error, nil, true)
		panic("Main::Failed to initialize EventHub")
	}

	// Start the producer
	serviceInstance := service.Initialize(ctx, eventHubInstance)
	if serviceInstance == nil {
		telemetryClient.TrackException(ctx, "Main::Failed to initialize service", err, telemetry.Error, nil, true)
		panic("Main::Failed to initialize service")
	}
	telemetryClient.TrackTrace(ctx, "Producer started", telemetry.Information, nil, true)

	// Create a channel to listen for termination signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// Loop forever
	// Infinite loop
	for {
		select {
		case <-signals:
			telemetryClient.TrackTrace(ctx, "Main::Received termination signal", telemetry.Information, nil, true)
			return
		case <-time.After(10 * time.Second):
			// Generate an event UUID
			eventID := uuid.New().String()
			orderID := uuid.New().String()

			// Initialize a new event with random order ID
			event := event.Event{
				Type:      "create_order",
				EventID:   eventID,
				Timestamp: time.Now(),
				OrderPayload: order.Order{
					Id:              orderID,
					ProductCategory: "Electronics",
					ProductID:       "ABC",
					CustomerID:      "1234",
					Status:          "Pending",
				},
			}

			// Publish an event to the EventHub
			err := serviceInstance.PublishEvent(ctx, event)
			if err != nil {
				telemetryClient.TrackException(ctx, "Main::Failed to publish event", err, telemetry.Error, nil, true)
			}
			telemetryClient.TrackTrace(ctx, "Main::Publishing event", telemetry.Information, event.ToMap(), true)
		}
	}
}
