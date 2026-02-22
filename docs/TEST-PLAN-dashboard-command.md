# Test Plan: Dashboard Component Command Override

## Test Strategy

Changes span two repos (gorai core dashboard, gorai-tasmota switches). Each phase has unit tests that must pass before proceeding.

## Phase 1: Component Monitor OnStatusChange

### Unit Tests (in `pkg/dashboard/components/`)

| Test | Description |
|------|-------------|
| `TestOnStatusChange_FiresOnNewValue` | Set callback, send data message, verify callback fires with correct component name and value |
| `TestOnStatusChange_FiresOnValueChange` | Send "on", then "off" — verify callback fires both times |
| `TestOnStatusChange_NotFiredWithoutCallback` | Don't set callback, send data — verify no panic |

## Phase 2: Switch NATS Command Subscription

### Unit Tests (in `gorai-tasmota/switch/`)

| Test | Description |
|------|-------------|
| `TestBaseSwitch_CommandSubscription` | Verify subscription created on `.command` topic during init |
| `TestBaseSwitch_HandleNATSCommand_On` | Publish `{"command":"on"}` to command topic, verify doCommandFn called with correct args |
| `TestBaseSwitch_HandleNATSCommand_Off` | Same for "off" |
| `TestBaseSwitch_HandleNATSCommand_InvalidCommand` | Publish `{"command":"reboot"}`, verify doCommandFn NOT called |
| `TestBaseSwitch_HandleNATSCommand_InvalidJSON` | Publish garbage bytes, verify no panic |
| `TestBaseSwitch_CommandSubCleanup` | Call Close(), verify subscription unsubscribed |

## Phase 3: Dashboard Command HTTP Endpoint

### Unit Tests (in `pkg/dashboard/`)

| Test | Description |
|------|-------------|
| `TestHandleComponentCommand_ValidOn` | POST valid on command, verify 200 and NATS publish to correct topic |
| `TestHandleComponentCommand_ValidOff` | POST valid off command, verify 200 |
| `TestHandleComponentCommand_InvalidCommand` | POST `{"command":"reboot"}`, verify 400 |
| `TestHandleComponentCommand_UnknownComponent` | POST to nonexistent component, verify 404 |
| `TestHandleComponentCommand_DisabledComponent` | POST to disabled component, verify 400 |
| `TestHandleComponentCommand_NoBody` | POST with empty body, verify 400 |
| `TestHandleComponentCommand_NoNATS` | POST when NATS is nil, verify 503 |

## Phase 4: Frontend

Frontend JavaScript is tested via manual verification (no browser test framework in this project).

### Manual Test Checklist

- [ ] Binary component status badges render as `<button>` elements
- [ ] Non-binary status badges render as `<span>` elements (not clickable)
- [ ] Clicking "on" badge sends POST with `{"command":"off"}`
- [ ] Clicking "off" badge sends POST with `{"command":"on"}`
- [ ] Badge turns yellow immediately on click
- [ ] Badge updates to green/red when WebSocket confirms new state
- [ ] Multiple instances of same component on page all update
- [ ] Network error reverts badge to previous state

## Phase 5: Integration

| Test | Description |
|------|-------------|
| Full build of gorai dashboard package | `go build ./pkg/dashboard/...` |
| Full build of gorai-tasmota switch package | `go build ./switch/...` |
| Full build of gograi-cj-robot binary | `make build` in gograi-cj-robot |
| All gorai tests pass | `go test ./pkg/dashboard/...` |
| All gorai-tasmota tests pass | `go test ./switch/...` |
