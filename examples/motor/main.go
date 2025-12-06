// Example motor demonstrates using the motor component interface.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gorai/gorai/component/motor/fake"
	"github.com/gorai/gorai/pkg/registry"
)

func main() {
	ctx := context.Background()

	// Create a fake motor for demonstration
	m, err := fake.New(ctx, nil, registry.Config{"name": "demo_motor"})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create motor: %v\n", err)
		os.Exit(1)
	}

	motor := m.(*fake.Motor)
	defer motor.Close(ctx)

	fmt.Println("Motor demo starting...")

	// Set power
	fmt.Println("Setting power to 50%...")
	if err := motor.SetPower(ctx, 0.5); err != nil {
		fmt.Fprintf(os.Stderr, "SetPower error: %v\n", err)
	}

	powered, power, _ := motor.IsPowered(ctx)
	fmt.Printf("Motor powered: %v, power level: %.2f\n", powered, power)

	// Move to position
	fmt.Println("Moving to position 10...")
	if err := motor.GoTo(ctx, 100, 10); err != nil {
		fmt.Fprintf(os.Stderr, "GoTo error: %v\n", err)
	}

	pos, _ := motor.Position(ctx)
	fmt.Printf("Current position: %.2f\n", pos)

	// Stop motor
	fmt.Println("Stopping motor...")
	motor.Stop(ctx)

	powered, power, _ = motor.IsPowered(ctx)
	fmt.Printf("Motor powered: %v, power level: %.2f\n", powered, power)

	fmt.Println("Motor demo complete.")
}
