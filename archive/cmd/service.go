package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/emergingrobotics/gorai/pkg/discovery"
	"github.com/emergingrobotics/gorai/pkg/services/rdl"
	"github.com/emergingrobotics/gorai/pkg/validation"
	"github.com/spf13/cobra"
)

// NewServiceCmd creates the service management command
func NewServiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage external services",
		Long:  "Search, pull, and manage external container-based services",
	}

	cmd.AddCommand(
		newServiceSearchCmd(),
		newServiceInfoCmd(),
		newServicePullCmd(),
		newServiceValidateCmd(),
	)

	return cmd
}

// service search
func newServiceSearchCmd() *cobra.Command {
	var (
		typeFilter  string
		accelerator string
		platform    string
		limit       int
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search for services",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")

			registry := discovery.NewRegistryClient()
			filters := discovery.ServiceSearchFilters{
				Type:        typeFilter,
				Accelerator: accelerator,
				Platform:    platform,
				Limit:       limit,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			results, err := registry.SearchServices(ctx, query, filters)
			if err != nil {
				return fmt.Errorf("search failed: %w", err)
			}

			if len(results) == 0 {
				fmt.Println("No services found.")
				return nil
			}

			fmt.Printf("\nFound %d services:\n\n", len(results))

			for _, result := range results {
				fmt.Printf("  %s\n", result.Image)
				if result.Description != "" {
					fmt.Printf("    %s\n", result.Description)
				}

				meta := []string{}
				if result.Type != "" {
					meta = append(meta, fmt.Sprintf("Type: %s", result.Type))
				}
				if result.Performance != "" {
					meta = append(meta, result.Performance)
				}
				if result.License != "" {
					meta = append(meta, fmt.Sprintf("License: %s", result.License))
				}
				if result.Maturity != "" {
					meta = append(meta, fmt.Sprintf("★ %s", strings.Title(result.Maturity)))
				}

				if len(meta) > 0 {
					fmt.Printf("    %s\n", strings.Join(meta, " | "))
				}

				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&typeFilter, "type", "", "Filter by service type")
	cmd.Flags().StringVar(&accelerator, "accelerator", "", "Filter by accelerator support")
	cmd.Flags().StringVar(&platform, "platform", "", "Filter by platform")
	cmd.Flags().IntVar(&limit, "limit", 20, "Limit number of results")

	return cmd
}

// service info
func newServiceInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <image-or-rdl-url>",
		Short: "Show detailed service information",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			source := args[0]

			var serviceRDL *rdl.ServiceRDL
			var err error

			// Check if it's a URL
			if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
				serviceRDL, err = rdl.FetchFromURL(source)
			} else {
				// Try to find metadata for container image
				// For now, show error that this needs implementation
				return fmt.Errorf("fetching service metadata from container image registry not yet implemented. Please provide RDL URL")
			}

			if err != nil {
				return fmt.Errorf("failed to fetch service info: %w", err)
			}

			// Print formatted info
			printServiceInfo(serviceRDL)

			return nil
		},
	}

	return cmd
}

// service pull
func newServicePullCmd() *cobra.Command {
	var (
		platform    string
		allVariants bool
	)

	cmd := &cobra.Command{
		Use:   "pull <image>",
		Short: "Pull service container image",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			image := args[0]

			fmt.Printf("Pulling %s...\n", image)

			pullCmd := exec.Command("podman", "pull", image)
			if platform != "" {
				pullCmd.Args = append(pullCmd.Args, "--platform", platform)
			}

			pullCmd.Stdout = os.Stdout
			pullCmd.Stderr = os.Stderr

			if err := pullCmd.Run(); err != nil {
				// Try docker as fallback
				pullCmd = exec.Command("docker", "pull", image)
				if platform != "" {
					pullCmd.Args = append(pullCmd.Args, "--platform", platform)
				}
				pullCmd.Stdout = os.Stdout
				pullCmd.Stderr = os.Stderr

				if err := pullCmd.Run(); err != nil {
					return fmt.Errorf("failed to pull image: %w", err)
				}
			}

			fmt.Printf("✓ Downloaded %s\n", image)

			return nil
		},
	}

	cmd.Flags().StringVar(&platform, "platform", "", "Pull for specific platform")
	cmd.Flags().BoolVar(&allVariants, "all-variants", false, "Pull all image variants")

	return cmd
}

// service validate
func newServiceValidateCmd() *cobra.Command {
	var strict bool

	cmd := &cobra.Command{
		Use:   "validate <rdl-file>",
		Short: "Validate a service RDL file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rdlPath := args[0]

			fmt.Printf("Validating %s...\n\n", rdlPath)

			// Check if it's a local file or directory
			absPath, err := filepath.Abs(rdlPath)
			if err != nil {
				return fmt.Errorf("invalid path: %w", err)
			}

			// If directory, find service.rdl.json in it
			if info, err := os.Stat(absPath); err == nil && info.IsDir() {
				absPath, err = rdl.FindInDirectory(absPath)
				if err != nil {
					return err
				}
			}

			validator := validation.NewServiceValidator(filepath.Dir(absPath), strict)
			result, err := validator.Validate()
			if err != nil {
				return fmt.Errorf("validation error: %w", err)
			}

			// Print checks
			for _, check := range result.Checks {
				if check.Passed {
					fmt.Printf("✓ %s\n", check.Name)
				} else {
					fmt.Printf("✗ %s\n", check.Name)
				}
			}

			// Print errors
			if len(result.Errors) > 0 {
				fmt.Println("\nErrors:")
				for _, e := range result.Errors {
					fmt.Printf("  ✗ %s\n", e)
				}
			}

			// Print warnings
			if len(result.Warnings) > 0 {
				fmt.Println("\nWarnings:")
				for _, w := range result.Warnings {
					fmt.Printf("  ⚠ %s\n", w)
				}
			}

			// Print quality score
			fmt.Printf("\nQuality score: %d/100\n", result.QualityScore)

			if result.Valid {
				fmt.Println("\n✓ Service RDL is valid!")
			} else {
				fmt.Println("\n✗ Service validation failed")
				return fmt.Errorf("validation failed")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&strict, "strict", false, "Enable strict validation")

	return cmd
}

// Helper functions

func printServiceInfo(serviceRDL *rdl.ServiceRDL) {
	s := serviceRDL.Service

	fmt.Printf("\n┌────────────────────────────────────────────────────────────────┐\n")
	fmt.Printf("│ %-62s │\n", s.Name)
	fmt.Printf("└────────────────────────────────────────────────────────────────┘\n\n")

	fmt.Printf("Name:    %s\n", s.Name)
	fmt.Printf("Type:    %s\n", s.Type)
	fmt.Printf("Model:   %s\n", s.Model)
	fmt.Printf("Version: %s\n", s.Version)
	if s.License != "" {
		fmt.Printf("License: %s\n", s.License)
	}

	if s.Description != "" {
		fmt.Printf("\nDescription:\n  %s\n", s.Description)
	}

	// Container images
	if s.Container != nil {
		fmt.Println("\nContainer Images:")
		fmt.Printf("  • default: %s\n", s.Container.DefaultImage)

		if len(s.Container.ImageVariants) > 0 {
			for name, variant := range s.Container.ImageVariants {
				fmt.Printf("  • %s: %s", name, variant.Image)
				if variant.Description != "" {
					fmt.Printf(" (%s)", variant.Description)
				}
				fmt.Println()
			}
		}
	}

	// NATS topics
	if s.NATSTopics != nil {
		fmt.Println("\nNATS Topics:")
		fmt.Println("  Subscribes:")
		for _, topic := range s.NATSTopics.Subscribes {
			fmt.Printf("    • %s", topic.Pattern)
			if topic.Description != "" {
				fmt.Printf(" - %s", topic.Description)
			}
			fmt.Println()
		}
		fmt.Println("  Publishes:")
		for _, topic := range s.NATSTopics.Publishes {
			fmt.Printf("    • %s", topic.Pattern)
			if topic.Description != "" {
				fmt.Printf(" - %s", topic.Description)
			}
			fmt.Println()
		}
	}

	// Configuration
	if len(s.Configuration) > 0 {
		fmt.Println("\nConfiguration:")
		for name, attr := range s.Configuration {
			req := ""
			if attr.Required {
				req = ", required"
			}
			fmt.Printf("  • %s (%s%s)\n", name, attr.Type, req)
			if attr.Description != "" {
				fmt.Printf("      %s\n", attr.Description)
			}
			if attr.Default != nil {
				fmt.Printf("      Default: %v\n", attr.Default)
			}
		}
	}

	// Performance
	if s.Performance != nil {
		fmt.Println("\nPerformance:")
		if s.Performance.Latency != nil {
			if s.Performance.Latency.Typical != "" {
				fmt.Printf("  Latency:    %s typical", s.Performance.Latency.Typical)
				if s.Performance.Latency.Max != "" {
					fmt.Printf(", %s max", s.Performance.Latency.Max)
				}
				fmt.Println()
			}
		}
		if s.Performance.Throughput != "" {
			fmt.Printf("  Throughput: %s\n", s.Performance.Throughput)
		}
		if s.Performance.StartupTime != "" {
			fmt.Printf("  Startup:    %s\n", s.Performance.StartupTime)
		}
	}

	// Resources
	if s.Container != nil && s.Container.ResourceRequirements != nil {
		fmt.Println("\nResources:")
		rr := s.Container.ResourceRequirements
		if rr.Memory != "" {
			limits := ""
			if s.Container.ResourceLimits != nil && s.Container.ResourceLimits.Memory != "" {
				limits = fmt.Sprintf(", %s limit", s.Container.ResourceLimits.Memory)
			}
			fmt.Printf("  Memory: %s required%s\n", rr.Memory, limits)
		}
		if rr.CPU != "" {
			limits := ""
			if s.Container.ResourceLimits != nil && s.Container.ResourceLimits.CPU != "" {
				limits = fmt.Sprintf(", %s limit", s.Container.ResourceLimits.CPU)
			}
			fmt.Printf("  CPU:    %s required%s\n", rr.CPU, limits)
		}
		if rr.GPU != "" {
			fmt.Printf("  GPU:    %s\n", rr.GPU)
		}
	}

	// Links
	fmt.Println()
	if s.Documentation != "" {
		fmt.Printf("Documentation: %s\n", s.Documentation)
	}
	if s.Repository != "" {
		fmt.Printf("Repository:    %s\n", s.Repository)
	}
}
