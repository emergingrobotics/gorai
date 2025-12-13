// Gorai Dashboard - AI / Models JavaScript

// Model status WebSocket
let modelWs = null;
let reconnectAttempts = 0;
const maxReconnectAttempts = 10;

function connectModelWebSocket() {
    const wsProtocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${wsProtocol}//${location.host}/ws/models`;

    modelWs = new WebSocket(wsUrl);

    modelWs.onopen = () => {
        console.log('Model WebSocket connected');
        reconnectAttempts = 0;
    };

    modelWs.onmessage = (event) => {
        try {
            const data = JSON.parse(event.data);
            if (data.type === 'model_status') {
                updateModelStatus(data.model);
            } else if (data.type === 'detection') {
                addDetectionEvent(data.detection);
            }
        } catch (e) {
            console.error('Failed to parse model message:', e);
        }
    };

    modelWs.onclose = () => {
        console.log('Model WebSocket disconnected');
        if (reconnectAttempts < maxReconnectAttempts) {
            reconnectAttempts++;
            const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
            console.log(`Reconnecting in ${delay}ms (attempt ${reconnectAttempts})`);
            setTimeout(connectModelWebSocket, delay);
        }
    };

    modelWs.onerror = (error) => {
        console.error('Model WebSocket error:', error);
    };
}

function updateModelStatus(model) {
    const card = document.querySelector(`[data-model="${model.name}"]`);
    if (!card) return;

    // Update status badge
    const statusEl = card.querySelector('.camera-status');
    if (statusEl) {
        statusEl.className = 'camera-status ' + getStatusClass(model.status);
        statusEl.textContent = model.status;
    }

    // Update metrics
    const metrics = card.querySelectorAll('.metric');
    if (metrics.length >= 4) {
        metrics[0].querySelector('.metric-value').textContent = model.fps.toFixed(1);
        metrics[1].querySelector('.metric-value').textContent = model.inference_ms.toFixed(1);
        metrics[2].querySelector('.metric-value').textContent = model.frames_processed;
        metrics[3].querySelector('.metric-value').textContent = model.total_detections;
    }

    // Update uptime
    const uptimeEl = card.querySelector('.model-uptime');
    if (uptimeEl) {
        uptimeEl.textContent = 'Uptime: ' + formatUptime(model.uptime_seconds);
    }
}

function getStatusClass(status) {
    switch (status) {
        case 'running':
            return 'online';
        case 'error':
            return 'error';
        default:
            return 'offline';
    }
}

function formatUptime(seconds) {
    const hours = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    const secs = Math.floor(seconds % 60);

    if (hours > 0) {
        return `${hours}h ${mins}m ${secs}s`;
    }
    if (mins > 0) {
        return `${mins}m ${secs}s`;
    }
    return `${secs}s`;
}

function addDetectionEvent(detection) {
    const container = document.getElementById('detections-container');
    if (!container) return;

    // Remove loading message if present
    const loading = container.querySelector('.loading');
    if (loading) {
        loading.remove();
    }

    // Create detection item
    const item = document.createElement('div');
    item.className = 'detection-item';

    const detectionCount = detection.detections ? detection.detections.length : 0;
    const classes = detection.detections ?
        [...new Set(detection.detections.map(d => d.class_name))].join(', ') :
        'none';

    item.innerHTML = `
        <div class="detection-header">
            <span class="detection-time">${formatTimestamp(detection.timestamp)}</span>
            <span class="detection-count">${detectionCount} detection${detectionCount !== 1 ? 's' : ''}</span>
        </div>
        <div class="detection-details">
            <span class="detection-classes">${classes}</span>
            <span class="detection-metrics">${detection.inference_time_ms?.toFixed(1) || '?'}ms</span>
        </div>
    `;

    // Insert at top
    container.insertBefore(item, container.firstChild);

    // Limit displayed items
    while (container.children.length > 50) {
        container.removeChild(container.lastChild);
    }
}

function formatTimestamp(timestamp) {
    if (!timestamp) return '--:--:--';
    const date = new Date(timestamp);
    return date.toLocaleTimeString();
}

// Load recent detections
async function loadRecentDetections() {
    const container = document.getElementById('detections-container');
    if (!container) return;

    try {
        const response = await fetch('/models/detections?limit=20');
        if (!response.ok) throw new Error('Failed to fetch detections');

        const detections = await response.json();

        // Remove loading message
        const loading = container.querySelector('.loading');
        if (loading) {
            loading.remove();
        }

        if (!detections || detections.length === 0) {
            container.innerHTML = '<p class="no-data">No recent detections</p>';
            return;
        }

        // Add detections (they come newest first)
        detections.forEach(detection => {
            addDetectionEvent(detection);
        });
    } catch (error) {
        console.error('Failed to load detections:', error);
        const loading = container.querySelector('.loading');
        if (loading) {
            loading.textContent = 'Failed to load detections';
        }
    }
}

// Toggle annotated stream view
function toggleAnnotatedStream(modelName) {
    const card = document.querySelector(`[data-model="${modelName}"]`);
    if (!card) return;

    let streamContainer = card.querySelector('.model-stream');

    if (streamContainer) {
        // Remove stream
        streamContainer.remove();
    } else {
        // Add stream
        streamContainer = document.createElement('div');
        streamContainer.className = 'model-stream';
        streamContainer.innerHTML = `
            <img src="/models/${modelName}/stream" alt="${modelName} annotated output" class="annotated-feed">
        `;
        card.querySelector('.model-metrics').after(streamContainer);
    }
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    connectModelWebSocket();
    loadRecentDetections();
});
