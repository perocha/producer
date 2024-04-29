package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
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

	// Start the service instance
	serviceInstance := service.Initialize(ctx, eventHubInstance)
	if serviceInstance == nil {
		xTelemetry.Error(ctx, "Main::Failed to initialize service", telemetry.String("Error", "Failed to initialize service"))
		panic("Failed to initialize service")
	}
	xTelemetry.Info(ctx, "Main::Service initialized successfully")

	// Define an HTTP handler function for refreshing configuration
	http.HandleFunc("/refresh-config", func(w http.ResponseWriter, r *http.Request) {
		// Refresh the configuration with the latest values
		err := cfg.RefreshConfig()
		if err != nil {
			xTelemetry.Error(ctx, "Main::Failed to refresh configuration", telemetry.String("Error", err.Error()))
			http.Error(w, "Failed to refresh configuration", http.StatusInternalServerError)
			return
		}
		xTelemetry.Info(ctx, "Main::Configuration refreshed successfully")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Configuration refreshed successfully"))
	})

	// Start an HTTP server
	go func() {
		err := http.ListenAndServe(":"+cfg.HttpPortNumber, nil)
		if err != nil {
			xTelemetry.Error(ctx, "Main::Failed to start HTTP server", telemetry.String("Error", err.Error()))
			panic(err)
		}
	}()

	// Create a channel to listen for termination signals
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// Infinite loop
	for {
		select {
		case <-signals:
			// Termination signal received
			xTelemetry.Info(ctx, "Main::Received termination signal")
			return
		case <-time.After(cfg.GetTimerDuration()):
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
				xTelemetry.Error(ctx, "Main::Failed to serialize order", telemetry.String("Error", err.Error()))
				continue
			}

			// Create the new message
			newMessage := message.NewMessage(operationID, nil, "", commandType, jsonData)

			// Publish an event to the EventHub
			err = serviceInstance.PublishEvent(ctx, newMessage)
			if err != nil {
				xTelemetry.Error(ctx, "Main::Failed to publish event", telemetry.String("Error", err.Error()))
			}

			// Add the operation ID to the context
			ctx := context.WithValue(context.Background(), telemetry.OperationIDKeyContextKey, operationID)
			xTelemetry.Info(ctx, "Main::Published event")
		}
	}
}
