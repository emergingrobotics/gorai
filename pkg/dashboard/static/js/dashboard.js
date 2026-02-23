// Dashboard component toggle and WebSocket status updates.
(function() {
    'use strict';

    // Click handler for component toggle buttons
    document.addEventListener('click', function(event) {
        var button = event.target.closest('.component-toggle');
        if (!button) {
            return;
        }

        var componentName = button.getAttribute('data-component');
        var currentState = button.getAttribute('data-state');
        if (!componentName || !currentState) {
            return;
        }

        var newCommand = (currentState === 'on') ? 'off' : 'on';

        // Set pending state immediately
        button.className = 'camera-status pending component-toggle';
        button.textContent = newCommand + '...';

        fetch('/api/components/' + encodeURIComponent(componentName) + '/command', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({command: newCommand})
        }).then(function(response) {
            if (!response.ok) {
                revertButton(button, currentState);
            }
        }).catch(function() {
            revertButton(button, currentState);
        });
    });

    function revertButton(button, state) {
        var cssClass = (state === 'on') ? 'online' : 'offline';
        button.className = 'camera-status ' + cssClass + ' component-toggle';
        button.textContent = state;
        button.setAttribute('data-state', state);
    }

    // Sanitize a string for safe use in CSS attribute selectors
    function escapeCSSSelector(str) {
        return str.replace(/["\\]/g, '\\$&');
    }

    // WebSocket connection with exponential backoff reconnect
    var reconnectDelay = 1000;
    var maxReconnectDelay = 30000;

    function connectWebSocket() {
        var wsProtocol = (location.protocol === 'https:') ? 'wss:' : 'ws:';
        var ws = new WebSocket(wsProtocol + '//' + location.host + '/ws');

        ws.onopen = function() {
            reconnectDelay = 1000;
        };

        ws.onmessage = function(event) {
            var msg;
            try {
                msg = JSON.parse(event.data);
            } catch (e) {
                return;
            }

            if (msg.type === 'service_status') {
                var svcSelector = '[data-service="' + escapeCSSSelector(msg.service) + '"]';
                var badges = document.querySelectorAll(svcSelector);
                for (var j = 0; j < badges.length; j++) {
                    badges[j].textContent = msg.status_value || 'active';
                    badges[j].className = 'camera-status online';
                }
                return;
            }

            if (msg.type !== 'component_status') {
                return;
            }
            if (msg.value_type !== 'binary') {
                return;
            }

            var state = String(msg.value);
            var cssClass = (state === 'on') ? 'online' : 'offline';

            // Update all toggle buttons for this component (may appear multiple times)
            var selector = '.component-toggle[data-component="' + escapeCSSSelector(msg.component) + '"]';
            var buttons = document.querySelectorAll(selector);
            for (var i = 0; i < buttons.length; i++) {
                buttons[i].setAttribute('data-state', state);
                buttons[i].textContent = state;
                buttons[i].className = 'camera-status ' + cssClass + ' component-toggle';
            }
        };

        ws.onclose = function() {
            setTimeout(function() {
                connectWebSocket();
            }, reconnectDelay);
            reconnectDelay = Math.min(reconnectDelay * 2, maxReconnectDelay);
        };
    }

    connectWebSocket();
})();
