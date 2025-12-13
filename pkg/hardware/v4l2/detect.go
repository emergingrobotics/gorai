// Package v4l2 provides V4L2 (Video4Linux2) device detection and information.
package v4l2

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// DeviceInfo contains information about a V4L2 video device.
type DeviceInfo struct {
	// Path is the device path (e.g., "/dev/video0")
	Path string
	// Name is the device name from the driver
	Name string
	// Driver is the driver name
	Driver string
	// BusInfo is the bus location
	BusInfo string
	// Capabilities are the device capabilities
	Capabilities []string
	// Formats are the supported video formats
	Formats []string
	// Index is the device index (0, 1, 2, etc.)
	Index int
}

// DetectResult represents the result of device detection.
type DetectResult struct {
	// Found indicates if the device was found
	Found bool
	// Device contains device information if found
	Device *DeviceInfo
	// Error contains error details if not found
	Error string
}

// DetectDevice checks if a V4L2 device exists and returns information about it.
func DetectDevice(devicePath string) *DetectResult {
	result := &DetectResult{
		Found: false,
	}

	// Check if device exists
	info, err := os.Stat(devicePath)
	if os.IsNotExist(err) {
		result.Error = fmt.Sprintf("device %s does not exist", devicePath)
		return result
	}
	if err != nil {
		result.Error = fmt.Sprintf("cannot access device %s: %v", devicePath, err)
		return result
	}

	// Check if it's a character device (V4L2 devices are char devices)
	if info.Mode()&os.ModeCharDevice == 0 {
		result.Error = fmt.Sprintf("%s is not a character device", devicePath)
		return result
	}

	// Try to get device information
	deviceInfo := &DeviceInfo{
		Path: devicePath,
	}

	// Extract device index from path
	if idx := extractDeviceIndex(devicePath); idx >= 0 {
		deviceInfo.Index = idx
	}

	// Try to read device info from sysfs
	if err := readSysfsInfo(devicePath, deviceInfo); err != nil {
		// Not fatal - we can still use the device
		deviceInfo.Name = "Unknown V4L2 Device"
	}

	// Try to check if device can be opened
	file, err := os.OpenFile(devicePath, os.O_RDWR, 0)
	if err != nil {
		// Try read-only
		file, err = os.OpenFile(devicePath, os.O_RDONLY, 0)
		if err != nil {
			result.Error = fmt.Sprintf("cannot open device %s: %v", devicePath, err)
			return result
		}
	}
	file.Close()

	result.Found = true
	result.Device = deviceInfo
	return result
}

// ListDevices returns a list of all V4L2 video devices.
func ListDevices() ([]*DeviceInfo, error) {
	var devices []*DeviceInfo

	// Look for /dev/video* devices
	matches, err := filepath.Glob("/dev/video*")
	if err != nil {
		return nil, fmt.Errorf("failed to list video devices: %w", err)
	}

	for _, path := range matches {
		result := DetectDevice(path)
		if result.Found {
			devices = append(devices, result.Device)
		}
	}

	return devices, nil
}

// extractDeviceIndex extracts the numeric index from a device path.
func extractDeviceIndex(path string) int {
	re := regexp.MustCompile(`video(\d+)$`)
	matches := re.FindStringSubmatch(path)
	if len(matches) == 2 {
		if idx, err := strconv.Atoi(matches[1]); err == nil {
			return idx
		}
	}
	return -1
}

// readSysfsInfo reads device information from sysfs.
func readSysfsInfo(devicePath string, info *DeviceInfo) error {
	// Get the device index
	idx := extractDeviceIndex(devicePath)
	if idx < 0 {
		return fmt.Errorf("cannot determine device index")
	}

	// Try to read from /sys/class/video4linux/video{N}/
	sysPath := fmt.Sprintf("/sys/class/video4linux/video%d", idx)

	// Read device name
	if name, err := readSysfsFile(filepath.Join(sysPath, "name")); err == nil {
		info.Name = name
	}

	// Try to read from uevent for more details
	if uevent, err := readUevent(filepath.Join(sysPath, "uevent")); err == nil {
		if driver, ok := uevent["DRIVER"]; ok {
			info.Driver = driver
		}
	}

	// Check device capabilities by looking at index file
	if indexData, err := readSysfsFile(filepath.Join(sysPath, "index")); err == nil {
		info.Capabilities = append(info.Capabilities, "index:"+indexData)
	}

	return nil
}

// readSysfsFile reads a single value from a sysfs file.
func readSysfsFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// readUevent reads uevent file and returns key-value pairs.
func readUevent(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	result := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if parts := strings.SplitN(line, "=", 2); len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result, scanner.Err()
}

// IsVideoCapture checks if the device supports video capture.
// This is a basic check - for full capability checking, use v4l2 ioctls.
func IsVideoCapture(devicePath string) bool {
	// Basic check - if we can open it and it's a video device, assume capture
	result := DetectDevice(devicePath)
	return result.Found
}

// GetDeviceSummary returns a human-readable summary of a device.
func GetDeviceSummary(info *DeviceInfo) string {
	if info == nil {
		return "No device"
	}

	name := info.Name
	if name == "" {
		name = "Unknown"
	}

	if info.Driver != "" {
		return fmt.Sprintf("%s (%s) at %s", name, info.Driver, info.Path)
	}
	return fmt.Sprintf("%s at %s", name, info.Path)
}
