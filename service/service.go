// Package service defines the base interfaces for Gorai services.
//
// Services represent software capabilities such as vision processing,
// ML inference, SLAM, and navigation. Unlike components which abstract
// hardware, services abstract algorithms and processing pipelines.
package service

import "context"

// Service is the base interface for all services.
type Service interface {
	// Name returns the service's unique name.
	Name() string

	// Reconfigure updates the service with new configuration.
	Reconfigure(ctx context.Context, conf map[string]any) error

	// Close releases all resources held by the service.
	Close(ctx context.Context) error
}
