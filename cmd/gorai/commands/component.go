package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/gorai/gorai/pkg/components/metadata"
	"github.com/gorai/gorai/pkg/discovery"
	"github.com/gorai/gorai/pkg/validation"
	"github.com/spf13/cobra"
)

// NewComponentCmd creates the component management command
func NewComponentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "component",
		Short: "Manage third-party components",
		Long:  "Search, install, and manage third-party Gorai components",
	}

	cmd.AddCommand(
		newComponentSearchCmd(),
		newComponentInfoCmd(),
		newComponentAddCmd(),
		newComponentListCmd(),
		newComponentRemoveCmd(),
		newComponentUpdateCmd(),
		newComponentValidateCmd(),
	)

	return cmd
}

// component search
func newComponentSearchCmd() *cobra.Command {
	var (
		typeFilter     string
		platform       string
		license        string
		limit          int
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search for components",
		Long:  "Search for third-party components by keyword, type, or capability",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")

			registry := discovery.NewRegistryClient()
			filters := discovery.SearchFilters{
				Type:     typeFilter,
				Platform: platform,
				License:  license,
				Limit:    limit,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			results, err := registry.SearchComponents(ctx, query, filters)
			if err != nil {
				return fmt.Errorf("search failed: %w", err)
			}

			if len(results) == 0 {
				fmt.Println("No components found.")
				return nil
			}

			fmt.Printf("\nFound %d components:\n\n", len(results))

			for _, result := range results {
				fmt.Printf("  %s %s\n", result.Repository, result.Version)
				if result.Description != "" {
					fmt.Printf("    %s\n", result.Description)
				}

				// Show what it provides
				if len(result.Provides) > 0 {
					provides := make([]string, len(result.Provides))
					for i, p := range result.Provides {
						provides[i] = fmt.Sprintf("%s/%s", p.Type, p.Model)
					}
					fmt.Printf("    Provides: %s\n", strings.Join(provides, ", "))
				}

				// Show metadata
				meta := []string{}
				if result.License != "" {
					meta = append(meta, fmt.Sprintf("License: %s", result.License))
				}
				if result.Maturity != "" {
					meta = append(meta, fmt.Sprintf("★ %s", strings.Title(result.Maturity)))
				}
				if !result.LastUpdated.IsZero() {
					meta = append(meta, fmt.Sprintf("Updated: %s", result.LastUpdated.Format("2006-01-02")))
				}
				if len(meta) > 0 {
					fmt.Printf("    %s\n", strings.Join(meta, " | "))
				}

				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&typeFilter, "type", "", "Filter by component type")
	cmd.Flags().StringVar(&platform, "platform", "", "Filter by platform (e.g., linux/arm64)")
	cmd.Flags().StringVar(&license, "license", "", "Filter by license")
	cmd.Flags().IntVar(&limit, "limit", 20, "Limit number of results")

	return cmd
}

// component info
func newComponentInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info <repository> [version]",
		Short: "Show detailed component information",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository := args[0]
			version := "main"
			if len(args) > 1 {
				version = args[1]
			}

			registry := discovery.NewRegistryClient()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			md, err := registry.GetComponentMetadata(ctx, repository, version)
			if err != nil {
				return fmt.Errorf("failed to fetch metadata: %w", err)
			}

			// Print formatted info
			printComponentInfo(md)

			return nil
		},
	}

	return cmd
}

// component add
func newComponentAddCmd() *cobra.Command {
	var (
		autoImport bool
		models     string
		dryRun     bool
	)

	cmd := &cobra.Command{
		Use:   "add <repository>[@version]",
		Short: "Add a component to the project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse repository@version
			parts := strings.SplitN(args[0], "@", 2)
			repository := parts[0]
			version := ""
			if len(parts) > 1 {
				version = parts[1]
			}

			// Fetch metadata to validate
			registry := discovery.NewRegistryClient()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			md, err := registry.GetComponentMetadata(ctx, repository, version)
			if err != nil {
				return fmt.Errorf("failed to fetch component metadata: %w", err)
			}

			if dryRun {
				fmt.Println("Would add:")
				fmt.Printf("  • %s %s to go.mod\n", repository, md.Component.Version)
				if autoImport {
					fmt.Printf("  • Import statement to main.go\n")
				}
				fmt.Println("\nNo changes made (dry run)")
				return nil
			}

			// Run go get
			versionArg := repository
			if version != "" {
				versionArg = fmt.Sprintf("%s@%s", repository, version)
			}

			fmt.Printf("Adding %s...\n", versionArg)

			goGet := exec.Command("go", "get", versionArg)
			goGet.Stdout = os.Stdout
			goGet.Stderr = os.Stderr

			if err := goGet.Run(); err != nil {
				return fmt.Errorf("go get failed: %w", err)
			}

			fmt.Printf("✓ Added %s %s to go.mod\n", repository, md.Component.Version)
			fmt.Println("✓ Downloaded dependencies")

			// Show next steps
			fmt.Println("\nNext steps:")
			fmt.Println("  1. Add import to your robot binary (e.g., cmd/robot/main.go):")

			for _, p := range md.Component.Provides {
				fmt.Printf("     import _ \"%s/%s\"\n", repository, p.Model)
			}

			fmt.Println("\n  2. Configure in robot.json (example):")
			if len(md.Component.Provides) > 0 {
				p := md.Component.Provides[0]
				fmt.Printf("     {\n")
				fmt.Printf("       \"components\": [\n")
				fmt.Printf("         {\n")
				fmt.Printf("           \"name\": \"%s\",\n", p.Model)
				fmt.Printf("           \"type\": \"%s\",\n", p.Type)
				fmt.Printf("           \"model\": \"%s\",\n", p.Model)
				fmt.Printf("           \"attributes\": {\n")

				// Show example configuration if available
				count := 0
				for name, attr := range md.Component.Configuration {
					if count > 0 {
						fmt.Printf(",\n")
					}
					if attr.Example != nil {
						fmt.Printf("             \"%s\": %v", name, attr.Example)
					} else if attr.Default != nil {
						fmt.Printf("             \"%s\": %v", name, attr.Default)
					}
					count++
					if count >= 3 {
						break
					}
				}
				fmt.Printf("\n")

				fmt.Printf("           }\n")
				fmt.Printf("         }\n")
				fmt.Printf("       ]\n")
				fmt.Printf("     }\n")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&autoImport, "import", false, "Automatically add import statement")
	cmd.Flags().StringVar(&models, "models", "", "Comma-separated list of models to import")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without making changes")

	return cmd
}

// component list
func newComponentListCmd() *cobra.Command {
	var (
		typeFilter string
		format     string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed components",
		RunE: func(cmd *cobra.Command, args []string) error {
			// This would query the component registry at runtime
			// For now, show message that this requires a running robot
			fmt.Println("Installed components:")
			fmt.Println()
			fmt.Println("Note: To see components registered in your binary, the robot must be built.")
			fmt.Println("This feature requires runtime registry introspection.")
			fmt.Println()
			fmt.Println("To see dependencies in your project:")
			fmt.Println("  go list -m all | grep gorai-component")

			return nil
		},
	}

	cmd.Flags().StringVar(&typeFilter, "type", "", "Filter by component type")
	cmd.Flags().StringVar(&format, "format", "table", "Output format (table, json, yaml)")

	return cmd
}

// component remove
func newComponentRemoveCmd() *cobra.Command {
	var cleanImports bool

	cmd := &cobra.Command{
		Use:   "remove <repository>",
		Short: "Remove a component from the project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository := args[0]

			fmt.Printf("Removing %s...\n", repository)

			// Run go mod edit -droprequire
			goModEdit := exec.Command("go", "mod", "edit", "-droprequire", repository)
			goModEdit.Stdout = os.Stdout
			goModEdit.Stderr = os.Stderr

			if err := goModEdit.Run(); err != nil {
				return fmt.Errorf("failed to remove from go.mod: %w", err)
			}

			// Run go mod tidy
			goModTidy := exec.Command("go", "mod", "tidy")
			goModTidy.Stdout = os.Stdout
			goModTidy.Stderr = os.Stderr

			if err := goModTidy.Run(); err != nil {
				return fmt.Errorf("go mod tidy failed: %w", err)
			}

			fmt.Printf("✓ Removed %s\n", repository)

			if cleanImports {
				fmt.Println("✓ Note: Remember to remove import statements from your code")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&cleanImports, "clean-imports", false, "Remove import statements from main.go")

	return cmd
}

// component update
func newComponentUpdateCmd() *cobra.Command {
	var (
		all   bool
		check bool
		major bool
	)

	cmd := &cobra.Command{
		Use:   "update [repository]",
		Short: "Update components to latest versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			if check {
				fmt.Println("Checking for updates...")

				goListCmd := exec.Command("go", "list", "-m", "-u", "all")
				output, err := goListCmd.Output()
				if err != nil {
					return fmt.Errorf("failed to check updates: %w", err)
				}

				fmt.Println(string(output))
				return nil
			}

			if all {
				fmt.Println("Updating all components...")

				goGetCmd := exec.Command("go", "get", "-u", "./...")
				goGetCmd.Stdout = os.Stdout
				goGetCmd.Stderr = os.Stderr

				if err := goGetCmd.Run(); err != nil {
					return fmt.Errorf("update failed: %w", err)
				}

				fmt.Println("✓ Updated all components")
			} else if len(args) > 0 {
				repository := args[0]
				fmt.Printf("Updating %s...\n", repository)

				goGetCmd := exec.Command("go", "get", "-u", repository)
				goGetCmd.Stdout = os.Stdout
				goGetCmd.Stderr = os.Stderr

				if err := goGetCmd.Run(); err != nil {
					return fmt.Errorf("update failed: %w", err)
				}

				fmt.Printf("✓ Updated %s\n", repository)
			} else {
				return fmt.Errorf("specify a repository or use --all")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Update all components")
	cmd.Flags().BoolVar(&check, "check", false, "Only check for updates")
	cmd.Flags().BoolVar(&major, "major", false, "Allow major version updates")

	return cmd
}

// component validate
func newComponentValidateCmd() *cobra.Command {
	var (
		strict bool
		fix    bool
	)

	cmd := &cobra.Command{
		Use:   "validate [path]",
		Short: "Validate a component repository",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}

			absPath, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("invalid path: %w", err)
			}

			fmt.Printf("Validating component at %s...\n\n", absPath)

			validator := validation.NewComponentValidator(absPath, strict)
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
				fmt.Println("\n✓ Component is valid!")
			} else {
				fmt.Println("\n✗ Component validation failed")
				return fmt.Errorf("validation failed")
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&strict, "strict", false, "Enable strict validation")
	cmd.Flags().BoolVar(&fix, "fix", false, "Attempt to auto-fix issues")

	return cmd
}

// Helper functions

func printComponentInfo(md *metadata.ComponentMetadata) {
	fmt.Printf("\n┌────────────────────────────────────────────────────────────────┐\n")
	fmt.Printf("│ %-62s │\n", md.Component.Name)
	fmt.Printf("└────────────────────────────────────────────────────────────────┘\n\n")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	fmt.Fprintf(w, "Repository:\t%s\n", md.Component.Repository)
	fmt.Fprintf(w, "Version:\t%s\n", md.Component.Version)
	if md.Component.Author != "" {
		fmt.Fprintf(w, "Author:\t%s\n", md.Component.Author)
	}
	if md.Component.License != "" {
		fmt.Fprintf(w, "License:\t%s\n", md.Component.License)
	}
	if md.Component.Metadata != nil && md.Component.Metadata.Maturity != "" {
		fmt.Fprintf(w, "Maturity:\t%s\n", strings.Title(md.Component.Metadata.Maturity))
	}

	w.Flush()

	if md.Component.Description != "" {
		fmt.Printf("\nDescription:\n  %s\n", md.Component.Description)
	}

	// Provides
	if len(md.Component.Provides) > 0 {
		fmt.Println("\nProvides:")
		for _, p := range md.Component.Provides {
			fmt.Printf("  • %s/%s", p.Type, p.Model)
			if p.Description != "" {
				fmt.Printf(" - %s", p.Description)
			}
			fmt.Println()
		}
	}

	// Compatibility
	fmt.Println("\nCompatibility:")
	fmt.Printf("  Gorai:      %s\n", md.Component.Compatibility.GoraiVersion)
	if md.Component.Compatibility.GoVersion != "" {
		fmt.Printf("  Go:         %s\n", md.Component.Compatibility.GoVersion)
	}
	if len(md.Component.Compatibility.Platforms) > 0 {
		fmt.Printf("  Platforms:  %s\n", strings.Join(md.Component.Compatibility.Platforms, ", "))
	}

	// Hardware requirements
	if md.Component.HardwareRequirements != nil {
		fmt.Println("\nHardware Requirements:")
		hr := md.Component.HardwareRequirements
		fmt.Printf("  %s GPIO\n", boolToIcon(hr.GPIO))
		fmt.Printf("  %s I2C\n", boolToIcon(hr.I2C))
		fmt.Printf("  %s SPI\n", boolToIcon(hr.SPI))
		fmt.Printf("  %s UART\n", boolToIcon(hr.UART))
	}

	// Configuration
	if len(md.Component.Configuration) > 0 {
		fmt.Println("\nConfiguration Attributes:")
		for name, attr := range md.Component.Configuration {
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

	// Links
	fmt.Println()
	if md.Component.Homepage != "" {
		fmt.Printf("Homepage:      %s\n", md.Component.Homepage)
	}
	if md.Component.Documentation != "" {
		fmt.Printf("Documentation: %s\n", md.Component.Documentation)
	}
}

func boolToIcon(b bool) string {
	if b {
		return "✓"
	}
	return "✗"
}
