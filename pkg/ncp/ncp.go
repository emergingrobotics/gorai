// Package ncp implements the wire form of the NATS Capability Protocol (NCP):
// local components are exposed as tools on their `.command` subject and their
// snapshots are published on `.state`. A remote robot invokes a capability by
// sending a Request and receiving a Response, with no MCP server in the path.
//
// Subject conventions come from pkg/subjects:
//
//	gorai.<robot_id>.<component>.command  - tool invocation (request/reply)
//	gorai.<robot_id>.<component>.state    - resource snapshot (published)
package ncp

// Request is an NCP command envelope sent to a capability's `.command` subject.
type Request struct {
	// Method is the capability method to invoke (e.g. "set_pulse").
	Method string `json:"method"`

	// Args holds method arguments keyed by name.
	Args map[string]any `json:"args,omitempty"`
}

// Response is the reply envelope returned on a `.command` request.
type Response struct {
	// OK indicates the method executed successfully.
	OK bool `json:"ok"`

	// Error contains the failure reason when OK is false.
	Error string `json:"error,omitempty"`

	// Result carries method return values (e.g. current state).
	Result map[string]any `json:"result,omitempty"`
}
