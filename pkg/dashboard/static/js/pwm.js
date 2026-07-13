// Gorai Dashboard - PWM Control
//
// Wires each .pwm-control widget's slider and buttons to the dashboard PWM API.
// Slider input is debounced to avoid flooding the actuator with commands.

const DEBOUNCE_MS = 50;

async function postForm(url, params) {
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

function setupControl(el) {
    const name = el.dataset.name;
    const initial = parseFloat(el.dataset.initial);
    const slider = el.querySelector('[data-role="slider"]');
    const readout = el.querySelector('[data-role="readout"]');
    const neutralBtn = el.querySelector('[data-role="neutral"]');
    const enableBtn = el.querySelector('[data-role="enable"]');
    const disableBtn = el.querySelector('[data-role="disable"]');
    const armBtn = el.querySelector('[data-role="arm"]');
    const armStatus = el.querySelector('[data-role="arm-status"]');

    let pending = null;

    function render(state) {
        if (state && typeof state.pulse_us === 'number') {
            readout.textContent = `${Math.round(state.pulse_us)} us`;
            slider.value = state.pulse_us;
        }
        if (state && typeof state.enabled === 'boolean') {
            el.classList.toggle('enabled', state.enabled);
        }
    }

    async function sendPulse(us) {
        try {
            const state = await postForm(`/api/pwm/${encodeURIComponent(name)}/pulse`, { us });
            render(state);
        } catch (e) {
            console.error(`PWM ${name} set pulse failed:`, e);
        }
    }

    async function sendEnable(on) {
        try {
            const state = await postForm(`/api/pwm/${encodeURIComponent(name)}/enable`, { on });
            render(state);
        } catch (e) {
            console.error(`PWM ${name} enable failed:`, e);
        }
    }

    slider.addEventListener('input', () => {
        const us = parseFloat(slider.value);
        readout.textContent = `${Math.round(us)} us`;
        if (pending) clearTimeout(pending);
        pending = setTimeout(() => sendPulse(us), DEBOUNCE_MS);
    });

    function setControlsDisabled(disabled) {
        slider.disabled = disabled;
        neutralBtn.disabled = disabled;
        enableBtn.disabled = disabled;
        disableBtn.disabled = disabled;
        armBtn.disabled = disabled;
    }

    async function sendArm() {
        setControlsDisabled(true);
        if (armStatus) armStatus.textContent = 'Arming...';
        try {
            const state = await postForm(`/api/pwm/${encodeURIComponent(name)}/arm`, {});
            render(state);
            if (armStatus) armStatus.textContent = 'Armed';
        } catch (e) {
            console.error(`PWM ${name} arm failed:`, e);
            if (armStatus) armStatus.textContent = `Arm failed: ${e.message}`;
        } finally {
            setControlsDisabled(false);
        }
    }

    neutralBtn.addEventListener('click', () => {
        slider.value = initial;
        readout.textContent = `${Math.round(initial)} us`;
        sendPulse(initial);
    });
    enableBtn.addEventListener('click', () => sendEnable(true));
    disableBtn.addEventListener('click', () => sendEnable(false));
    armBtn.addEventListener('click', sendArm);
}

async function refreshState() {
    try {
        const res = await fetch('/api/pwm');
        if (!res.ok) return;
        const infos = await res.json();
        for (const info of infos) {
            const el = document.querySelector(`.pwm-control[data-name="${CSS.escape(info.name)}"]`);
            if (!el) continue;
            const slider = el.querySelector('[data-role="slider"]');
            const readout = el.querySelector('[data-role="readout"]');
            if (typeof info.pulse_us === 'number') {
                slider.value = info.pulse_us;
                readout.textContent = `${Math.round(info.pulse_us)} us`;
            }
            el.classList.toggle('enabled', !!info.enabled);
        }
    } catch (e) {
        console.error('PWM state refresh failed:', e);
    }
}

document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('.pwm-control').forEach(setupControl);
    refreshState();
});
