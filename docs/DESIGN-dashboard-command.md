# Design: Dashboard Component Command Override

## Problem

The dashboard displays component status (on/off for switches, power values for meters) but provides no way to change component state. Operators must use external tools (curl, NATS CLI) to toggle a switch. The dashboard should let users click a component's status badge to send an on/off command as a manual override.

## Scope

Only `binary` status value types (on/off switches) are clickable. Number and string status types remain display-only.

## Architecture

```
Browser                    Dashboard Server              NATS                  Component
  |                            |                           |                      |
  |  click "on" badge          |                           |                      |
  |  POST /api/components/     |                           |                      |
  |    plug_a/command          |                           |                      |
  |  {"command":"off"}         |                           |                      |
  |--------------------------->|                           |                      |
  |                            |  publish to               |                      |
  |                            |  gorai.<robot>.plug_a     |                      |
  |                            |    .command               |                      |
  |                            |-------------------------->|                      |
  |                            |                           |--------------------->|
  |  200 OK                    |                           |                      |
  |<---------------------------|                           |  component executes  |
  |                            |                           |  HTTP to device,     |
  |  badge turns yellow        |                           |  updates state,      |
  |  (pending)                 |                           |  publishes new       |
  |                            |                           |  readings to .data   |
  |                            |                           |<---------------------|
  |                            |  component monitor        |                      |
  |                            |  receives updated         |                      |
  |                            |  status_value             |                      |
  |                            |<--------------------------|                      |
  |                            |                           |                      |
  |  WebSocket: component      |                           |                      |
  |  status update             |                           |                      |
  |<---------------------------|                           |                      |
  |                            |                           |                      |
  |  badge turns green/red     |                           |                      |
  |  (confirmed new state)     |                           |                      |
```

## Data Flow

### 1. Dashboard HTTP API (new endpoint)

**Endpoint:** `POST /api/components/{name}/command`

**Request body:**
```json
{
  "command": "on"
}
```

or:

```json
{
  "command": "off"
}
```

**Response:** `200 OK` with:
```json
{
  "status": "sent",
  "component": "plug_a",
  "command": "on"
}
```

**Error responses:**
- `400` if body is invalid or command is not `on`/`off`
- `404` if component name doesn't match any configured component
- `503` if NATS is not connected

**Validation:**
- Component name must exist in `robotCfg.Components`
- Component must not be disabled
- Command must be exactly `"on"` or `"off"` (no arbitrary commands from the browser)
- Only components with `status_value_type: "binary"` in the component monitor cache are commandable

### 2. NATS Command Message

The dashboard publishes to `gorai.<robot>.<component>.command`:

```json
{
  "command": "on",
  "source": "dashboard",
  "timestamp": "2026-02-22T12:00:00Z"
}
```

This uses the existing `topics.ComponentCommand(name)` method. The `source` field distinguishes dashboard overrides from light-controller schedule commands.

### 3. Component Command Subscription (new in gorai-tasmota)

Switch components subscribe to their `.command` topic at startup and handle incoming commands by calling their existing DoCommand logic.

**In `baseSwitch.initBase()`**, after setting `b.dataTopic`:

```go
commandTopic := fmt.Sprintf("gorai.%s.%s.command", robotID, nameStr)
b.commandSub, _ = b.natsConn.Subscribe(commandTopic, b.handleNATSCommand)
```

**Handler:**

```go
func (b *baseSwitch) handleNATSCommand(msg *nats.Msg) {
    var cmd map[string]any
    if err := json.Unmarshal(msg.Data, &cmd); err != nil {
        return
    }
    command, _ := cmd["command"].(string)
    if command != "on" && command != "off" {
        return
    }
    // Call the concrete type's DoCommand
    b.doCommandFn(context.Background(), cmd)
}
```

The `doCommandFn` is a function pointer set during construction (NewTasmota, NewKauf, NewShelly) pointing to the concrete DoCommand implementation. This avoids interface gymnastics while reusing the shared subscription logic.

**Cleanup in `baseSwitch.Close()`:**

```go
if b.commandSub != nil {
    b.commandSub.Unsubscribe()
}
```

### 4. Component Monitor WebSocket Broadcast (new)

The component monitor already caches status values. Add a callback mechanism (matching the camera monitor pattern) so that when a status value changes, the dashboard broadcasts the update to all WebSocket clients.

**In `components.Monitor`:**

```go
type StatusChangeCallback func(componentName string, value *ComponentValue)

func (m *Monitor) OnStatusChange(fn StatusChangeCallback) {
    m.mu.Lock()
    m.onStatusChange = fn
    m.mu.Unlock()
}
```

**In `handleDataMessage()`, after updating the cached value:**

```go
if m.onStatusChange != nil {
    m.onStatusChange(componentName, cv)
}
```

**In `dashboard.go` New():**

```go
d.componentMonitor.OnStatusChange(func(name string, cv *components.ComponentValue) {
    d.wsHub.BroadcastJSON(map[string]any{
        "type":       "component_status",
        "component":  name,
        "value":      cv.Value,
        "value_type": cv.ValueType,
        "unit":       cv.Unit,
    })
})
```

### 5. Dashboard Frontend (inline JavaScript in handlers.go)

The status badges for binary components become clickable buttons with three visual states:

| State | CSS Class | Color | Meaning |
|-------|-----------|-------|---------|
| On (confirmed) | `online` | Green | Component is on, confirmed by data |
| Off (confirmed) | `offline` | Gray/Red | Component is off, confirmed by data |
| Pending | `pending` | Pale yellow | Command sent, waiting for confirmation |

**HTML structure change for binary components:**

Currently:
```html
<span class="camera-status online">on</span>
```

Becomes:
```html
<button class="camera-status online component-toggle"
        data-component="plug_a"
        data-state="on">on</button>
```

Only binary status_value components get the `<button>` and `component-toggle` class. Number/string types remain `<span>`.

**JavaScript (added to page footer):**

```javascript
document.addEventListener('click', function(e) {
    var btn = e.target.closest('.component-toggle');
    if (!btn) return;

    var name = btn.getAttribute('data-component');
    var currentState = btn.getAttribute('data-state');
    var newCommand = (currentState === 'on') ? 'off' : 'on';

    // Immediately set pending state
    btn.className = 'camera-status pending component-toggle';
    btn.textContent = newCommand + '...';

    fetch('/api/components/' + encodeURIComponent(name) + '/command', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({command: newCommand})
    }).catch(function() {
        // Revert on network error
        btn.className = 'camera-status ' + (currentState === 'on' ? 'online' : 'offline') + ' component-toggle';
        btn.textContent = currentState;
    });
});
```

**WebSocket listener (updates badge when confirmed state arrives):**

```javascript
var ws = new WebSocket('ws://' + location.host + '/ws');
ws.onmessage = function(e) {
    var msg = JSON.parse(e.data);
    if (msg.type !== 'component_status') return;
    if (msg.value_type !== 'binary') return;

    var buttons = document.querySelectorAll('.component-toggle[data-component="' + msg.component + '"]');
    buttons.forEach(function(btn) {
        var state = String(msg.value);
        btn.setAttribute('data-state', state);
        btn.textContent = state;
        btn.className = 'camera-status ' + (state === 'on' ? 'online' : 'offline') + ' component-toggle';
    });
};
```

A component may appear multiple times on the page (once under its service, once standalone) so the selector finds all matching buttons.

### 6. CSS Addition

Add to `/er/gorai/pkg/dashboard/static/css/main.css`:

```css
.component-toggle {
    cursor: pointer;
    border: none;
    font: inherit;
    transition: background-color 0.2s;
}

.component-toggle:hover {
    opacity: 0.8;
}

.camera-status.pending {
    background-color: #fff3cd;
    color: #856404;
}
```

### 7. Content-Security-Policy Update

The current CSP header is:
```
default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'
```

This blocks inline `<script>` tags. Two options:

**Option A: Add script-src 'unsafe-inline'** -- simple, works, slightly reduces CSP protection.

**Option B: Move JavaScript to a static .js file** and serve from `/static/js/dashboard.js`. CSP stays strict.

**Recommendation: Option B.** Create `/er/gorai/pkg/dashboard/static/js/dashboard.js` with the click handler and WebSocket code. Add `<script src="/static/js/dashboard.js"></script>` to the HTML footer. The CSP remains `script-src 'self'` (default from `default-src 'self'`).

## Files to Modify

### gorai (core framework)

| File | Change |
|------|--------|
| `pkg/dashboard/server.go` | Add `POST /api/components/{name}/command` route |
| `pkg/dashboard/handlers.go` | Add `handleComponentCommand()` handler; update `handleIndex()` to render `<button>` for binary components |
| `pkg/dashboard/dashboard.go` | Wire component monitor OnStatusChange to WebSocket broadcast |
| `pkg/dashboard/components/monitor.go` | Add `OnStatusChange` callback; fire on value update |
| `pkg/dashboard/static/js/dashboard.js` | New file: click handler + WebSocket listener |
| `pkg/dashboard/static/css/main.css` | Add `.component-toggle` and `.pending` styles |
| `pkg/dashboard/server.go` | Update CSP to allow `connect-src 'self' ws:` for WebSocket |

### gorai-tasmota (switch components)

| File | Change |
|------|--------|
| `switch/common.go` | Add `commandSub *nats.Subscription`, `doCommandFn`, subscribe to `.command` topic, handle incoming commands, unsubscribe on Close |
| `switch/tasmota.go` | Set `doCommandFn` in NewTasmota |
| `switch/kauf.go` | Set `doCommandFn` in NewKauf |
| `switch/shelly.go` | Set `doCommandFn` in NewShelly |

## Security Considerations

1. **Command validation**: Only `on` and `off` are accepted. No arbitrary command strings from the browser reach the component.
2. **Component validation**: The handler checks that the named component exists in the RDL and is not disabled.
3. **No authentication**: The dashboard currently has no auth. This design does not add auth. The existing CLAUDE.md warns when binding to `0.0.0.0`. Adding auth is a separate concern.
4. **NATS message source**: The `"source": "dashboard"` field in the command message lets components and the light-controller distinguish manual overrides from scheduled actions. The light-controller could choose to respect or override manual state.
5. **CSP**: JavaScript stays in a static file. No inline scripts. WebSocket connections use `connect-src 'self'` which allows same-origin WebSocket.
6. **Rate limiting**: Not added in this design. The dashboard is a trusted internal tool on a local network. If needed later, Chi middleware can add rate limiting.

## Timeout and Error Handling

- The fetch call has no explicit timeout (browser default applies, typically 300s). Network errors revert the button.
- If the component is offline (device unreachable), the NATS command is still published. The component's retry logic handles device communication. The button stays yellow until the component publishes new data confirming state change.
- If no state confirmation arrives within a reasonable time (device truly dead), the button stays yellow indefinitely. This is acceptable -- it visually indicates "something is wrong." A future enhancement could add a client-side timeout that reverts to the previous state after 30s.

## Testing

1. **Unit test `handleComponentCommand`**: mock NATS publish, verify correct topic and payload, verify 400/404 for bad input.
2. **Unit test `OnStatusChange` callback**: verify callback fires when status_value changes in monitor.
3. **Integration test**: send POST to command endpoint, verify NATS message published on correct topic.
4. **Manual test**: click button on dashboard, observe yellow pending state, observe state change to green/red when component responds.
