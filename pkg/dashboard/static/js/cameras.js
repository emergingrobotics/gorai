// Gorai Dashboard - Camera Monitor JavaScript

// Camera status WebSocket
let cameraWs = null;
let reconnectAttempts = 0;
const maxReconnectAttempts = 10;

function connectCameraWebSocket() {
    const wsProtocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${wsProtocol}//${location.host}/ws/cameras`;

    cameraWs = new WebSocket(wsUrl);

    cameraWs.onopen = () => {
        console.log('Camera WebSocket connected');
        reconnectAttempts = 0;
    };

    cameraWs.onmessage = (event) => {
        try {
            const status = JSON.parse(event.data);
            if (status.type === 'camera_status') {
                updateCameraStatus(status);
            }
        } catch (e) {
            console.error('Failed to parse camera status:', e);
        }
    };

    cameraWs.onclose = () => {
        console.log('Camera WebSocket disconnected');
        if (reconnectAttempts < maxReconnectAttempts) {
            reconnectAttempts++;
            const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
            console.log(`Reconnecting in ${delay}ms (attempt ${reconnectAttempts})`);
            setTimeout(connectCameraWebSocket, delay);
        }
    };

    cameraWs.onerror = (error) => {
        console.error('Camera WebSocket error:', error);
    };
}

function updateCameraStatus(status) {
    const card = document.querySelector(`[data-camera="${status.camera}"]`);
    if (!card) return;

    const statusEl = card.querySelector('.camera-status');
    if (!statusEl) return;

    if (status.online) {
        statusEl.className = 'camera-status online';
        statusEl.textContent = `${status.fps.toFixed(1)} fps`;
    } else {
        statusEl.className = 'camera-status offline';
        statusEl.textContent = 'offline';

        // Show offline placeholder
        const img = card.querySelector('.camera-feed');
        if (img && !img.dataset.paused) {
            showOfflinePlaceholder(card);
        }
    }
}

function toggleStream(cameraName) {
    const card = document.querySelector(`[data-camera="${cameraName}"]`);
    if (!card) return;

    const img = card.querySelector('.camera-feed');
    const btn = card.querySelector('.camera-controls button');
    if (!img || !btn) return;

    if (img.dataset.paused === 'true') {
        // Resume stream
        img.src = `/cameras/${cameraName}/stream`;
        img.dataset.paused = 'false';
        btn.textContent = 'Pause';
        img.style.display = 'block';

        // Remove offline placeholder if exists
        const placeholder = card.querySelector('.camera-offline');
        if (placeholder) {
            placeholder.remove();
        }
    } else {
        // Pause stream
        img.dataset.paused = 'true';
        img.src = '';
        btn.textContent = 'Resume';

        // Show paused placeholder
        showPausedPlaceholder(card, img);
    }
}

function showOfflinePlaceholder(card) {
    const feedContainer = card.querySelector('.camera-feed');
    if (!feedContainer) return;

    // Check if placeholder already exists
    if (card.querySelector('.camera-offline')) return;

    feedContainer.style.display = 'none';

    const placeholder = document.createElement('div');
    placeholder.className = 'camera-offline';
    placeholder.innerHTML = `
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M16.5 9.4l-9-5.19M21 16V8a2 2 0 00-1-1.73l-7-4a2 2 0 00-2 0l-7 4A2 2 0 003 8v8a2 2 0 001 1.73l7 4a2 2 0 002 0l7-4A2 2 0 0021 16z"/>
            <polyline points="3.27 6.96 12 12.01 20.73 6.96"/>
            <line x1="12" y1="22.08" x2="12" y2="12"/>
            <line x1="1" y1="1" x2="23" y2="23" stroke-width="2"/>
        </svg>
        <p>Camera Offline</p>
    `;

    feedContainer.parentNode.insertBefore(placeholder, feedContainer.nextSibling);
}

function showPausedPlaceholder(card, img) {
    // Check if placeholder already exists
    if (card.querySelector('.camera-offline')) return;

    img.style.display = 'none';

    const placeholder = document.createElement('div');
    placeholder.className = 'camera-offline';
    placeholder.innerHTML = `
        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="6" y="4" width="4" height="16"/>
            <rect x="14" y="4" width="4" height="16"/>
        </svg>
        <p>Stream Paused</p>
    `;

    img.parentNode.insertBefore(placeholder, img.nextSibling);
}

function setLayout(layout) {
    const grid = document.getElementById('camera-grid');
    if (!grid) return;

    // Remove all layout classes
    grid.classList.remove('grid-1x1', 'grid-2x2', 'grid-3x3');

    // Add new layout class
    grid.classList.add(`grid-${layout}`);

    // Save preference
    localStorage.setItem('gorai-camera-layout', layout);
}

function loadLayoutPreference() {
    const saved = localStorage.getItem('gorai-camera-layout');
    if (saved) {
        setLayout(saved);
        const select = document.getElementById('layout-select');
        if (select) {
            select.value = saved;
        }
    }
}

// Handle image load errors
function handleImageError(img, cameraName) {
    img.onerror = null; // Prevent infinite loop
    const card = img.closest('.camera-card');
    if (card) {
        showOfflinePlaceholder(card);
    }
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    connectCameraWebSocket();
    loadLayoutPreference();

    // Set up error handlers for all camera feeds
    document.querySelectorAll('.camera-feed').forEach(img => {
        const cameraName = img.closest('[data-camera]')?.dataset.camera;
        if (cameraName) {
            img.onerror = () => handleImageError(img, cameraName);
        }
    });
});
