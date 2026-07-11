package mesh

import "testing"

// capabilityChannelFromSchema is what closes the channels-vs-schemas gap:
// RegisterSchema uses it to auto-register a discovery channel for capability
// subjects while leaving shared/type schemas as schema-only.
func TestCapabilityChannelFromSchema(t *testing.T) {
	cases := []struct {
		name    string
		schema  string
		version string
		wantOK  bool
		wantDir Direction
		wantBot string
	}{
		{"command is a tool (sub)", "gorai.picarx.drive.command", "1", true, DirectionSub, "picarx"},
		{"data is a resource (pub)", "gorai.picarx.battery.data", "1", true, DirectionPub, "picarx"},
		{"state is a resource (pub)", "gorai.picarx.sysinfo.state", "1", true, DirectionPub, "picarx"},
		{"event (pub)", "gorai.picarx.cliff.event", "1", true, DirectionPub, "picarx"},
		{"dotted capability name", "gorai.picarx.arm.joint.command", "1", true, DirectionSub, "picarx"},
		{"shared type schema is skipped", "gorai.sensor.IMUReading", "v1", false, "", ""},
		{"unknown trailing type is skipped", "gorai.picarx.drive.foo", "1", false, "", ""},
		{"too few parts is skipped", "gorai.picarx.data", "1", false, "", ""},
		{"non-gorai prefix is skipped", "other.picarx.drive.command", "1", false, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ch, ok := capabilityChannelFromSchema(SchemaDescriptor{Name: tc.schema, Version: tc.version})
			if ok != tc.wantOK {
				t.Fatalf("ok=%v, want %v", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if ch.Subject != tc.schema {
				t.Errorf("subject=%q, want %q", ch.Subject, tc.schema)
			}
			if ch.Direction != tc.wantDir {
				t.Errorf("direction=%q, want %q", ch.Direction, tc.wantDir)
			}
			if ch.RobotID != tc.wantBot {
				t.Errorf("robot=%q, want %q", ch.RobotID, tc.wantBot)
			}
			if want := SchemaKey(tc.schema, tc.version); ch.Schema != want {
				t.Errorf("schema key=%q, want %q", ch.Schema, want)
			}
		})
	}
}
