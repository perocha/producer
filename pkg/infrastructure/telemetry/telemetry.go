package telemetry

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/perocha/producer/pkg/appcontext"

	"github.com/microsoft/ApplicationInsights-Go/appinsights"
	"github.com/microsoft/ApplicationInsights-Go/appinsights/contracts"
)

const (
	SERVICE_NAME = "Producer"
)

// Telemetry defines the telemetry client
type Telemetry struct {
	client appinsights.TelemetryClient
}

// SeverityLevel defines the telemetry severity level
type SeverityLevel contracts.SeverityLevel

// Telemetry severity levels
const (
	Verbose     SeverityLevel = SeverityLevel(contracts.Verbose)
	Information SeverityLevel = SeverityLevel(contracts.Information)
	Warning     SeverityLevel = SeverityLevel(contracts.Warning)
	Error       SeverityLevel = SeverityLevel(contracts.Error)
	Critical    SeverityLevel = SeverityLevel(contracts.Critical)
)

// Initializes a new telemetry client
func Initialize(instrumentationKey string, serviceName string) (*Telemetry, error) {
	if instrumentationKey == "" {
		return nil, errors.New("app insights instrumentation key not initialized")
	}

	// Initialize telemetry client
	client := appinsights.NewTelemetryClient(instrumentationKey)

	// Set the role name
	client.Context().Tags.Cloud().SetRole(serviceName)

	return &Telemetry{client: client}, nil
}

// TrackTrace sends a trace telemetry event
func (t *Telemetry) TrackTrace(ctx context.Context, message string, severity SeverityLevel, properties map[string]string, logToConsole bool) {
	// Validate the telemetry client
	if t.client == nil {
		panic("Telemetry client not initialized")
	}

	// Create the log message
	txtMessage := fmt.Sprintf("%s::%s", SERVICE_NAME, message)
	// Retrieve the operationID from the context and add it to the log message
	operationID, ok := ctx.Value(appcontext.OperationIDKeyContextKey).(string)
	if ok && operationID != "" {
		// Add operationID to the console message
		txtMessage = fmt.Sprintf("%s::OperationID=%s", txtMessage, operationID)
	}
	consoleMessage := fmt.Sprintf("%s::Sev=%v", txtMessage, severity)
	if len(properties) > 0 {
		consoleMessage = fmt.Sprintf("%s::Properties=%v", consoleMessage, properties)
	}

	// If logToConsole is true, print the log message
	if logToConsole {
		log.Println(consoleMessage)
	}

	// Create the new trace
	trace := appinsights.NewTraceTelemetry(txtMessage, contracts.SeverityLevel(severity))
	for k, v := range properties {
		trace.Properties[k] = v
	}

	// Set parent id, using the operationID from the context
	if operationID != "" {
		trace.Tags.Operation().SetParentId(operationID)
	}

	// Send the trace to App Insights
	t.client.Track(trace)
}

// TrackException sends an exception telemetry event
func (t *Telemetry) TrackException(ctx context.Context, message string, err error, severity SeverityLevel, properties map[string]string, logToConsole bool) {
	if t.client == nil {
		panic("Telemetry client not initialized")
	}

	// Create the log message
	txtMessage := fmt.Sprintf("%s::%s", SERVICE_NAME, message)
	// Retrieve the operationID from the context and add it to the log message
	operationID, ok := ctx.Value(appcontext.OperationIDKeyContextKey).(string)
	if ok && operationID != "" {
		// Add operationID to the console message
		txtMessage = fmt.Sprintf("%s::OperationID=%s", txtMessage, operationID)
	}
	consoleMessage := fmt.Sprintf("%s::Error=%s::Sev=%v", txtMessage, err.Error(), severity)
	if len(properties) > 0 {
		consoleMessage = fmt.Sprintf("%s::Properties=%v", consoleMessage, properties)
	}

	// If logToConsole is true, print the log message
	if logToConsole {
		log.Println(consoleMessage)
	}

	// Create the new exception
	exception := appinsights.NewExceptionTelemetry(err)
	exception.SeverityLevel = (contracts.SeverityLevel)(severity)
	for k, v := range properties {
		exception.Properties[k] = v
	}

	// Set parent id, using the operationID from the context
	if operationID != "" {
		exception.Tags.Operation().SetParentId(operationID)
	}

	t.client.Track(exception)
}

// TrackRequest sends a request telemetry event
func (t *Telemetry) TrackRequest(ctx context.Context, method string, url string, duration time.Duration, responseCode string, success bool, source string, properties map[string]string, logToConsole bool) string {
	if t.client == nil {
		panic("Telemetry client not initialized")
	}

	// Create the log message
	consoleMessage := fmt.Sprintf("%s::Method=%s::URL=%s", SERVICE_NAME, method, url)
	if len(properties) > 0 {
		consoleMessage = fmt.Sprintf("%s::Properties=%v", consoleMessage, properties)
	}

	// If logToConsole is true, print the log message
	if logToConsole {
		log.Println(consoleMessage)
	}

	// Create the new request
	request := appinsights.NewRequestTelemetry(method, url, duration, responseCode)
	request.Success = success
	for k, v := range properties {
		request.Properties[k] = v
	}

	// Send the request to App Insights
	t.client.Track(request)

	// Return the operation id
	return request.Tags.Operation().GetId()
}

// TrackDependency sends a dependency telemetry event
func (t *Telemetry) TrackDependency(ctx context.Context, dependencyData string, dependencyName string, dependencyType string, dependencyTarget string, dependencySuccess bool, startTime time.Time, endTime time.Time, properties map[string]string, logToConsole bool) string {
	if t.client == nil {
		panic("Telemetry client not initialized")
	}

	// Create the log message
	txtMessage := fmt.Sprintf("%s::%s::%s", SERVICE_NAME, dependencyData, dependencyName)
	consoleMessage := txtMessage
	if len(properties) > 0 {
		consoleMessage = fmt.Sprintf("%s::Properties=%v", consoleMessage, properties)
	}

	// If logToConsole is true, print the log message
	if logToConsole {
		log.Println(consoleMessage)
	}

	// Create a new dependency
	dependency := appinsights.NewRemoteDependencyTelemetry(txtMessage, dependencyType, dependencyTarget, dependencySuccess)

	dependency.Data = dependencyData
	dependency.MarkTime(startTime, endTime)
	for k, v := range properties {
		dependency.Properties[k] = v
	}

	// Get the operationID from the context
	if operationID, ok := ctx.Value(appcontext.OperationIDKeyContextKey).(string); ok {
		// Set parent id
		if operationID != "" {
			dependency.Tags.Operation().SetParentId(operationID)
		}
	}

	// Send the dependency to App Insights
	t.client.Track(dependency)

	return dependency.Tags.Operation().GetId()
}

// Helper function to retrieve the telemetry client from the context
func GetTelemetryClient(ctx context.Context) *Telemetry {
	telemetryClient, ok := ctx.Value(appcontext.TelemetryContextKey).(*Telemetry)
	if !ok {
		log.Panic("Telemetry client not found in context")
	}
	return telemetryClient
}
