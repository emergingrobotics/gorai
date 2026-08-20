package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"time"

	"github.com/emergingrobotics/gorai/components/sensor"
	"github.com/go-chi/chi/v5"
)

// imuControl is the interface an AHRS component must satisfy to be configured
// from the dashboard.
type imuControl interface {
	sensor.AHRS
	sensor.OrientationConfigurable
}

// imuInfo describes an AHRS component for the control UI.
type imuInfo struct {
	Name      string          `json:"name"`
	Model     string          `json:"model"`
	Mounting  sensor.Mounting `json:"mounting"`
	OffsetDeg [3]float64      `json:"offset_deg"`
}

// axisOptions are the selectable signed IMU axes for a mounting remap.
var axisOptions = []string{"+x", "-x", "+y", "-y", "+z", "-z"}

// imuComponents returns the configured AHRS components with mounting/offset
// defaults from the RDL for initial render.
func (d *Dashboard) imuComponents() []imuInfo {
	var out []imuInfo
	for _, comp := range d.robotCfg.Components {
		if comp.Type != "ahrs" || comp.Disabled {
			continue
		}
		info := imuInfo{
			Name:      comp.Name,
			Model:     comp.Model,
			Mounting:  sensor.Mounting{X: "+x", Y: "+y", Z: "+z"},
			OffsetDeg: [3]float64{},
		}
		if m, ok := comp.Attributes["mounting"].(map[string]any); ok {
			if s, ok := m["x"].(string); ok && s != "" {
				info.Mounting.X = s
			}
			if s, ok := m["y"].(string); ok && s != "" {
				info.Mounting.Y = s
			}
			if s, ok := m["z"].(string); ok && s != "" {
				info.Mounting.Z = s
			}
		}
		if o, ok := comp.Attributes["offset"].(map[string]any); ok {
			info.OffsetDeg[0] = attrFloat(o, "roll_deg", 0)
			info.OffsetDeg[1] = attrFloat(o, "pitch_deg", 0)
			info.OffsetDeg[2] = attrFloat(o, "yaw_deg", 0)
		}
		out = append(out, info)
	}
	return out
}

// lookupIMU resolves a live AHRS component by name.
func (d *Dashboard) lookupIMU(name string) (imuControl, error) {
	if d.componentGetter == nil {
		return nil, fmt.Errorf("component control not available")
	}
	comp, ok := d.componentGetter(name)
	if !ok {
		return nil, fmt.Errorf("component %q not found", name)
	}
	a, ok := comp.(imuControl)
	if !ok {
		return nil, fmt.Errorf("component %q is not a configurable AHRS", name)
	}
	return a, nil
}

// writeImuPanels renders a calibration + reorientation control section for each
// configured AHRS component.
func (d *Dashboard) writeImuPanels(w http.ResponseWriter) {
	imus := d.imuComponents()
	if len(imus) == 0 {
		return
	}
	w.Write([]byte(`
        <div class="status-card">
            <h3>IMU Calibration &amp; Orientation</h3>`))
	for _, info := range imus {
		safeName := html.EscapeString(info.Name)
		w.Write([]byte(fmt.Sprintf(`
            <div class="imu-control" data-name="%s">
                <div class="imu-header">
                    <span class="imu-name">%s</span>
                    <span class="imu-model">%s</span>
                    <span class="imu-readout" data-role="readout">R 0.0 / P 0.0 / Y 0.0</span>
                    <span class="imu-zeroed" data-role="zeroed">offset</span>
                </div>
                <div class="imu-section">
                    <button data-role="calibrate">Calibrate (zero)</button>
                    <button data-role="clear">Clear zero</button>
                </div>
                <div class="imu-section imu-mounting">
                    <label>Mounting (body = IMU axis)</label>
                    <span>X</span>%s
                    <span>Y</span>%s
                    <span>Z</span>%s
                    <button data-role="apply-mounting">Apply</button>
                </div>
                <div class="imu-section imu-offset">
                    <label>Offset (deg)</label>
                    <span>Roll</span><input type="number" data-role="offset-roll" step="1" value="%g">
                    <span>Pitch</span><input type="number" data-role="offset-pitch" step="1" value="%g">
                    <span>Yaw</span><input type="number" data-role="offset-yaw" step="1" value="%g">
                    <button data-role="apply-offset">Apply</button>
                </div>
                <div class="imu-status" data-role="status"></div>
            </div>`,
			safeName, safeName, html.EscapeString(info.Model),
			axisSelect("mount-x", info.Mounting.X),
			axisSelect("mount-y", info.Mounting.Y),
			axisSelect("mount-z", info.Mounting.Z),
			info.OffsetDeg[0], info.OffsetDeg[1], info.OffsetDeg[2])))
	}
	w.Write([]byte(`
        </div>`))
}

// axisSelect renders a mounting axis <select> with the given option selected.
func axisSelect(role, selected string) string {
	out := fmt.Sprintf(`<select data-role="%s">`, role)
	for _, opt := range axisOptions {
		sel := ""
		if opt == selected {
			sel = " selected"
		}
		out += fmt.Sprintf(`<option value="%s"%s>%s</option>`, opt, sel, opt)
	}
	out += `</select>`
	return out
}

// handleImuList returns the configured AHRS components with live state.
func (d *Dashboard) handleImuList(w http.ResponseWriter, r *http.Request) {
	infos := d.imuComponents()
	out := make([]map[string]any, 0, len(infos))
	for _, info := range infos {
		entry := map[string]any{"name": info.Name, "model": info.Model}
		if a, err := d.lookupIMU(info.Name); err == nil {
			d.fillImuState(r.Context(), a, entry)
		}
		out = append(out, entry)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

// handleImuCalibrate runs a blocking calibration for an AHRS component.
func (d *Dashboard) handleImuCalibrate(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	a, err := d.lookupIMU(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	dur := 2 * time.Second
	if v, err := strconv.ParseFloat(r.FormValue("duration_ms"), 64); err == nil && v > 0 {
		dur = time.Duration(v) * time.Millisecond
	}
	if err := a.Calibrate(r.Context(), dur); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d.writeImuState(w, r.Context(), name, a)
}

// handleImuClear clears a calibrated zero, reverting to the offset.
func (d *Dashboard) handleImuClear(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	a, err := d.lookupIMU(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := a.ClearZero(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d.writeImuState(w, r.Context(), name, a)
}

// handleImuMounting sets the axis-remap mounting for an AHRS component.
func (d *Dashboard) handleImuMounting(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	a, err := d.lookupIMU(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	m := sensor.Mounting{X: r.FormValue("x"), Y: r.FormValue("y"), Z: r.FormValue("z")}
	if err := a.SetMounting(r.Context(), m); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	d.writeImuState(w, r.Context(), name, a)
}

// handleImuOffset sets the hardcoded orientation offset for an AHRS component.
func (d *Dashboard) handleImuOffset(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	a, err := d.lookupIMU(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	roll, _ := strconv.ParseFloat(r.FormValue("roll_deg"), 64)
	pitch, _ := strconv.ParseFloat(r.FormValue("pitch_deg"), 64)
	yaw, _ := strconv.ParseFloat(r.FormValue("yaw_deg"), 64)
	if err := a.SetOffset(r.Context(), roll, pitch, yaw); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d.writeImuState(w, r.Context(), name, a)
}

// fillImuState populates entry with the current orientation + frame config.
func (d *Dashboard) fillImuState(ctx context.Context, a imuControl, entry map[string]any) {
	const rad2deg = 180.0 / 3.141592653589793
	roll, pitch, yaw, _ := a.GetEulerAngles(ctx)
	oc, _ := a.OrientationConfig(ctx)
	entry["roll"] = roll * rad2deg
	entry["pitch"] = pitch * rad2deg
	entry["yaw"] = yaw * rad2deg
	entry["zeroed"] = oc.Zeroed
	entry["mounting"] = map[string]string{"x": oc.Mounting.X, "y": oc.Mounting.Y, "z": oc.Mounting.Z}
	entry["offset_deg"] = map[string]float64{"roll": oc.OffsetDeg[0], "pitch": oc.OffsetDeg[1], "yaw": oc.OffsetDeg[2]}
}

// writeImuState responds with the current AHRS state as JSON.
func (d *Dashboard) writeImuState(w http.ResponseWriter, ctx context.Context, name string, a imuControl) {
	entry := map[string]any{"ok": true, "name": name}
	d.fillImuState(ctx, a, entry)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entry)
}
