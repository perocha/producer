package service

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/perocha/goadapters/comms"
	"github.com/perocha/goadapters/comms/httpadapter"
	"github.com/perocha/goadapters/messaging"
	"github.com/perocha/goutils/pkg/telemetry"
)

// ServiceImpl struct
type ServiceImpl struct {
	messagingClient messaging.MessagingSystem
	httpReceiver    comms.CommsReceiver
}

// Creates a new instance of ServiceImpl.
func Initialize(ctx context.Context, messagingSystem messaging.MessagingSystem, httpReceiver comms.CommsReceiver) *ServiceImpl {
	return &ServiceImpl{
		messagingClient: messagingSystem,
		httpReceiver:    httpReceiver,
	}
}

// Start the service
func (s *ServiceImpl) Start(ctx context.Context, signals <-chan os.Signal) error {
	xTelemetry := telemetry.GetXTelemetryClient(ctx)

	// Register the refresh configuration callback function
	err := s.httpReceiver.RegisterEndPoint(ctx, "/refresh-config", s.RefreshConfig)
	if err != nil {
		xTelemetry.Error(ctx, "Service::Start::Failed to register refresh config endpoint", telemetry.String("Error", err.Error()))
		return err
	}

	// Register the health check callback function
	err = s.httpReceiver.RegisterEndPoint(ctx, "/health", s.HealthCheck)
	if err != nil {
		xTelemetry.Error(ctx, "Service::Start::Failed to register health check endpoint", telemetry.String("Error", err.Error()))
		return err
	}

	// Register the publish event callback function
	err = s.httpReceiver.RegisterEndPoint(ctx, "/publish", s.NewEvent)
	if err != nil {
		xTelemetry.Error(ctx, "Service::Start::Failed to register publish event endpoint", telemetry.String("Error", err.Error()))
		return err
	}

	// Start the HTTP server
	err = s.httpReceiver.Start(ctx)
	if err != nil {
		xTelemetry.Error(ctx, "Service::Start::Failed to start HTTP server", telemetry.String("Error", err.Error()))
		return err
	}

	// Wait for signals
	for range signals {
		xTelemetry.Info(ctx, "Service::Start::Received signal to stop")
		s.Stop(ctx)
		return nil
	}

	return nil
}

// Stop the service
func (s *ServiceImpl) Stop(ctx context.Context) {
	xTelemetry := telemetry.GetXTelemetryClient(ctx)

	// Stop the HTTP server
	err := s.httpReceiver.Stop(ctx)
	if err != nil {
		xTelemetry.Error(ctx, "Service::Stop::Failed to stop HTTP server", telemetry.String("Error", err.Error()))
	}
}

// Refresh the configuration
func (s *ServiceImpl) RefreshConfig(ctx context.Context, w comms.ResponseWriter, r comms.Request) {
	xTelemetry := telemetry.GetXTelemetryClient(ctx)

	/*
		err := cfg.RefreshConfig()
		if err != nil {
			xTelemetry.Error(ctx, "Service::RefreshConfig::Failed", telemetry.String("Error", err.Error()))
			w.WriteHeader(int(httpadapter.StatusInternalServerError))
		}
	*/

	xTelemetry.Info(ctx, "Service::RefreshConfig::OK")
	w.WriteHeader(int(httpadapter.StatusOK))
	w.Write([]byte("Configuration refreshed successfully"))
}

// Health check
func (s *ServiceImpl) HealthCheck(ctx context.Context, w comms.ResponseWriter, r comms.Request) {
	xTelemetry := telemetry.GetXTelemetryClient(ctx)
	xTelemetry.Info(ctx, "Service::HealthCheck::OK")

	w.WriteHeader(int(httpadapter.StatusOK))
	w.Write([]byte("OK"))
}

// Create a new event
func (s *ServiceImpl) NewEvent(ctx context.Context, w comms.ResponseWriter, r comms.Request) {
	startTime := time.Now()
	xTelemetry := telemetry.GetXTelemetryClient(ctx)

	// Retrieve the body of the request
	body := r.Body()
	if body == nil {
		xTelemetry.Error(ctx, "Service::NewEvent::Failed to read body", telemetry.String("Error", "Body is nil"))
		w.WriteHeader(int(httpadapter.StatusBadRequest))
		w.Write([]byte("Failed to read body"))
		return
	}

	// Generate a new OperationID
	operationID := uuid.New().String()
	// Append the OperationID to the context
	ctx = context.WithValue(ctx, telemetry.OperationIDKeyContextKey, operationID)

	/*
		// Retrieve the headers
		operationID := r.Header("OperationID")
		if operationID == "" {
			xTelemetry.Error(ctx, "Service::NewEvent::OperationID not found", telemetry.String("Error", "OperationID not found in the request header"))
			w.WriteHeader(int(httpadapter.StatusBadRequest))
			w.Write([]byte("OperationID not found in the request header"))
			return
		}
	*/
	status := r.Header("Status")
	if status == "" {
		xTelemetry.Error(ctx, "Service::NewEvent::Status not found", telemetry.String("Error", "Status not found in the request header"))
		w.WriteHeader(int(httpadapter.StatusBadRequest))
		w.Write([]byte("Status not found in the request header"))
		return
	}
	command := r.Header("Command")
	if command == "" {
		xTelemetry.Error(ctx, "Service::NewEvent::Command not found", telemetry.String("Error", "Command not found in the request header"))
		w.WriteHeader(int(httpadapter.StatusBadRequest))
		w.Write([]byte("Command not found in the request header"))
		return
	}
	// Create a new message
	msg := messaging.NewMessage(operationID, nil, status, command, body)

	// Publish the event
	err := s.publishEvent(ctx, msg)
	if err != nil {
		xTelemetry.Error(ctx, "Service::NewEvent::Failed to publish event", telemetry.String("Error", err.Error()))
		w.WriteHeader(int(httpadapter.StatusInternalServerError))
		w.Write([]byte("Failed to publish event"))
		return
	}

	// Log the telemetry request
	duration := time.Since(startTime)
	hostname := r.Header("Host")
	userAgent := r.Header("User-Agent")
	xTelemetry.Request(ctx, http.MethodPost, hostname, duration, strconv.Itoa(http.StatusOK), true, userAgent, "HTTPAdapter::Publish::Success")

	// Return the response
	w.WriteHeader(int(httpadapter.StatusOK))
	w.Write([]byte("Event published successfully::OperationID=" + operationID))
}

// Publish an event to the messaging system
func (s *ServiceImpl) publishEvent(ctx context.Context, data messaging.Message) error {
	xTelemetry := telemetry.GetXTelemetryClient(ctx)

	err := s.messagingClient.Publish(ctx, data)
	if err != nil {
		xTelemetry.Error(ctx, "Service::Publish::Failed", telemetry.String("Error", err.Error()))
		return err
	}

	return nil
}

func (s *ServiceImpl) Close(ctx context.Context) {
	s.messagingClient.Close(ctx)
}
