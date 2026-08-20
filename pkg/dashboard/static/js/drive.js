// Gorai Dashboard - Drive Control
//
// Manual teleop for a drive component: a throttle bar sets the speed authority
// (0..1) and four direction arrows (press-and-hold) select the motion. The
// browser computes a normalized {surge, yaw} intent and streams it at ~10Hz
// while any arrow is held, sending {0,0} on release (surf owns the failsafe).

const KEEPALIVE_MS = 100;

async function drivePostForm(url, params) {
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

function setupDrive(el) {
    const name = el.dataset.name;
    const throttle = el.querySelector('[data-role="throttle"]');
    const throttleVal = el.querySelector('[data-role="throttle-val"]');
    const readout = el.querySelector('[data-role="readout"]');
    const status = el.querySelector('[data-role="status"]');
    const armBtn = el.querySelector('[data-role="arm"]');
    const stopBtn = el.querySelector('[data-role="stop"]');
    const arrows = el.querySelectorAll('.drive-arrow');

    // Which directions are currently held.
    const held = { up: false, down: false, left: false, right: false };
    let timer = null;
    let lastSent = null;

    function authority() {
        return parseFloat(throttle.value) / 100;
    }

    function computeIntent() {
        const a = authority();
        let surge = 0;
        let yaw = 0;
        if (held.up) surge += a;
        if (held.down) surge -= a;
        if (held.right) yaw += a;
        if (held.left) yaw -= a;
        surge = Math.max(-1, Math.min(1, surge));
        yaw = Math.max(-1, Math.min(1, yaw));
        return { surge, yaw };
    }

    function render(state) {
        if (state && typeof state.left === 'number' && typeof state.right === 'number') {
            readout.textContent = `L ${state.left.toFixed(2)} / R ${state.right.toFixed(2)}`;
        }
        if (state && typeof state.armed === 'boolean') {
            el.classList.toggle('armed', state.armed);
        }
    }

    async function sendIntent(surge, yaw) {
        try {
            const state = await drivePostForm(`/api/drive/${encodeURIComponent(name)}/intent`, { surge, yaw });
            render(state);
        } catch (e) {
            console.error(`Drive ${name} intent failed:`, e);
        }
    }

    function tick() {
        const { surge, yaw } = computeIntent();
        // Always send while active; this doubles as the keepalive.
        sendIntent(surge, yaw);
        lastSent = { surge, yaw };
    }

    function anyHeld() {
        return held.up || held.down || held.left || held.right;
    }

    function startLoop() {
        if (timer !== null) return;
        tick();
        timer = setInterval(tick, KEEPALIVE_MS);
    }

    function stopLoop() {
        if (timer !== null) {
            clearInterval(timer);
            timer = null;
        }
        // Command neutral on release.
        sendIntent(0, 0);
    }

    function press(dir) {
        if (held[dir]) return;
        held[dir] = true;
        startLoop();
    }

    function release(dir) {
        if (!held[dir]) return;
        held[dir] = false;
        if (!anyHeld()) stopLoop();
    }

    arrows.forEach((btn) => {
        const dir = btn.dataset.dir;
        btn.addEventListener('pointerdown', (e) => {
            e.preventDefault();
            btn.setPointerCapture?.(e.pointerId);
            btn.classList.add('active');
            press(dir);
        });
        const up = () => { btn.classList.remove('active'); release(dir); };
        btn.addEventListener('pointerup', up);
        btn.addEventListener('pointercancel', up);
        btn.addEventListener('pointerleave', up);
    });

    // Keyboard control (arrow keys / WASD) when the widget is on screen.
    const keyMap = {
        ArrowUp: 'up', ArrowDown: 'down', ArrowLeft: 'left', ArrowRight: 'right',
        w: 'up', s: 'down', a: 'left', d: 'right',
    };
    window.addEventListener('keydown', (e) => {
        const dir = keyMap[e.key];
        if (!dir || e.repeat) return;
        e.preventDefault();
        press(dir);
    });
    window.addEventListener('keyup', (e) => {
        const dir = keyMap[e.key];
        if (!dir) return;
        e.preventDefault();
        release(dir);
    });

    // Release everything if focus leaves the page (safety).
    window.addEventListener('blur', () => {
        Object.keys(held).forEach((dir) => { held[dir] = false; });
        stopLoop();
    });

    throttle.addEventListener('input', () => {
        throttleVal.textContent = `${throttle.value}%`;
    });

    function setControlsDisabled(disabled) {
        armBtn.disabled = disabled;
        stopBtn.disabled = disabled;
        arrows.forEach((b) => { b.disabled = disabled; });
    }

    armBtn.addEventListener('click', async () => {
        setControlsDisabled(true);
        if (status) status.textContent = 'Arming...';
        try {
            const state = await drivePostForm(`/api/drive/${encodeURIComponent(name)}/arm`, {});
            render(state);
            if (status) status.textContent = 'Armed';
        } catch (e) {
            console.error(`Drive ${name} arm failed:`, e);
            if (status) status.textContent = `Arm failed: ${e.message}`;
        } finally {
            setControlsDisabled(false);
        }
    });

    stopBtn.addEventListener('click', async () => {
        Object.keys(held).forEach((dir) => { held[dir] = false; });
        stopLoop();
        try {
            const state = await drivePostForm(`/api/drive/${encodeURIComponent(name)}/stop`, {});
            render(state);
            if (status) status.textContent = 'Stopped';
        } catch (e) {
            console.error(`Drive ${name} stop failed:`, e);
        }
    });
}

document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('.drive-control').forEach(setupDrive);
});
