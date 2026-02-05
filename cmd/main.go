package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"fleet-telemetry-monitor/internal/api"
	"fleet-telemetry-monitor/internal/db"
	"fleet-telemetry-monitor/internal/models"
	"fleet-telemetry-monitor/internal/parser"

	"github.com/spf13/cobra"
)

var (
	dbPath string
	database *db.Database
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "fleet-monitor",
		Short: "Fleet Telemetry Monitor - Vehicle telemetry ingestion and analysis",
		Long: `A CLI tool for ingesting, processing, and analyzing vehicle telemetry data.
Supports GPS, speed, diagnostics, and other sensor data with SQLite storage
and REST API access.`,
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "fleet_telemetry.db", "Path to SQLite database")

	// Add commands
	rootCmd.AddCommand(serverCmd())
	rootCmd.AddCommand(ingestCmd())
	rootCmd.AddCommand(queryCmd())
	rootCmd.AddCommand(statsCmd())
	rootCmd.AddCommand(generateCmd())
	rootCmd.AddCommand(vehicleCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// initDB initializes database connection
func initDB() error {
	var err error
	database, err = db.New(dbPath)
	return err
}

// serverCmd starts the REST API server
func serverCmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the REST API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initDB(); err != nil {
				return fmt.Errorf("database error: %w", err)
			}
			defer database.Close()

			server := api.NewServer(database)
			addr := fmt.Sprintf(":%d", port)
			
			fmt.Printf("🚀 Fleet Telemetry Monitor API Server\n")
			fmt.Printf("   Listening on http://localhost%s\n", addr)
			fmt.Printf("   Database: %s\n\n", dbPath)
			// Serve web dashboard at root
			server.Router().PathPrefix("/").Handler(http.FileServer(http.Dir("./web/")))
			fmt.Println("Available endpoints:")
			fmt.Println("  GET  /health")
			fmt.Println("  GET  /api/v1/vehicles")
			fmt.Println("  POST /api/v1/vehicles")
			fmt.Println("  GET  /api/v1/vehicles/{id}")
			fmt.Println("  GET  /api/v1/telemetry")
			fmt.Println("  POST /api/v1/telemetry")
			fmt.Println("  POST /api/v1/telemetry/batch")
			fmt.Println("  GET  /api/v1/telemetry/latest/{vehicle_id}")
			fmt.Println("  GET  /api/v1/telemetry/summary/{vehicle_id}")
			fmt.Println("  GET  /api/v1/diagnostics")
			fmt.Println("  GET  /api/v1/stats")
			fmt.Println()

			return http.ListenAndServe(addr, server.Router())
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Server port")
	return cmd
}

// ingestCmd ingests telemetry data from files
func ingestCmd() *cobra.Command {
	var format string
	var validate bool

	cmd := &cobra.Command{
		Use:   "ingest [file...]",
		Short: "Ingest telemetry data from files",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initDB(); err != nil {
				return fmt.Errorf("database error: %w", err)
			}
			defer database.Close()

			p := parser.NewParser(format)
			totalRecords := 0
			totalErrors := 0

			for _, file := range args {
				fmt.Printf("Processing %s...\n", file)
				start := time.Now()

				records, err := p.ParseFile(file)
				if err != nil {
					fmt.Printf("  Error: %v\n", err)
					totalErrors++
					continue
				}

				// Validate if requested
				if validate {
					var valid []models.TelemetryData
					for _, r := range records {
						if errs := parser.ValidateTelemetry(&r); len(errs) == 0 {
							valid = append(valid, r)
						} else {
							totalErrors++
						}
					}
					records = valid
				}

				// Insert into database
				count, err := database.InsertTelemetryBatch(records)
				if err != nil {
					fmt.Printf("  Database error: %v\n", err)
					continue
				}

				elapsed := time.Since(start)
				fmt.Printf("  ✓ Inserted %d records in %v (%.0f records/sec)\n",
					count, elapsed, float64(count)/elapsed.Seconds())
				totalRecords += int(count)
			}

			fmt.Printf("\nTotal: %d records ingested", totalRecords)
			if totalErrors > 0 {
				fmt.Printf(", %d errors", totalErrors)
			}
			fmt.Println()

			return nil
		},
	}

	cmd.Flags().StringVarP(&format, "format", "f", "csv", "File format (csv, json, log)")
	cmd.Flags().BoolVarP(&validate, "validate", "v", true, "Validate records before inserting")
	return cmd
}

// queryCmd queries telemetry data
func queryCmd() *cobra.Command {
	var vehicleID string
	var startTime string
	var endTime string
	var limit int
	var outputFormat string

	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query telemetry data",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initDB(); err != nil {
				return fmt.Errorf("database error: %w", err)
			}
			defer database.Close()

			q := models.TelemetryQuery{
				VehicleID: vehicleID,
				Limit:     limit,
			}

			if startTime != "" {
				t, err := time.Parse(time.RFC3339, startTime)
				if err != nil {
					return fmt.Errorf("invalid start_time format (use RFC3339): %w", err)
				}
				q.StartTime = t
			}

			if endTime != "" {
				t, err := time.Parse(time.RFC3339, endTime)
				if err != nil {
					return fmt.Errorf("invalid end_time format (use RFC3339): %w", err)
				}
				q.EndTime = t
			}

			start := time.Now()
			results, err := database.QueryTelemetry(q)
			if err != nil {
				return fmt.Errorf("query error: %w", err)
			}
			elapsed := time.Since(start)

			switch outputFormat {
			case "json":
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				enc.Encode(results)
			default:
				fmt.Printf("Found %d records (query time: %v)\n\n", len(results), elapsed)
				for _, r := range results {
					fmt.Printf("[%s] Vehicle: %s | Pos: %.6f,%.6f | Speed: %.1f km/h | RPM: %d | Fuel: %.1f%%\n",
						r.Timestamp.Format("2006-01-02 15:04:05"),
						r.VehicleID, r.Latitude, r.Longitude,
						r.Speed, r.EngineRPM, r.FuelLevel)
					if r.DiagnosticCode != "" {
						fmt.Printf("     ⚠️  Diagnostic: %s\n", r.DiagnosticCode)
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&vehicleID, "vehicle", "V", "", "Filter by vehicle ID")
	cmd.Flags().StringVarP(&startTime, "start", "s", "", "Start time (RFC3339)")
	cmd.Flags().StringVarP(&endTime, "end", "e", "", "End time (RFC3339)")
	cmd.Flags().IntVarP(&limit, "limit", "l", 100, "Maximum records to return")
	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json)")
	return cmd
}

// statsCmd shows database statistics
func statsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show database statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initDB(); err != nil {
				return fmt.Errorf("database error: %w", err)
			}
			defer database.Close()

			stats, err := database.GetStats()
			if err != nil {
				return fmt.Errorf("error getting stats: %w", err)
			}

			fmt.Println("📊 Fleet Telemetry Monitor Statistics")
			fmt.Println("=====================================")
			fmt.Printf("  Total Vehicles:     %v\n", stats["total_vehicles"])
			fmt.Printf("  Telemetry Records:  %v\n", stats["total_telemetry_records"])
			fmt.Printf("  Diagnostic Alerts:  %v\n", stats["diagnostic_alerts"])
			fmt.Printf("  Database:           %s\n", dbPath)

			return nil
		},
	}
}

// generateCmd generates sample telemetry data
func generateCmd() *cobra.Command {
	var count int
	var vehicleCount int
	var output string

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate sample telemetry data",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initDB(); err != nil {
				return fmt.Errorf("database error: %w", err)
			}
			defer database.Close()

			rand.Seed(time.Now().UnixNano())

			// Create sample vehicles
			vehicles := []models.Vehicle{}
			vehicleTypes := []string{"Truck", "Van", "Sedan", "SUV"}
			
			for i := 1; i <= vehicleCount; i++ {
				v := models.Vehicle{
					ID:           fmt.Sprintf("VEH-%03d", i),
					Name:         fmt.Sprintf("Vehicle %d", i),
					LicensePlate: fmt.Sprintf("FL-%04d", rand.Intn(10000)),
					VehicleType:  vehicleTypes[rand.Intn(len(vehicleTypes))],
				}
				vehicles = append(vehicles, v)
				database.InsertVehicle(&v)
			}

			fmt.Printf("Created %d vehicles\n", vehicleCount)

			// Generate telemetry data
			diagnosticCodes := []string{"", "", "", "", "", "", "", "P0420", "P0171", "P0300", "P0442"}
			var records []models.TelemetryData
			baseTime := time.Now().Add(-24 * time.Hour)

			for i := 0; i < count; i++ {
				v := vehicles[rand.Intn(len(vehicles))]
				t := models.TelemetryData{
					VehicleID:      v.ID,
					Timestamp:      baseTime.Add(time.Duration(i) * time.Second),
					Latitude:       28.5383 + (rand.Float64()-0.5)*0.1,  // Orlando area
					Longitude:      -81.3792 + (rand.Float64()-0.5)*0.1,
					Speed:          rand.Float64() * 120,
					Heading:        rand.Float64() * 360,
					EngineRPM:      1000 + rand.Intn(6000),
					FuelLevel:      20 + rand.Float64()*80,
					OdometerKM:     float64(50000 + rand.Intn(100000)),
					EngineTemp:     80 + rand.Float64()*30,
					BatteryVolt:    12.0 + rand.Float64()*2,
					DiagnosticCode: diagnosticCodes[rand.Intn(len(diagnosticCodes))],
				}
				records = append(records, t)
			}

			// Insert in batches of 1000
			start := time.Now()
			batchSize := 1000
			inserted := 0

			for i := 0; i < len(records); i += batchSize {
				end := i + batchSize
				if end > len(records) {
					end = len(records)
				}
				batch := records[i:end]
				count, _ := database.InsertTelemetryBatch(batch)
				inserted += int(count)
				fmt.Printf("\rInserted %d/%d records...", inserted, len(records))
			}

			elapsed := time.Since(start)
			fmt.Printf("\n✓ Generated %d telemetry records in %v (%.0f records/sec)\n",
				inserted, elapsed, float64(inserted)/elapsed.Seconds())

			// Export to file if requested
			if output != "" {
				file, err := os.Create(output)
				if err != nil {
					return fmt.Errorf("error creating output file: %w", err)
				}
				defer file.Close()

				enc := json.NewEncoder(file)
				enc.SetIndent("", "  ")
				enc.Encode(records)
				fmt.Printf("Data exported to %s\n", output)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&count, "count", "c", 10000, "Number of records to generate")
	cmd.Flags().IntVarP(&vehicleCount, "vehicles", "n", 10, "Number of vehicles to create")
	cmd.Flags().StringVarP(&output, "output", "o", "", "Export generated data to JSON file")
	return cmd
}

// vehicleCmd manages vehicles
func vehicleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vehicle",
		Short: "Vehicle management commands",
	}

	// List subcommand
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all vehicles",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initDB(); err != nil {
				return fmt.Errorf("database error: %w", err)
			}
			defer database.Close()

			vehicles, err := database.ListVehicles()
			if err != nil {
				return fmt.Errorf("error listing vehicles: %w", err)
			}

			if len(vehicles) == 0 {
				fmt.Println("No vehicles found. Use 'fleet-monitor generate' to create sample data.")
				return nil
			}

			fmt.Printf("%-10s %-20s %-12s %-10s\n", "ID", "Name", "Plate", "Type")
			fmt.Println(string(make([]byte, 55)))
			for _, v := range vehicles {
				fmt.Printf("%-10s %-20s %-12s %-10s\n", v.ID, v.Name, v.LicensePlate, v.VehicleType)
			}

			return nil
		},
	}

	// Summary subcommand
	summaryCmd := &cobra.Command{
		Use:   "summary [vehicle_id]",
		Short: "Show vehicle telemetry summary",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initDB(); err != nil {
				return fmt.Errorf("database error: %w", err)
			}
			defer database.Close()

			start := time.Now()
			summary, err := database.GetTelemetrySummary(args[0])
			if err != nil {
				return fmt.Errorf("error getting summary: %w", err)
			}
			elapsed := time.Since(start)

			fmt.Printf("📈 Telemetry Summary for %s (query: %v)\n", args[0], elapsed)
			fmt.Println("==========================================")
			fmt.Printf("  Total Records:    %d\n", summary.TotalRecords)
			fmt.Printf("  Average Speed:    %.1f km/h\n", summary.AvgSpeed)
			fmt.Printf("  Maximum Speed:    %.1f km/h\n", summary.MaxSpeed)
			fmt.Printf("  Total Distance:   %.1f km\n", summary.TotalDistanceKM)
			fmt.Printf("  Avg Fuel Level:   %.1f%%\n", summary.AvgFuelLevel)
			fmt.Printf("  Avg Engine Temp:  %.1f°C\n", summary.AvgEngineTemp)

			return nil
		},
	}

	cmd.AddCommand(listCmd, summaryCmd)
	return cmd
}
