package service

import (
	"context"

	"github.com/perocha/goadapters/messaging/message"
	"github.com/perocha/goutils/pkg/telemetry"
)

// ServiceImpl struct
type ServiceImpl struct {
	messagingClient message.MessagingSystem
}

// Creates a new instance of ServiceImpl.
func Initialize(ctx context.Context, messagingSystem message.MessagingSystem) *ServiceImpl {
	return &ServiceImpl{
		messagingClient: messagingSystem,
	}
}

// Publish an event to the messaging system
func (s *ServiceImpl) PublishEvent(ctx context.Context, data message.Message) error {
	telemetryClient := telemetry.GetTelemetryClient(ctx)

	err := s.messagingClient.Publish(ctx, data)
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
