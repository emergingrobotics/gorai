#!/bin/bash
# gorai wrapper script - runs gorai CLI in a container
#
# Install this script as /usr/local/bin/gorai on the host to run
# gorai commands without installing Go or podman-compose.
#
# Requirements:
#   - podman installed on host
#   - podman socket enabled: systemctl --user enable --now podman.socket
#
# Usage:
#   gorai start --config robot.json --build --detach
#   gorai stop --config robot.json
#   gorai status --config robot.json
#   gorai logs --config robot.json --follow

set -e

# Container image to use
GORAI_IMAGE="${GORAI_IMAGE:-ghcr.io/gorai/gorai:latest}"

# Detect podman socket location
if [[ -S "/run/podman/podman.sock" ]]; then
    # Root podman socket
    PODMAN_SOCK="/run/podman/podman.sock"
elif [[ -S "/run/user/$(id -u)/podman/podman.sock" ]]; then
    # Rootless podman socket
    PODMAN_SOCK="/run/user/$(id -u)/podman/podman.sock"
else
    echo "Error: Podman socket not found." >&2
    echo "Enable it with: systemctl --user enable --now podman.socket" >&2
    exit 1
fi

# Run gorai in container
exec podman run --rm -it \
    --security-opt label=disable \
    -v "${PODMAN_SOCK}:/run/podman/podman.sock:rw" \
    -v "$(pwd):/workspace:rw" \
    -w /workspace \
    -e "GORAI_ROBOT_NAME=${GORAI_ROBOT_NAME:-}" \
    -e "HOME=/workspace" \
    "$GORAI_IMAGE" "$@"
