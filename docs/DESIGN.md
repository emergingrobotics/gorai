# Gorai Dashboard Design

## Current State

The dashboard at `pkg/dashboard/` serves a status page showing:
- Robot name
- Component count
- Camera online/total count
- A component list with name, type/model, and status badge
- Separate tabs: Status, Cameras, AI/Models (if external services exist)

The title is hardcoded to "Gorai Dashboard". The nav brand is "Gorai". Services are not listed on the status page. Cameras get a dedicated separate tab despite being components.

## Requirements

### R-1: Title includes robot name
The HTML `<title>` and the nav brand must include the robot name from `robotCfg.Robot.Name`. Format: `"{RobotName} - Gorai Dashboard"` for the title, `"Gorai - {RobotName}"` for the nav brand.

### R-2: Services listed on the status page
The status page must show a "Services" section listing all configured services (from `robotCfg.Services`), similar to the existing "Components" section. Each service shows:
- Name
- Type / Model
- Status badge: "active" (not disabled), "disabled" (disabled)

### R-3: Services count in summary grid
Add a summary card for services (count of configured services) next to the existing components count card.

### R-4: Cameras are components, not a separate category
Remove the dedicated "Cameras" summary card from the status grid. Cameras appear in the component list like any other component (with their online/offline status from the camera monitor). The Cameras tab remains for the camera stream viewer -- that is specialized UI for viewing feeds, not a status listing.

### R-5: Remove cameras count from summary
Replace the three-card grid (Robot, Components, Cameras) with a three-card grid (Robot, Components, Services). Camera components appear in the components list with online/offline status from the camera monitor.

## Implementation

### File: `pkg/dashboard/handlers.go`

#### `handleIndex` changes

1. **Title**: Change `<title>Gorai Dashboard</title>` to `<title>{RobotName} - Gorai Dashboard</title>` using `html.EscapeString(d.robotCfg.Robot.Name)`.

2. **Nav brand**: Change `<div class="nav-brand">Gorai</div>` to `<div class="nav-brand">Gorai - {RobotName}</div>`.

3. **Summary grid**: Replace the "Cameras" card with a "Services" card showing `len(d.robotCfg.Services)` count.

4. **Services section**: After the Components section, add a "Services" section iterating over `d.robotCfg.Services`. Each service rendered as a `component-item` div with:
   - Name
   - Type / Model (same format as components)
   - Status badge: "active" if not disabled, "disabled" if disabled

5. **Component status**: Keep the existing camera online check for camera-type components. Non-camera components show "active" or "disabled" as before.

#### `handleStatus` changes

The `/api/status` JSON endpoint already includes `Services` count. No change needed.

### File: `pkg/dashboard/static/css/main.css`

Add `.camera-status.active` style (currently only online/offline/connecting/error exist):

```css
.camera-status.active {
    background: var(--success-bg);
    color: var(--success-text);
}

.camera-status.disabled {
    background: var(--bg-header);
    color: var(--text-muted);
}
```

### No new files needed

All changes are to existing files. No new templates, JS, or CSS files required.

## Test Plan

1. Verify `<title>` includes robot name
2. Verify nav brand shows "Gorai - {RobotName}"
3. Verify services section appears with correct services
4. Verify summary grid shows Robot, Components, Services (no Cameras card)
5. Verify camera components still show online/offline status in component list
6. Verify disabled components and services show "disabled" badge
7. Verify `/api/status` JSON still returns correct data
8. Verify `/health` endpoint still works
9. Verify Cameras tab still works for stream viewing
