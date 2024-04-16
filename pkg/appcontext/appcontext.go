package appcontext

// OperationIDKey represents the key type for the operation ID in context
type OperationIDKey string
type TelemetryObj string

const (
	// OperationIDKeyContextKey is the key used to store the operation ID in context
	OperationIDKeyContextKey OperationIDKey = "operationID"

	// TelemetryContextKey represents the key type for the telemetry object in context
	TelemetryContextKey TelemetryObj = "telemetry"
)
