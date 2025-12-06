// Package param provides parameter storage backed by NATS KV.
package param

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
)

// Store provides parameter storage.
type Store struct {
	kv nats.KeyValue
}

// JetStreamGetter is an interface for types that provide JetStream.
type JetStreamGetter interface {
	JetStream() nats.JetStreamContext
}

// New creates a new parameter store.
func New(n JetStreamGetter, bucket string) (*Store, error) {
	js := n.JetStream()

	kv, err := js.KeyValue(bucket)
	if err != nil {
		// Try to create if it doesn't exist
		kv, err = js.CreateKeyValue(&nats.KeyValueConfig{
			Bucket: bucket,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create KV bucket: %w", err)
		}
	}

	return &Store{kv: kv}, nil
}

// Get retrieves a parameter value.
func (s *Store) Get(ctx context.Context, key string) (any, error) {
	entry, err := s.kv.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get parameter: %w", err)
	}

	var value any
	if err := json.Unmarshal(entry.Value(), &value); err != nil {
		return nil, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return value, nil
}

// Set stores a parameter value.
func (s *Store) Set(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	if _, err := s.kv.Put(key, data); err != nil {
		return fmt.Errorf("failed to set parameter: %w", err)
	}

	return nil
}

// Delete removes a parameter.
func (s *Store) Delete(ctx context.Context, key string) error {
	if err := s.kv.Delete(key); err != nil {
		return fmt.Errorf("failed to delete parameter: %w", err)
	}
	return nil
}

// Keys returns all parameter keys.
func (s *Store) Keys(ctx context.Context) ([]string, error) {
	keys, err := s.kv.Keys()
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}
	return keys, nil
}
