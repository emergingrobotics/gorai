package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strconv"

	"github.com/emergingrobotics/gorai/components/drive"
	"github.com/go-chi/chi/v5"
)

// writeDrivePanels renders a manual drive control section (throttle bar + four
// direction arrows) for each configured drive component.
func (d *Dashboard) writeDrivePanels(w http.ResponseWriter, drives []driveInfo) {
	if len(drives) == 0 {
		return
	}
	w.Write([]byte(`
        <div class="status-card">
            <h3>Drive Control</h3>`))
	for _, info := range drives {
		safeName := html.EscapeString(info.Name)
		w.Write([]byte(fmt.Sprintf(`
            <div class="drive-control" data-name="%s">
                <div class="drive-header">
                    <span class="drive-name">%s</span>
                    <span class="drive-model">%s</span>
                    <span class="drive-readout" data-role="readout">L 0.00 / R 0.00</span>
                </div>
                <div class="drive-body">
                    <div class="drive-throttle">
                        <label>Throttle</label>
                        <input type="range" data-role="throttle" min="0" max="100" step="1" value="50" orient="vertical">
                        <span class="drive-throttle-val" data-role="throttle-val">50%%</span>
                    </div>
                    <div class="drive-pad">
                        <button class="drive-arrow" data-dir="up">&#9650;</button>
                        <div class="drive-pad-mid">
                            <button class="drive-arrow" data-dir="left">&#9664;</button>
                            <button class="drive-arrow" data-dir="right">&#9654;</button>
                        </div>
                        <button class="drive-arrow" data-dir="down">&#9660;</button>
                    </div>
                </div>
                <div class="drive-buttons">
                    <button data-role="arm">Arm</button>
                    <button data-role="stop">Stop</button>
                </div>
                <div class="drive-status" data-role="status"></div>
            </div>`,
			safeName, safeName, html.EscapeString(info.Model))))
	}
	w.Write([]byte(`
        </div>`))
}

// driveInfo describes a drive component for the control UI.
type driveInfo struct {
	Name  string `json:"name"`
	Model string `json:"model"`
}

// driveComponents returns the configured drive components from the RDL.
func (d *Dashboard) driveComponents() []driveInfo {
	var out []driveInfo
	for _, comp := range d.robotCfg.Components {
		if comp.Type != "drive" || comp.Disabled {
			continue
		}
		out = append(out, driveInfo{Name: comp.Name, Model: comp.Model})
	}
	return out
}

// lookupDrive resolves a live drive component by name.
func (d *Dashboard) lookupDrive(name string) (drive.Drive, error) {
	if d.componentGetter == nil {
		return nil, fmt.Errorf("component control not available")
	}
	comp, ok := d.componentGetter(name)
	if !ok {
		return nil, fmt.Errorf("component %q not found", name)
	}
	dr, ok := comp.(drive.Drive)
	if !ok {
		return nil, fmt.Errorf("component %q is not a drive", name)
	}
	return dr, nil
}

// handleDriveList returns the configured drive components with live state.
func (d *Dashboard) handleDriveList(w http.ResponseWriter, r *http.Request) {
	infos := d.driveComponents()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(infos)
}

// handleDriveIntent sets the normalized surge/yaw intent for a drive component.
func (d *Dashboard) handleDriveIntent(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	surge, err := strconv.ParseFloat(r.FormValue("surge"), 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid surge: %v", err), http.StatusBadRequest)
		return
	}
	yaw, err := strconv.ParseFloat(r.FormValue("yaw"), 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid yaw: %v", err), http.StatusBadRequest)
		return
	}

	dr, err := d.lookupDrive(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	ctx := r.Context()
	if err := dr.SetIntent(ctx, surge, yaw); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d.writeDriveState(w, ctx, name, dr)
}

// handleDriveStop commands a drive component to neutral.
func (d *Dashboard) handleDriveStop(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	dr, err := d.lookupDrive(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	ctx := r.Context()
	if err := dr.Stop(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d.writeDriveState(w, ctx, name, dr)
}

// handleDriveArm runs the arming sequence for a drive component. This blocks
// until the (multi-second) sequence completes.
func (d *Dashboard) handleDriveArm(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	dr, err := d.lookupDrive(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	ctx := r.Context()
	if err := dr.Arm(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d.writeDriveState(w, ctx, name, dr)
}

// writeDriveState responds with the current drive state as JSON.
func (d *Dashboard) writeDriveState(w http.ResponseWriter, ctx context.Context, name string, dr drive.Drive) {
	st, _ := dr.State(ctx)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"ok":     true,
		"name":   name,
		"surge":  st.Surge,
		"yaw":    st.Yaw,
		"left":   st.Left,
		"right":  st.Right,
		"armed":  st.Armed,
		"active": st.Active,
	})
}
