package sub

// QoS defines quality of service levels for subscribers.
type QoS int

const (
	// BestEffort uses core NATS subscription.
	// May miss messages if subscriber is slow or temporarily disconnected.
	BestEffort QoS = iota

	// Reliable uses JetStream consumer with acknowledgment.
	// Messages are redelivered if not acknowledged.
	Reliable

	// Durable uses JetStream with a durable consumer.
	// Subscription state persists across reconnects.
	Durable
)

// String returns the string representation of QoS level.
func (q QoS) String() string {
	switch q {
	case BestEffort:
		return "BestEffort"
	case Reliable:
		return "Reliable"
	case Durable:
		return "Durable"
	default:
		return "Unknown"
	}
}
