package service_test

import (
	"testing"

	"github.com/gorai/gorai/pkg/resource"
	"github.com/gorai/gorai/services"
)

// TestService_IsResource verifies that Service embeds resource.Resource.
func TestService_IsResource(t *testing.T) {
	// This is a compile-time check that Service embeds resource.Resource.
	// If this compiles, the interface relationship is correct.
	var _ resource.Resource = (service.Service)(nil)
}
