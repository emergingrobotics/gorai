package pub

// QoS defines quality of service levels for publishers.
type QoS int

const (
	// BestEffort uses core NATS - no persistence, fire and forget.
	// Messages may be lost if no subscribers are listening.
	BestEffort QoS = iota

	// Reliable uses JetStream with acknowledgment.
	// Messages are persisted and delivery is guaranteed.
	Reliable

	// Retained uses JetStream with last-value retention.
	// Only the most recent message is kept per subject.
	Retained

	// History keeps the last N messages using JetStream.
	// Useful for late-joining subscribers to catch up.
	History
)

// String returns the string representation of QoS level.
func (q QoS) String() string {
	switch q {
	case BestEffort:
		return "BestEffort"
	case Reliable:
		return "Reliable"
	case Retained:
		return "Retained"
	case History:
		return "History"
	default:
		return "Unknown"
	}
}
