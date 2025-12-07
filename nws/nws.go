// Package nws provides Network Wrapper Server/Client for remote resource access.
//
// The NWS/NWC pattern (inspired by YARP) allows resources to be accessed
// transparently whether they are local or remote. A ResourceServer wraps a
// local resource and exposes it over NATS; a ResourceClient provides access
// to a remote resource using the same interface.
//
// # Server Example
//
// Create a server to expose a local motor:
//
//	motor := fake.NewWithName(resource.NewComponentName("gorai", "motor", "left"))
//	server, err := nws.Wrap(nc, motor)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer server.Close()
//
//	// Optional: start health checks
//	hc := server.StartHealthCheck(5 * time.Second)
//	defer hc.Stop()
//
// # Client Example
//
// Connect to a remote motor:
//
//	name := resource.NewComponentName("gorai", "motor", "left")
//	client, err := nws.Connect(nc, name)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Call methods on the remote resource
//	result, err := client.Call(ctx, "GetPosition", nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	position := result.(float64)
//
// # Protocol
//
// The RPC protocol uses JSON-encoded Request and Response messages over
// NATS request-reply. The subject is derived from the resource name:
//
//	<namespace>.<type>.<subtype>.<name>.rpc
//
// For example: gorai.component.motor.left.rpc
//
// Health status is published periodically to:
//
//	<namespace>.<type>.<subtype>.<name>.rpc.health
package nws
