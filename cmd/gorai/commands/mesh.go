package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"

	"github.com/emergingrobotics/gorai/pkg/mesh"
)

// NewMeshCmd creates the mesh command tree.
func NewMeshCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mesh",
		Short: "Service mesh discovery and management",
		Long: `Interact with the Gorai service mesh for runtime discovery.

The mesh uses NATS KV for service registration and discovery, allowing
independent processes to find each other's services and channels.

Requires a running NATS server with JetStream enabled.`,
	}

	cmd.PersistentFlags().StringP("nats-url", "n", "nats://localhost:4222", "NATS server URL")

	cmd.AddCommand(
		newMeshServicesCmd(),
		newMeshChannelsCmd(),
		newMeshSchemasCmd(),
		newMeshWatchCmd(),
		newMeshSummaryCmd(),
		newMeshRobotsCmd(),
		newMeshInitCmd(),
		newMeshResetCmd(),
	)

	return cmd
}

func getNATSURL(cmd *cobra.Command) string {
	url, _ := cmd.Flags().GetString("nats-url")
	if envURL := os.Getenv("NATS_URL"); envURL != "" && url == "nats://localhost:4222" {
		return envURL
	}
	return url
}

func connectMesh(cmd *cobra.Command) (*mesh.Client, error) {
	url := getNATSURL(cmd)

	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS at %s: %w\nIs NATS running with JetStream enabled?", url, err)
	}

	client, err := mesh.NewClient(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to initialize mesh client: %w", err)
	}

	return client, nil
}

// services subcommand
func newMeshServicesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "services [robot-id]",
		Short: "List registered services",
		Long: `List all services registered in the mesh.

Optionally filter by robot ID. Shows service name, type, subtype, model,
status, and registration time.`,
		Example: `  gorai mesh services
  gorai mesh services robot-alpha
  gorai mesh services --type component
  gorai mesh services --subtype motor`,
		RunE: runMeshServices,
	}

	cmd.Flags().StringP("type", "t", "", "Filter by type (component/service)")
	cmd.Flags().StringP("subtype", "s", "", "Filter by subtype")
	cmd.Flags().StringP("model", "m", "", "Filter by model")
	cmd.Flags().BoolP("json", "j", false, "Output as JSON")

	return cmd
}

func runMeshServices(cmd *cobra.Command, args []string) error {
	client, err := connectMesh(cmd)
	if err != nil {
		return err
	}
	defer client.Conn().Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	q := mesh.Query{}
	if len(args) > 0 {
		q.RobotID = args[0]
	}

	if t, _ := cmd.Flags().GetString("type"); t != "" {
		q.Type = mesh.ServiceType(t)
	}
	if s, _ := cmd.Flags().GetString("subtype"); s != "" {
		q.Subtype = s
	}
	if m, _ := cmd.Flags().GetString("model"); m != "" {
		q.Model = m
	}

	services, err := client.FindServices(ctx, q)
	if err != nil {
		return err
	}

	if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(services)
	}

	if len(services) == 0 {
		fmt.Println("No services registered.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ROBOT\tSERVICE\tTYPE\tSUBTYPE\tMODEL\tSTATUS\tUPTIME")

	for _, svc := range services {
		uptime := time.Since(svc.StartedAt).Truncate(time.Second)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			svc.RobotID,
			svc.Name,
			svc.Type,
			svc.Subtype,
			svc.Model,
			svc.Status,
			uptime,
		)
	}
	w.Flush()

	return nil
}

// channels subcommand
func newMeshChannelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channels [robot-id]",
		Short: "List registered channels",
		Long: `List all NATS channels/subjects registered in the mesh.

Shows the subject pattern, QoS level, direction, publisher, and sample rate.`,
		Example: `  gorai mesh channels
  gorai mesh channels robot-alpha
  gorai mesh channels --qos reliable`,
		RunE: runMeshChannels,
	}

	cmd.Flags().StringP("qos", "q", "", "Filter by QoS (best_effort/reliable/retained/history)")
	cmd.Flags().StringP("direction", "d", "", "Filter by direction (pub/sub/req-rep)")
	cmd.Flags().BoolP("json", "j", false, "Output as JSON")

	return cmd
}

func runMeshChannels(cmd *cobra.Command, args []string) error {
	client, err := connectMesh(cmd)
	if err != nil {
		return err
	}
	defer client.Conn().Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	q := mesh.ChannelQuery{}
	if len(args) > 0 {
		q.RobotID = args[0]
	}

	if qos, _ := cmd.Flags().GetString("qos"); qos != "" {
		q.QoS = mesh.QoS(qos)
	}
	if dir, _ := cmd.Flags().GetString("direction"); dir != "" {
		q.Direction = mesh.Direction(dir)
	}

	channels, err := client.FindChannels(ctx, q)
	if err != nil {
		return err
	}

	if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(channels)
	}

	if len(channels) == 0 {
		fmt.Println("No channels registered.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SUBJECT\tQOS\tDIRECTION\tRATE\tDESCRIPTION")

	for _, ch := range channels {
		desc := ch.Description
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			ch.Subject,
			ch.QoS,
			ch.Direction,
			ch.SampleRate,
			desc,
		)
	}
	w.Flush()

	return nil
}

// schemas subcommand
func newMeshSchemasCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schemas [name/version]",
		Short: "List or show message schemas",
		Long: `List all registered schemas or show a specific schema definition.

Schemas define the structure of messages on channels, using JSON Schema format.`,
		Example: `  gorai mesh schemas
  gorai mesh schemas gorai.sensor.IMUReading/v1`,
		RunE: runMeshSchemas,
	}

	cmd.Flags().BoolP("json", "j", false, "Output as JSON")

	return cmd
}

func runMeshSchemas(cmd *cobra.Command, args []string) error {
	client, err := connectMesh(cmd)
	if err != nil {
		return err
	}
	defer client.Conn().Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	asJSON, _ := cmd.Flags().GetBool("json")

	// Show specific schema
	if len(args) > 0 {
		parts := strings.SplitN(args[0], "/", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid schema key format, expected name/version")
		}

		schema, err := client.GetSchema(ctx, parts[0], parts[1])
		if err != nil {
			return err
		}

		if asJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(schema)
		}

		fmt.Printf("Name:        %s\n", schema.Name)
		fmt.Printf("Version:     %s\n", schema.Version)
		fmt.Printf("Format:      %s\n", schema.Format)
		fmt.Printf("Description: %s\n", schema.Description)
		fmt.Printf("\nDefinition:\n")

		// Pretty print the definition
		var def interface{}
		json.Unmarshal(schema.Definition, &def)
		pretty, _ := json.MarshalIndent(def, "", "  ")
		fmt.Println(string(pretty))

		return nil
	}

	// List all schemas
	schemas, err := client.ListSchemas(ctx)
	if err != nil {
		return err
	}

	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(schemas)
	}

	if len(schemas) == 0 {
		fmt.Println("No schemas registered.")
		fmt.Println("Run 'gorai mesh init' to register predefined schemas.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION\tFORMAT\tDESCRIPTION")

	for _, s := range schemas {
		desc := s.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			s.Name,
			s.Version,
			s.Format,
			desc,
		)
	}
	w.Flush()

	return nil
}

// watch subcommand
func newMeshWatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch [robot-id]",
		Short: "Watch for mesh changes",
		Long: `Watch for services joining/leaving and channel changes in real-time.

Useful for debugging and monitoring the mesh state.`,
		Example: `  gorai mesh watch
  gorai mesh watch robot-alpha`,
		RunE: runMeshWatch,
	}

	return cmd
}

func runMeshWatch(cmd *cobra.Command, args []string) error {
	client, err := connectMesh(cmd)
	if err != nil {
		return err
	}
	defer client.Conn().Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	q := mesh.Query{}
	if len(args) > 0 {
		q.RobotID = args[0]
	}

	fmt.Println("Watching for mesh changes... (Ctrl+C to stop)")
	fmt.Println()

	watcher, err := client.WatchServices(ctx, q)
	if err != nil {
		return err
	}

	for event := range watcher.Events() {
		ts := event.Timestamp.Format("15:04:05")

		switch event.Type {
		case mesh.EventServiceJoined:
			fmt.Printf("[%s] JOIN    %s/%s (%s/%s)\n",
				ts,
				event.Service.RobotID,
				event.Service.Name,
				event.Service.Subtype,
				event.Service.Model,
			)
		case mesh.EventServiceLeft:
			fmt.Printf("[%s] LEAVE   %s/%s\n",
				ts,
				event.Service.RobotID,
				event.Service.Name,
			)
		case mesh.EventServiceUpdated:
			fmt.Printf("[%s] UPDATE  %s/%s (status: %s)\n",
				ts,
				event.Service.RobotID,
				event.Service.Name,
				event.Service.Status,
			)
		}
	}

	return nil
}

// summary subcommand
func newMeshSummaryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "summary",
		Short: "Show mesh summary",
		Long:  `Display a summary of the mesh state including counts of services, channels, and schemas.`,
		RunE:  runMeshSummary,
	}
}

func runMeshSummary(cmd *cobra.Command, args []string) error {
	client, err := connectMesh(cmd)
	if err != nil {
		return err
	}
	defer client.Conn().Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	summary, err := client.GetSummary(ctx)
	if err != nil {
		return err
	}

	fmt.Println("Mesh Summary")
	fmt.Println("============")
	fmt.Printf("Robots:     %d\n", summary.Robots)
	fmt.Printf("Services:   %d (healthy: %d, stale: %d)\n",
		summary.Services, summary.HealthyCount, summary.StaleCount)
	fmt.Printf("Channels:   %d\n", summary.Channels)
	fmt.Printf("Schemas:    %d\n", summary.Schemas)

	if len(summary.ServicesByType) > 0 {
		fmt.Println("\nServices by Type:")
		for t, count := range summary.ServicesByType {
			fmt.Printf("  %s: %d\n", t, count)
		}
	}

	return nil
}

// robots subcommand
func newMeshRobotsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "robots",
		Short: "List robots with registered services",
		RunE:  runMeshRobots,
	}
}

func runMeshRobots(cmd *cobra.Command, args []string) error {
	client, err := connectMesh(cmd)
	if err != nil {
		return err
	}
	defer client.Conn().Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	robots, err := client.GetRobots(ctx)
	if err != nil {
		return err
	}

	if len(robots) == 0 {
		fmt.Println("No robots registered.")
		return nil
	}

	fmt.Println("Registered Robots:")
	for _, r := range robots {
		fmt.Printf("  - %s\n", r)
	}

	return nil
}

// init subcommand
func newMeshInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize mesh with predefined schemas",
		Long: `Register predefined message schemas in the mesh.

This populates the schema registry with common Gorai message types
like IMUReading, GPSReading, MotorCommand, etc.`,
		RunE: runMeshInit,
	}
}

func runMeshInit(cmd *cobra.Command, args []string) error {
	client, err := connectMesh(cmd)
	if err != nil {
		return err
	}
	defer client.Conn().Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("Registering predefined schemas...")

	if err := client.RegisterPredefinedSchemas(ctx); err != nil {
		return err
	}

	fmt.Printf("Registered %d schemas.\n", len(mesh.PredefinedSchemas))

	return nil
}

// reset subcommand
func newMeshResetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset mesh (delete all data)",
		Long: `Delete all mesh KV buckets and their data.

WARNING: This will remove all service registrations, channels, and schemas.
This action cannot be undone.`,
		RunE: runMeshReset,
	}

	cmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")

	return cmd
}

func runMeshReset(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")

	if !force {
		fmt.Print("This will delete all mesh data. Continue? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	client, err := connectMesh(cmd)
	if err != nil {
		return err
	}
	defer client.Conn().Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.Reset(ctx); err != nil {
		return err
	}

	fmt.Println("Mesh data deleted.")
	return nil
}

// cmdMesh handles the 'gorai mesh' command
func cmdMesh() error {
	cmd := NewMeshCmd()
	cmd.SetArgs(os.Args[2:])
	return cmd.Execute()
}
