package messaging

import (
	"context"

	"github.com/perocha/producer/pkg/domain/event"
)

// EventWithOperationID struct that contains an operation ID and an Event
type Message struct {
	OperationID string
	Error       error
	Event       event.Event
}

type MessagingSystem interface {
	Publish(ctx context.Context, event event.Event) error
	Close(ctx context.Context) error
}
