package eventhub

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azeventhubs"
	"github.com/perocha/goutils/pkg/telemetry"
	"github.com/perocha/producer/pkg/domain/event"
)

type ProducerClient struct {
	client       *azeventhubs.ProducerClient
	eventHubName string
}

// Initialize a new EventHub producer instance
func ProducerInit(ctx context.Context, connectionString, eventHubName string) (*ProducerClient, error) {
	telemetryClient := telemetry.GetTelemetryClient(ctx)

	// Create a new producer client
	client, err := azeventhubs.NewProducerClientFromConnectionString(connectionString, eventHubName, nil)
	if err != nil {
		properties := map[string]string{
			"Error": err.Error(),
		}
		telemetryClient.TrackException(ctx, "EventHub::ProducerInit::Failed", err, telemetry.Critical, properties, true)
		return nil, err
	}

	telemetryClient.TrackTrace(ctx, "EventHub::ProducerInit::Eventhub initialization completed successfully", telemetry.Information, nil, true)

	return &ProducerClient{
		client:       client,
		eventHubName: eventHubName,
	}, nil
}

// Publish an event to the EventHub
func (p *ProducerClient) Publish(ctx context.Context, event event.Event) error {
	telemetryClient := telemetry.GetTelemetryClient(ctx)
	startTime := time.Now()

	// Check if EventHub is initialized
	if p == nil {
		err := errors.New("eventhub producer is not initialized")
		properties := map[string]string{
			"Error": err.Error(),
		}
		telemetryClient.TrackException(ctx, "EventHub::Publish::Failed", err, telemetry.Critical, properties, true)
		return err
	}

	// Create a new batch
	batch, err := p.client.NewEventDataBatch(ctx, nil)
	if err != nil {
		panic(err)
	}

	// Convert the message to JSON
	jsonData, err := json.Marshal(event)
	if err != nil {
		// Failed to marshal message, log dependency failure to App Insights
		telemetryClient.TrackException(ctx, "EventHub::Publish::Failed", err, telemetry.Critical, nil, true)
		return err
	}

	// Can be called multiple times with new messages until you receive an azeventhubs.ErrMessageTooLarge
	err = batch.AddEventData(&azeventhubs.EventData{
		Body: []byte(jsonData),
	}, nil)

	if errors.Is(err, azeventhubs.ErrEventDataTooLarge) {
		// Message too large to fit into this batch.
		//
		// At this point you'd usually just send the batch (using ProducerClient.SendEventDataBatch),
		// create a new one, and start filling up the batch again.
		//
		// If this is the _only_ message being added to the batch then it's too big in general, and
		// will need to be split or shrunk to fit.
		log.Printf("Publish::Message too large to fit into this batch\n")
		telemetryClient.TrackException(ctx, "Publish::Message too large to fit into this batch", err, telemetry.Critical, nil, true)
		return err
	} else if err != nil {
		// Some other error occurred
		log.Printf("Publish::Failed to add message to batch: %s\n", err.Error())
		telemetryClient.TrackException(ctx, "Publish::Failed to add message to batch", err, telemetry.Critical, nil, true)
		return err
	}

	// Send the batch
	err = p.client.SendEventDataBatch(context.TODO(), batch, nil)

	if err != nil {
		telemetryClient.TrackException(ctx, "Publish::Failed to send message", err, telemetry.Critical, nil, true)
		return err
	}

	telemetryClient.TrackDependency(ctx, "Eventhub", "Publish EventHub message", "EventHub", p.eventHubName, true, startTime, time.Now(), nil, true)

	return nil
}

// Close the EventHub producer
func (p *ProducerClient) Close(ctx context.Context) error {
	telemetryClient := telemetry.GetTelemetryClient(ctx)

	err := p.client.Close(ctx)
	if err != nil {
		properties := map[string]string{
			"Error": err.Error(),
		}
		telemetryClient.TrackException(ctx, "EventHub::Close::Failed", err, telemetry.Critical, properties, true)
		return err
	}

	telemetryClient.TrackTrace(ctx, "EventHub::Closing eventhub", telemetry.Information, nil, true)

	return nil
}
