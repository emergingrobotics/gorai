// Gorai Dashboard - IMU Calibration & Orientation
//
// Configures a remote AHRS over NCP: a Calibrate button zeroes orientation by
// averaging ~2s of readings, three axis dropdowns set the mounting remap, and
// three inputs set a hardcoded offset. All state comes back from the server so
// the widget always reflects the vehicle's live frame configuration.
//
// The live readout is polled at 1Hz, but the editable controls (mounting
// dropdowns, offset inputs) are only synced from the server on initial load and
// in response to the user's own actions. Polling must never clobber them, or an
// in-flight edit would be reset to the vehicle's current value before Apply.

async function imuPostForm(url, params) {
    const body = Object.entries(params)
        .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(v)}`)
        .join('&');
    const res = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body,
    });
    if (!res.ok) {
        throw new Error(`${res.status}: ${await res.text()}`);
    }
    return res.json();
}

function setupImu(el) {
    const name = el.dataset.name;
    const readout = el.querySelector('[data-role="readout"]');
    const zeroed = el.querySelector('[data-role="zeroed"]');
    const status = el.querySelector('[data-role="status"]');
    const calibrateBtn = el.querySelector('[data-role="calibrate"]');
    const clearBtn = el.querySelector('[data-role="clear"]');
    const mountX = el.querySelector('[data-role="mount-x"]');
    const mountY = el.querySelector('[data-role="mount-y"]');
    const mountZ = el.querySelector('[data-role="mount-z"]');
    const applyMounting = el.querySelector('[data-role="apply-mounting"]');
    const offRoll = el.querySelector('[data-role="offset-roll"]');
    const offPitch = el.querySelector('[data-role="offset-pitch"]');
    const offYaw = el.querySelector('[data-role="offset-yaw"]');
    const applyOffset = el.querySelector('[data-role="apply-offset"]');

    const num = (v, d = 1) => (typeof v === 'number' ? v.toFixed(d) : '--');

    // renderReadout updates only the live orientation display; safe to call on
    // every poll.
    function renderReadout(state) {
        if (!state) return;
        readout.textContent = `R ${num(state.roll)} / P ${num(state.pitch)} / Y ${num(state.yaw)}`;
        if (typeof state.zeroed === 'boolean') {
            zeroed.textContent = state.zeroed ? 'calibrated' : 'offset';
            el.classList.toggle('zeroed', state.zeroed);
        }
    }

    // renderConfig syncs the editable controls to the server. Only call this on
    // initial load and after the user's own actions, never from the poll.
    function renderConfig(state) {
        if (!state) return;
        if (state.mounting) {
            if (state.mounting.x) mountX.value = state.mounting.x;
            if (state.mounting.y) mountY.value = state.mounting.y;
            if (state.mounting.z) mountZ.value = state.mounting.z;
        }
        if (state.offset_deg) {
            if (state.offset_deg.roll !== undefined) offRoll.value = state.offset_deg.roll;
            if (state.offset_deg.pitch !== undefined) offPitch.value = state.offset_deg.pitch;
            if (state.offset_deg.yaw !== undefined) offYaw.value = state.offset_deg.yaw;
        }
    }

    // applyState updates everything from an authoritative action response.
    function applyState(state) {
        renderReadout(state);
        renderConfig(state);
    }

    function disable(d) {
        [calibrateBtn, clearBtn, applyMounting, applyOffset].forEach((b) => { b.disabled = d; });
    }

    calibrateBtn.addEventListener('click', async () => {
        disable(true);
        status.textContent = 'Calibrating (hold still)...';
        try {
            applyState(await imuPostForm(`/api/imu/${encodeURIComponent(name)}/calibrate`, {}));
            status.textContent = 'Calibrated';
        } catch (e) {
            status.textContent = `Calibrate failed: ${e.message}`;
        } finally {
            disable(false);
        }
    });

    clearBtn.addEventListener('click', async () => {
        try {
            applyState(await imuPostForm(`/api/imu/${encodeURIComponent(name)}/clear`, {}));
            status.textContent = 'Zero cleared';
        } catch (e) {
            status.textContent = `Clear failed: ${e.message}`;
        }
    });

    applyMounting.addEventListener('click', async () => {
        try {
            applyState(await imuPostForm(`/api/imu/${encodeURIComponent(name)}/mounting`, {
                x: mountX.value, y: mountY.value, z: mountZ.value,
            }));
            status.textContent = 'Mounting applied';
        } catch (e) {
            status.textContent = `Mounting failed: ${e.message}`;
        }
    });

    applyOffset.addEventListener('click', async () => {
        try {
            applyState(await imuPostForm(`/api/imu/${encodeURIComponent(name)}/offset`, {
                roll_deg: offRoll.value || 0,
                pitch_deg: offPitch.value || 0,
                yaw_deg: offYaw.value || 0,
            }));
            status.textContent = 'Offset applied';
        } catch (e) {
            status.textContent = `Offset failed: ${e.message}`;
        }
    });

    return { name, renderReadout, renderConfig };
}

document.addEventListener('DOMContentLoaded', () => {
    const widgets = [];
    document.querySelectorAll('.imu-control').forEach((el) => widgets.push(setupImu(el)));
    if (widgets.length === 0) return;

    async function poll(includeConfig) {
        try {
            const res = await fetch('/api/imu');
            if (!res.ok) return;
            const list = await res.json();
            const byName = {};
            list.forEach((s) => { byName[s.name] = s; });
            widgets.forEach((wdg) => {
                const s = byName[wdg.name];
                if (!s) return;
                wdg.renderReadout(s);
                if (includeConfig) wdg.renderConfig(s);
            });
        } catch (e) {
            // ignore transient errors
        }
    }

    // Initial load syncs the editable controls once; thereafter poll the live
    // readout only so in-progress edits are never clobbered.
    poll(true);
    setInterval(() => poll(false), 1000);
});
