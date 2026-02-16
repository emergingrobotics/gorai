package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// Known component types and their descriptions
var knownComponents = map[string]componentInfo{
	// Sensors
	"imu":               {"Inertial Measurement Unit", []string{"mpu6050", "mpu9250", "lsm6ds3", "fake"}},
	"ahrs":              {"Attitude/Heading Reference System", []string{"bno055", "bno085", "fake"}},
	"gps":               {"GPS/GNSS Receiver", []string{"neo6m", "neo_m8n", "zed_f9p", "fake"}},
	"encoder":           {"Rotary/Linear Encoder", []string{"quadrature", "as5600", "amt10x", "fake"}},
	"range_sensor":      {"Distance Sensor", []string{"hcsr04", "vl53l0x", "vl53l1x", "fake"}},
	"lidar":             {"Laser Scanner", []string{"rplidar_a1", "rplidar_a2", "hokuyo", "fake"}},
	"presence_sensor":   {"Presence Detector", []string{"pir", "ld2410", "ld2450", "fake"}},
	"thermal_array":     {"Thermal Imager", []string{"amg8833", "mlx90640", "fake"}},
	"force_sensor":      {"Force Sensor", []string{"hx711", "fake"}},
	"force_6dof":        {"6-Axis F/T Sensor", []string{"ati_mini45", "fake"}},
	"current_sensor":    {"Current/Power Monitor", []string{"ina219", "ina260", "acs712", "fake"}},
	"reflectance_sensor": {"Line Sensor", []string{"qtr8rc", "tcrt5000", "fake"}},
	"camera":            {"Camera", []string{"v4l2", "picamera", "realsense", "fake"}},
	"temperature":       {"Temperature Sensor", []string{"linux_thermal", "ds18b20", "bme280", "fake"}},

	// Actuators
	"motor":    {"DC/Brushless Motor", []string{"gpio", "can", "serial", "odrive", "fake"}},
	"servo":    {"Position Servo", []string{"pwm", "dynamixel", "lx16a", "feetech", "fake"}},
	"stepper":  {"Stepper Motor", []string{"gpio", "tmc2209", "tmc5160", "fake"}},
	"thruster": {"Underwater Thruster", []string{"pwm", "bluerobotics", "fake"}},
	"valve":    {"Valve", []string{"gpio", "solenoid", "motorized", "fake"}},
	"gripper":  {"Gripper", []string{"servo", "pneumatic", "fake"}},
	"arm":      {"Robot Arm", []string{"custom", "fake"}},
	"base":     {"Mobile Base", []string{"differential", "mecanum", "ackermann", "fake"}},

	// Infrastructure
	"power": {"Power Source", []string{"battery", "adc", "ina219", "fake"}},
	"space": {"Virtual Container", []string{"container", "tank", "fake"}},
	"link":  {"Communication Link", []string{"serial", "radio", "can", "fake"}},
}

// Known service types and their descriptions
var knownServices = map[string]componentInfo{
	"vision":      {"Computer Vision", []string{"yolox", "yolov8", "tflite", "custom", "fake"}},
	"slam":        {"SLAM/Mapping", []string{"cartographer", "gmapping", "fake"}},
	"navigation":  {"Path Planning", []string{"default", "custom", "fake"}},
	"motion":      {"Motion Planning", []string{"default", "custom", "fake"}},
	"behavior":    {"Behavior Trees", []string{"default", "custom", "fake"}},
	"coordinator": {"Multi-Robot Coordination", []string{"default", "custom", "fake"}},
	"mlmodel":     {"ML Inference", []string{"tflite", "onnx", "tpu", "fake"}},
}

type componentInfo struct {
	Description string
	Models      []string
}

// cmdListComponents handles 'gorai components' by directly listing component types.
func cmdListComponents() error {
	args := os.Args[2:]
	return listComponents(args)
}

// cmdList handles the 'gorai list' command.
func cmdList() error {
	args := os.Args[2:]

	if len(args) == 0 {
		return printListUsage()
	}

	subCmd := args[0]
	switch subCmd {
	case "components":
		return listComponents(args[1:])
	case "services":
		return listServices(args[1:])
	case "models":
		return listModels(args[1:])
	case "-h", "--help":
		return printListUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nUse 'gorai list -h' for help.", subCmd)
	}
}

func printListUsage() error {
	fmt.Println(`gorai list - List available components and services

Usage:
  gorai list <subcommand> [flags]

Subcommands:
  components          List all component types
  services            List all service types
  models              List models for a specific type

Examples:
  gorai list components
  gorai list services
  gorai list models --type motor
  gorai list models --type ahrs`)
	return nil
}

func listComponents(args []string) error {
	// Parse flags
	var typeFilter string
	for i := 0; i < len(args); i++ {
		if args[i] == "--type" && i+1 < len(args) {
			typeFilter = args[i+1]
			i++
		} else if strings.HasPrefix(args[i], "--type=") {
			typeFilter = strings.TrimPrefix(args[i], "--type=")
		}
	}

	fmt.Println("Component Types:")
	fmt.Println()

	// Group by category
	sensors := []string{}
	actuators := []string{}
	infrastructure := []string{}

	for name := range knownComponents {
		if typeFilter != "" && name != typeFilter {
			continue
		}

		switch name {
		case "imu", "ahrs", "gps", "encoder", "range_sensor", "lidar", "presence_sensor",
			"thermal_array", "force_sensor", "force_6dof", "current_sensor", "reflectance_sensor",
			"camera", "temperature":
			sensors = append(sensors, name)
		case "motor", "servo", "stepper", "thruster", "valve", "gripper", "arm", "base":
			actuators = append(actuators, name)
		case "power", "space", "link":
			infrastructure = append(infrastructure, name)
		}
	}

	sort.Strings(sensors)
	sort.Strings(actuators)
	sort.Strings(infrastructure)

	if len(sensors) > 0 {
		fmt.Println("  Sensors:")
		for _, name := range sensors {
			info := knownComponents[name]
			models := strings.Join(info.Models, ", ")
			fmt.Printf("    %-18s - %s\n", name, info.Description)
			fmt.Printf("    %-18s   models: %s\n", "", models)
		}
		fmt.Println()
	}

	if len(actuators) > 0 {
		fmt.Println("  Actuators:")
		for _, name := range actuators {
			info := knownComponents[name]
			models := strings.Join(info.Models, ", ")
			fmt.Printf("    %-18s - %s\n", name, info.Description)
			fmt.Printf("    %-18s   models: %s\n", "", models)
		}
		fmt.Println()
	}

	if len(infrastructure) > 0 {
		fmt.Println("  Infrastructure:")
		for _, name := range infrastructure {
			info := knownComponents[name]
			models := strings.Join(info.Models, ", ")
			fmt.Printf("    %-18s - %s\n", name, info.Description)
			fmt.Printf("    %-18s   models: %s\n", "", models)
		}
	}

	return nil
}

func listServices(args []string) error {
	fmt.Println("Service Types:")
	fmt.Println()

	// Sort service names
	names := make([]string, 0, len(knownServices))
	for name := range knownServices {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		info := knownServices[name]
		models := strings.Join(info.Models, ", ")
		fmt.Printf("  %-14s - %s\n", name, info.Description)
		fmt.Printf("  %-14s   models: %s\n", "", models)
	}

	return nil
}

func listModels(args []string) error {
	// Parse flags
	var typeFilter string
	for i := 0; i < len(args); i++ {
		if args[i] == "--type" && i+1 < len(args) {
			typeFilter = args[i+1]
			i++
		} else if strings.HasPrefix(args[i], "--type=") {
			typeFilter = strings.TrimPrefix(args[i], "--type=")
		}
	}

	if typeFilter == "" {
		return fmt.Errorf("--type flag required\n\nUsage: gorai list models --type <component-type>")
	}

	// Check components first
	if info, ok := knownComponents[typeFilter]; ok {
		fmt.Printf("Models for component type %q:\n\n", typeFilter)
		fmt.Printf("  %s\n\n", info.Description)
		fmt.Println("  Available models:")
		for _, model := range info.Models {
			fmt.Printf("    - %s\n", model)
		}
		return nil
	}

	// Check services
	if info, ok := knownServices[typeFilter]; ok {
		fmt.Printf("Models for service type %q:\n\n", typeFilter)
		fmt.Printf("  %s\n\n", info.Description)
		fmt.Println("  Available models:")
		for _, model := range info.Models {
			fmt.Printf("    - %s\n", model)
		}
		return nil
	}

	// Not found
	return fmt.Errorf("unknown type %q\n\nUse 'gorai list components' or 'gorai list services' to see available types.", typeFilter)
}
