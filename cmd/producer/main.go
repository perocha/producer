package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/perocha/goadapters/messaging/eventhub"
	"github.com/perocha/goadapters/messaging/message"
	"github.com/perocha/goutils/pkg/telemetry"
	"github.com/perocha/producer/pkg/config"
	"github.com/perocha/producer/pkg/domain/order"
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

	// Initialize App Insights
	telemetryClient, err := telemetry.Initialize(cfg.AppInsightsInstrumentationKey, SERVICE_NAME)
	if err != nil {
		log.Fatalf("Main::Fatal error::Failed to initialize App Insights %s\n", err.Error())
	}
	// Add telemetry object to the context, so that it can be reused across the application
	ctx := context.WithValue(context.Background(), telemetry.TelemetryContextKey, telemetryClient)

	// Initialize EventHub
	eventHubInstance, err := eventhub.ProducerInitializer(ctx, cfg.EventHubName, cfg.EventHubConnectionString)
	if err != nil {
		telemetryClient.TrackException(ctx, "Main::Failed to initialize EventHub", err, telemetry.Error, nil, true)
		log.Fatalf("Main::Fatal error::Failed to initialize EventHub %s\n", err.Error())
	}

	// Start the producer
	serviceInstance := service.Initialize(ctx, eventHubInstance)
	if serviceInstance == nil {
		telemetryClient.TrackException(ctx, "Main::Failed to initialize service", err, telemetry.Error, nil, true)
		log.Fatalf("Main::Fatal error::Failed to initialize service %s\n", err.Error())
	}
	telemetryClient.TrackTrace(ctx, "Producer started", telemetry.Information, nil, true)

	// Create a channel to listen for termination signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// Infinite loop
	for {
		// Refresh the configuration to get the Timer Duration
		cfg.RefreshConfig()
		timerDuration := cfg.TimerDuration
		timerDurationInt, err := time.ParseDuration(timerDuration)
		if err != nil {
			timerDurationInt = 1 * time.Minute
		}

		select {
		case <-signals:
			telemetryClient.TrackTrace(ctx, "Main::Received termination signal", telemetry.Information, nil, true)
			return
		case <-time.After(timerDurationInt):
			// Create a new message
			operationID := uuid.New().String()
			commandType := "create_order"

			// Create the order info
			orderPayload := order.Order{
				Id:              uuid.New().String(),
				ProductCategory: "Electronics",
				ProductID:       "ABC",
				CustomerID:      "1234",
				Status:          "Pending",
			}

			// Serialize the orderPayload
			jsonData, err := json.Marshal(orderPayload)
			if err != nil {
				telemetryClient.TrackException(ctx, "Main::Failed to serialize order", err, telemetry.Error, nil, true)
				continue
			}

			// Create the new message
			newMessage := message.NewMessage(operationID, nil, "", commandType, jsonData)

			// Publish an event to the EventHub
			err = serviceInstance.PublishEvent(ctx, newMessage)
			if err != nil {
				telemetryClient.TrackException(ctx, "Main::Failed to publish event", err, telemetry.Error, nil, true)
			}
			// TODO, print the message to the console
			properties := map[string]string{
				"OperationID": operationID,
			}
			telemetryClient.TrackTrace(ctx, "Main::Publishing event", telemetry.Information, properties, true)
		}
	}
}
