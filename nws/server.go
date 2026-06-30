package nws

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/emergingrobotics/gorai/pkg/resource"
	"github.com/nats-io/nats.go"
)

// Request is a remote method call request.
type Request struct {
	Method string         `json:"method"`
	Args   map[string]any `json:"args,omitempty"`
}

// Response is a remote method call response.
type Response struct {
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

// ResourceServer wraps a resource and exposes it over NATS.
type ResourceServer struct {
	nc       *nats.Conn
	resource resource.Resource
	subject  string
	sub      *nats.Subscription
	methods  map[string]reflect.Method
}

// blockedRPCMethods are methods that must never be exposed over unauthenticated NATS RPC.
var blockedRPCMethods = map[string]bool{
	"Close":       true,
	"Reconfigure": true,
}

// Wrap creates a new ResourceServer that exposes the resource over NATS.
// Only methods in allowedMethods are exposed. If allowedMethods is empty,
// all public methods except blocked administrative methods are exposed.
func Wrap(nc *nats.Conn, res resource.Resource, allowedMethods ...string) (*ResourceServer, error) {
	if nc == nil {
		return nil, fmt.Errorf("NATS connection is required")
	}

	subject := res.Name().Subject() + ".rpc"

	s := &ResourceServer{
		nc:       nc,
		resource: res,
		subject:  subject,
		methods:  make(map[string]reflect.Method),
	}

	if len(allowedMethods) > 0 {
		// Only register explicitly allowed methods
		allowed := make(map[string]bool, len(allowedMethods))
		for _, name := range allowedMethods {
			allowed[name] = true
		}
		resType := reflect.TypeOf(res)
		for i := 0; i < resType.NumMethod(); i++ {
			method := resType.Method(i)
			if allowed[method.Name] {
				s.methods[method.Name] = method
			}
		}
	} else {
		// Register all public methods except blocked ones
		resType := reflect.TypeOf(res)
		for i := 0; i < resType.NumMethod(); i++ {
			method := resType.Method(i)
			if blockedRPCMethods[method.Name] {
				continue
			}
			s.methods[method.Name] = method
		}
	}

	// Subscribe to RPC requests
	sub, err := nc.Subscribe(subject, s.handleRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}
	s.sub = sub

	return s, nil
}

// handleRequest processes incoming RPC requests.
func (s *ResourceServer) handleRequest(msg *nats.Msg) {
	var req Request
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		s.respond(msg, Response{Error: fmt.Sprintf("invalid request: %v", err)})
		return
	}

	result, err := s.invoke(req.Method, req.Args)
	if err != nil {
		s.respond(msg, Response{Error: err.Error()})
		return
	}

	s.respond(msg, Response{Result: result})
}

// invoke calls a method on the resource using reflection.
func (s *ResourceServer) invoke(methodName string, args map[string]any) (any, error) {
	method, ok := s.methods[methodName]
	if !ok {
		return nil, fmt.Errorf("method not found: %s", methodName)
	}

	// Get method type info
	methodType := method.Type
	numIn := methodType.NumIn()

	// Build arguments
	callArgs := make([]reflect.Value, numIn)
	callArgs[0] = reflect.ValueOf(s.resource) // receiver

	// First non-receiver arg is usually context.Context
	if numIn > 1 {
		ctx := context.Background()
		callArgs[1] = reflect.ValueOf(ctx)
	}

	// Handle additional arguments based on method signature
	for i := 2; i < numIn; i++ {
		paramType := methodType.In(i)
		paramName := fmt.Sprintf("arg%d", i-1)

		if argVal, ok := args[paramName]; ok {
			// Convert the argument to the expected type
			converted, err := convertArg(argVal, paramType)
			if err != nil {
				return nil, fmt.Errorf("arg %s: %w", paramName, err)
			}
			callArgs[i] = converted
		} else {
			// Use zero value
			callArgs[i] = reflect.Zero(paramType)
		}
	}

	// Call the method
	results := method.Func.Call(callArgs)

	// Process results
	if len(results) == 0 {
		return nil, nil
	}

	// Check for error in last result
	lastResult := results[len(results)-1]
	if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
		if !lastResult.IsNil() {
			return nil, lastResult.Interface().(error)
		}
	}

	// Return non-error results
	if len(results) == 1 {
		if lastResult.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			return nil, nil
		}
		return results[0].Interface(), nil
	}

	// Multiple results (excluding error)
	resultMap := make(map[string]any)
	for i := 0; i < len(results)-1; i++ {
		resultMap[fmt.Sprintf("result%d", i)] = results[i].Interface()
	}
	return resultMap, nil
}

// convertArg converts a JSON-decoded value to the expected type.
func convertArg(val any, targetType reflect.Type) (reflect.Value, error) {
	if val == nil {
		return reflect.Zero(targetType), nil
	}

	valType := reflect.TypeOf(val)
	if valType.AssignableTo(targetType) {
		return reflect.ValueOf(val), nil
	}

	// Handle common type conversions
	switch targetType.Kind() {
	case reflect.Float64:
		switch v := val.(type) {
		case float64:
			return reflect.ValueOf(v), nil
		case int:
			return reflect.ValueOf(float64(v)), nil
		case int64:
			return reflect.ValueOf(float64(v)), nil
		}
	case reflect.Int:
		switch v := val.(type) {
		case float64:
			return reflect.ValueOf(int(v)), nil
		case int:
			return reflect.ValueOf(v), nil
		case int64:
			return reflect.ValueOf(int(v)), nil
		}
	case reflect.String:
		if s, ok := val.(string); ok {
			return reflect.ValueOf(s), nil
		}
	case reflect.Bool:
		if b, ok := val.(bool); ok {
			return reflect.ValueOf(b), nil
		}
	}

	return reflect.Zero(targetType), fmt.Errorf("cannot convert %T to %s", val, targetType)
}

// respond sends a response to the client.
func (s *ResourceServer) respond(msg *nats.Msg, resp Response) {
	if msg.Reply == "" {
		return
	}

	data, err := json.Marshal(resp)
	if err != nil {
		data = []byte(`{"error":"failed to marshal response"}`)
	}

	s.nc.Publish(msg.Reply, data)
}

// Subject returns the NATS subject for this server.
func (s *ResourceServer) Subject() string {
	return s.subject
}

// Resource returns the wrapped resource.
func (s *ResourceServer) Resource() resource.Resource {
	return s.resource
}

// Close stops the server and unsubscribes from NATS.
func (s *ResourceServer) Close() error {
	if s.sub != nil {
		return s.sub.Unsubscribe()
	}
	return nil
}

// NATSGetter is an interface for types that provide a NATS connection.
type NATSGetter interface {
	NATS() *nats.Conn
}

// WrapWithNode creates a ResourceServer using a node's NATS connection.
func WrapWithNode(n NATSGetter, res resource.Resource) (*ResourceServer, error) {
	nc := n.NATS()
	if nc == nil {
		return nil, fmt.Errorf("node has no NATS connection")
	}
	return Wrap(nc, res)
}

// HealthCheck publishes periodic health status for the resource.
type HealthCheck struct {
	server   *ResourceServer
	interval time.Duration
	cancel   context.CancelFunc
}

// StartHealthCheck begins publishing health status.
func (s *ResourceServer) StartHealthCheck(interval time.Duration) *HealthCheck {
	ctx, cancel := context.WithCancel(context.Background())

	hc := &HealthCheck{
		server:   s,
		interval: interval,
		cancel:   cancel,
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				status := map[string]any{
					"name":      s.resource.Name().String(),
					"timestamp": time.Now().Unix(),
					"status":    "healthy",
				}
				data, _ := json.Marshal(status)
				s.nc.Publish(s.subject+".health", data)
			}
		}
	}()

	return hc
}

// Stop stops the health check.
func (hc *HealthCheck) Stop() {
	if hc.cancel != nil {
		hc.cancel()
	}
}
