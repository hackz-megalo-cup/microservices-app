package order

// ==========================================================.
// Event types — define your domain events here.
// These map directly to your Event Storming sticky notes.
// ==========================================================.

const (
	EventOrderCreated     = "order.created"
	EventOrderFailed      = "order.failed"
	EventOrderCompensated = "order.compensated"
)

// ==========================================================.
// Event data — the payload for each event.
// Add fields that capture what happened.
// ==========================================================.

// OrderCreatedData is the payload for the created event.
type OrderCreatedData struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

// OrderFailedData is the payload for the failed event.
type OrderFailedData struct {
	Input string `json:"input"`
	Error string `json:"error"`
}

// OrderCompensatedData is the payload for the compensated event.
type OrderCompensatedData struct {
	Reason string `json:"reason"`
}
