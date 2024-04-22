package service

import (
	"context"

	"github.com/perocha/goadapters/messaging"
	"github.com/perocha/goutils/pkg/telemetry"
	"github.com/perocha/producer/pkg/domain/event"
)

// ServiceImpl struct
type ServiceImpl struct {
	messagingClient messaging.MessagingSystem
}

// Creates a new instance of ServiceImpl.
func Initialize(ctx context.Context, messagingSystem messaging.MessagingSystem) *ServiceImpl {
	return &ServiceImpl{
		messagingClient: messagingSystem,
	}
}

// Publish an event to the messaging system
func (s *ServiceImpl) PublishEvent(ctx context.Context, event event.Event) error {
	telemetryClient := telemetry.GetTelemetryClient(ctx)

	err := s.messagingClient.Publish(ctx, event)
	if err != nil {
		properties := map[string]string{
			"Error": err.Error(),
		}
		telemetryClient.TrackException(ctx, "Service::Publish::Failed", err, telemetry.Error, properties, true)
		return err
	}

	return nil
}

func (s *ServiceImpl) Close(ctx context.Context) {
	s.messagingClient.Close(ctx)
}
