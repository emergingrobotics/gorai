// Package tf provides coordinate frame transformations.
package tf

import (
	"fmt"
	"sync"
	"time"
)

// Vector3 represents a 3D vector.
type Vector3 struct {
	X, Y, Z float64
}

// Quaternion represents a rotation.
type Quaternion struct {
	X, Y, Z, W float64
}

// Transform represents a coordinate transformation.
type Transform struct {
	Translation Vector3
	Rotation    Quaternion
}

// StampedTransform is a Transform with timing information.
type StampedTransform struct {
	Transform
	Timestamp    time.Time
	FrameID      string
	ChildFrameID string
}

// Buffer stores and looks up transforms.
type Buffer struct {
	mu         sync.RWMutex
	transforms map[string]map[string][]StampedTransform // parent -> child -> transforms
	bufferTime time.Duration
}

// NewBuffer creates a new transform buffer.
func NewBuffer(bufferTime time.Duration) *Buffer {
	return &Buffer{
		transforms: make(map[string]map[string][]StampedTransform),
		bufferTime: bufferTime,
	}
}

// SetTransform adds a transform to the buffer.
func (b *Buffer) SetTransform(t StampedTransform) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.transforms[t.FrameID] == nil {
		b.transforms[t.FrameID] = make(map[string][]StampedTransform)
	}

	transforms := b.transforms[t.FrameID][t.ChildFrameID]

	// Remove old transforms
	cutoff := time.Now().Add(-b.bufferTime)
	filtered := transforms[:0]
	for _, tr := range transforms {
		if tr.Timestamp.After(cutoff) {
			filtered = append(filtered, tr)
		}
	}

	b.transforms[t.FrameID][t.ChildFrameID] = append(filtered, t)
}

// LookupTransform finds the transform between two frames.
func (b *Buffer) LookupTransform(targetFrame, sourceFrame string, when time.Time) (StampedTransform, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Direct lookup
	if children, ok := b.transforms[targetFrame]; ok {
		if transforms, ok := children[sourceFrame]; ok {
			// Find closest transform to requested time
			var closest StampedTransform
			minDiff := time.Duration(1<<63 - 1)
			for _, t := range transforms {
				diff := t.Timestamp.Sub(when)
				if diff < 0 {
					diff = -diff
				}
				if diff < minDiff {
					minDiff = diff
					closest = t
				}
			}
			return closest, nil
		}
	}

	return StampedTransform{}, fmt.Errorf("transform from %s to %s not found", sourceFrame, targetFrame)
}

// CanTransform checks if a transform is available.
func (b *Buffer) CanTransform(targetFrame, sourceFrame string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if children, ok := b.transforms[targetFrame]; ok {
		_, ok := children[sourceFrame]
		return ok
	}
	return false
}

// Identity returns the identity quaternion.
func Identity() Quaternion {
	return Quaternion{W: 1}
}
