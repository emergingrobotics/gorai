// Gorai Dashboard - Telemetry
//
// Subscribes to the dashboard WebSocket and renders always-on telemetry panels
// (GPS, IMU, environment) forwarded from the vehicle by the NATS->WS bridge.
// Frames are tagged {type:"telemetry", topic, data}; other /ws traffic is
// ignored.

const FIX_LABELS = { 0: 'no fix', 1: 'GPS', 2: 'DGPS', 4: 'RTK fix', 5: 'RTK float' };

function fmt(v, digits) {
    if (typeof v !== 'number' || Number.isNaN(v)) return '--';
    return v.toFixed(digits);
}

function setField(panel, field, value) {
    const el = document.querySelector(`.telemetry-card[data-panel="${panel}"] [data-field="${field}"]`);
    if (el) el.textContent = value;
}

function renderGPS(d) {
    if ('fix' in d || 'fix_status' in d) {
        const fix = d.fix ?? d.fix_status;
        setField('gps', 'fix', typeof fix === 'string' ? fix : (FIX_LABELS[fix] ?? String(fix)));
    }
    if ('latitude' in d) setField('gps', 'latitude', fmt(d.latitude, 6));
    if ('longitude' in d) setField('gps', 'longitude', fmt(d.longitude, 6));
    if ('hdop' in d) setField('gps', 'hdop', fmt(d.hdop, 2));
    if ('satellites' in d) setField('gps', 'satellites', String(d.satellites));
}

function renderIMU(d) {
    if ('yaw' in d) setField('imu', 'yaw', `${fmt(d.yaw, 1)}°`);
    if ('roll' in d) setField('imu', 'roll', `${fmt(d.roll, 1)}°`);
    if ('pitch' in d) setField('imu', 'pitch', `${fmt(d.pitch, 1)}°`);
}

function renderBMP(d) {
    if ('pressure_pa' in d) setField('bmp', 'pressure_pa', `${fmt(d.pressure_pa / 100, 1)} hPa`);
    if ('temperature_c' in d) setField('bmp', 'temperature_c', `${fmt(d.temperature_c, 1)} °C`);
    if ('altitude_m' in d) setField('bmp', 'altitude_m', `${fmt(d.altitude_m, 1)} m`);
}

// Map a component topic to its renderer. Names are matched by substring so
// "gps", "front_gps", etc. all route to the GPS panel.
function renderTopic(topic, data) {
    if (typeof data !== 'object' || data === null) return;
    const t = topic.toLowerCase();
    if (t.includes('gps')) renderGPS(data);
    else if (t.includes('imu') || t.includes('ahrs')) renderIMU(data);
    else if (t.includes('bmp') || t.includes('pressure') || t.includes('env')) renderBMP(data);
}

async function loadSnapshot() {
    try {
        const res = await fetch('/api/telemetry');
        if (!res.ok) return;
        const snap = await res.json();
        for (const [topic, data] of Object.entries(snap)) {
            renderTopic(topic, data);
        }
    } catch (e) {
        console.error('telemetry snapshot failed:', e);
    }
}

function connect() {
    const proto = location.protocol === 'https:' ? 'wss' : 'ws';
    const ws = new WebSocket(`${proto}://${location.host}/ws`);

    ws.addEventListener('message', (ev) => {
        let msg;
        try {
            msg = JSON.parse(ev.data);
        } catch {
            return;
        }
        if (!msg || msg.type !== 'telemetry') return;
        renderTopic(msg.topic, msg.data);
    });

    ws.addEventListener('close', () => {
        // Reconnect with a short backoff.
        setTimeout(connect, 1000);
    });
    ws.addEventListener('error', () => ws.close());
}

document.addEventListener('DOMContentLoaded', () => {
    if (!document.getElementById('telemetry')) return;
    loadSnapshot();
    connect();
});
