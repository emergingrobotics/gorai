package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strconv"

	"github.com/emergingrobotics/gorai/components/pwm"
	"github.com/go-chi/chi/v5"
)

// pwmInfo describes a PWM component for the control UI.
type pwmInfo struct {
	Name           string  `json:"name"`
	Model          string  `json:"model"`
	FrequencyHz    float64 `json:"frequency_hz"`
	MinPulseUs     float64 `json:"min_pulse_us"`
	MaxPulseUs     float64 `json:"max_pulse_us"`
	InitialPulseUs float64 `json:"initial_pulse_us"`
	PulseUs        float64 `json:"pulse_us"`
	Enabled        bool    `json:"enabled"`
}

// pwmComponents returns the configured PWM components with their attribute
// defaults from the RDL.
func (d *Dashboard) pwmComponents() []pwmInfo {
	var out []pwmInfo
	for _, comp := range d.robotCfg.Components {
		if comp.Type != "pwm" || comp.Disabled {
			continue
		}
		info := pwmInfo{
			Name:           comp.Name,
			Model:          comp.Model,
			FrequencyHz:    attrFloat(comp.Attributes, "frequency_hz", 50),
			MinPulseUs:     attrFloat(comp.Attributes, "min_pulse_us", 1000),
			MaxPulseUs:     attrFloat(comp.Attributes, "max_pulse_us", 2000),
			InitialPulseUs: attrFloat(comp.Attributes, "initial_pulse_us", 1500),
		}
		info.PulseUs = info.InitialPulseUs
		out = append(out, info)
	}
	return out
}

// attrFloat reads a float attribute with a fallback default.
func attrFloat(attrs map[string]any, key string, def float64) float64 {
	if attrs == nil {
		return def
	}
	if v, ok := attrs[key].(float64); ok {
		return v
	}
	return def
}

// lookupPWM resolves a live PWM component by name.
func (d *Dashboard) lookupPWM(name string) (pwm.PWM, error) {
	if d.componentGetter == nil {
		return nil, fmt.Errorf("component control not available")
	}
	comp, ok := d.componentGetter(name)
	if !ok {
		return nil, fmt.Errorf("component %q not found", name)
	}
	p, ok := comp.(pwm.PWM)
	if !ok {
		return nil, fmt.Errorf("component %q is not a PWM", name)
	}
	return p, nil
}

// handleControl serves the PWM control page with a slider per component.
func (d *Dashboard) handleControl(w http.ResponseWriter, r *http.Request) {
	infos := d.pwmComponents()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Gorai Control</title>
    <link rel="stylesheet" href="/static/css/main.css">
</head>
<body>
    <nav class="nav">
        <div class="nav-brand">Gorai</div>
        <ul class="nav-tabs">
            <li><a href="/">Status</a></li>
            <li><a href="/cameras">Cameras</a></li>
            <li><a href="/control" class="active">Control</a></li>
        </ul>
    </nav>
    <main>
        <div class="status-card">
            <h3>PWM Control</h3>`))

	if len(infos) == 0 {
		w.Write([]byte(`<p>No PWM components configured.</p>`))
	}

	for _, info := range infos {
		safeName := html.EscapeString(info.Name)
		w.Write([]byte(fmt.Sprintf(`
            <div class="pwm-control" data-name="%s" data-min="%g" data-max="%g" data-initial="%g">
                <div class="pwm-header">
                    <span class="pwm-name">%s</span>
                    <span class="pwm-model">%s</span>
                    <span class="pwm-readout" data-role="readout">%g us</span>
                </div>
                <input type="range" data-role="slider" min="%g" max="%g" step="1" value="%g">
                <div class="pwm-buttons">
                    <button data-role="arm">Arm</button>
                    <button data-role="neutral">Neutral (%g us)</button>
                    <button data-role="enable">Enable</button>
                    <button data-role="disable">Disable</button>
                </div>
                <div class="pwm-arm-status" data-role="arm-status"></div>
            </div>`,
			safeName, info.MinPulseUs, info.MaxPulseUs, info.InitialPulseUs,
			safeName, html.EscapeString(info.Model), info.InitialPulseUs,
			info.MinPulseUs, info.MaxPulseUs, info.InitialPulseUs,
			info.InitialPulseUs)))
	}

	w.Write([]byte(`
        </div>
    </main>
    <script src="/static/js/pwm.js"></script>
</body>
</html>
`))
}

// handlePWMList returns the PWM components with current live state when available.
func (d *Dashboard) handlePWMList(w http.ResponseWriter, r *http.Request) {
	infos := d.pwmComponents()
	for i := range infos {
		if p, err := d.lookupPWM(infos[i].Name); err == nil {
			if props, err := p.Properties(r.Context()); err == nil {
				if props.FrequencyHz > 0 {
					infos[i].FrequencyHz = props.FrequencyHz
				}
				if props.MinPulseUs > 0 {
					infos[i].MinPulseUs = props.MinPulseUs
				}
				if props.MaxPulseUs > 0 {
					infos[i].MaxPulseUs = props.MaxPulseUs
				}
			}
			if pulse, err := p.GetPulse(r.Context()); err == nil {
				infos[i].PulseUs = pulse
			}
			if enabled, err := p.IsEnabled(r.Context()); err == nil {
				infos[i].Enabled = enabled
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(infos)
}

// handlePWMSetPulse sets the pulse width for a PWM component.
func (d *Dashboard) handlePWMSetPulse(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	usStr := r.FormValue("us")
	us, err := strconv.ParseFloat(usStr, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid us value: %v", err), http.StatusBadRequest)
		return
	}

	p, err := d.lookupPWM(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	ctx := r.Context()
	if err := p.SetPulse(ctx, us); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d.writePWMState(w, ctx, name, p)
}

// handlePWMEnable enables or disables a PWM component.
func (d *Dashboard) handlePWMEnable(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	on := r.FormValue("on") == "true"

	p, err := d.lookupPWM(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	ctx := r.Context()
	if on {
		err = p.Enable(ctx)
	} else {
		err = p.Disable(ctx)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d.writePWMState(w, ctx, name, p)
}

// handlePWMArm runs the arming sequence for a PWM component. This blocks until
// the (multi-second) sequence completes.
func (d *Dashboard) handlePWMArm(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	p, err := d.lookupPWM(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	ctx := r.Context()
	if err := p.Arm(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d.writePWMState(w, ctx, name, p)
}

// writePWMState responds with the current PWM state as JSON.
func (d *Dashboard) writePWMState(w http.ResponseWriter, ctx context.Context, name string, p pwm.PWM) {
	pulse, _ := p.GetPulse(ctx)
	enabled, _ := p.IsEnabled(ctx)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"ok":       true,
		"name":     name,
		"pulse_us": pulse,
		"enabled":  enabled,
	})
}
