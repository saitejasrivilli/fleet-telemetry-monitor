package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"fleet-telemetry-monitor/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

// Database wraps the SQLite connection
type Database struct {
	conn *sql.DB
}

// New creates a new database connection
func New(dbPath string) (*Database, error) {
	// Enable WAL mode and other optimizations via connection string
	connStr := fmt.Sprintf("%s?_journal_mode=WAL&_synchronous=NORMAL&_cache_size=10000", dbPath)
	
	conn, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings for better performance
	conn.SetMaxOpenConns(1) // SQLite works best with single writer
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(time.Hour)

	db := &Database{conn: conn}
	
	if err := db.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return db, nil
}

// initialize creates tables and indexes
func (db *Database) initialize() error {
	schema := `
	CREATE TABLE IF NOT EXISTS vehicles (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		license_plate TEXT UNIQUE NOT NULL,
		vehicle_type TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS telemetry (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vehicle_id TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		latitude REAL NOT NULL,
		longitude REAL NOT NULL,
		speed REAL NOT NULL,
		heading REAL NOT NULL,
		engine_rpm INTEGER NOT NULL,
		fuel_level REAL NOT NULL,
		odometer_km REAL NOT NULL,
		engine_temp REAL NOT NULL,
		battery_volt REAL NOT NULL,
		diagnostic_code TEXT,
		FOREIGN KEY (vehicle_id) REFERENCES vehicles(id)
	);

	-- Indexes for fast queries (sub-100ms target)
	CREATE INDEX IF NOT EXISTS idx_telemetry_vehicle_id ON telemetry(vehicle_id);
	CREATE INDEX IF NOT EXISTS idx_telemetry_timestamp ON telemetry(timestamp);
	CREATE INDEX IF NOT EXISTS idx_telemetry_vehicle_timestamp ON telemetry(vehicle_id, timestamp);
	CREATE INDEX IF NOT EXISTS idx_telemetry_speed ON telemetry(speed);
	CREATE INDEX IF NOT EXISTS idx_telemetry_diagnostic ON telemetry(diagnostic_code) WHERE diagnostic_code IS NOT NULL;
	`

	_, err := db.conn.Exec(schema)
	return err
}

// Close closes the database connection
func (db *Database) Close() error {
	return db.conn.Close()
}

// InsertVehicle adds a new vehicle
func (db *Database) InsertVehicle(v *models.Vehicle) error {
	query := `INSERT INTO vehicles (id, name, license_plate, vehicle_type) VALUES (?, ?, ?, ?)`
	_, err := db.conn.Exec(query, v.ID, v.Name, v.LicensePlate, v.VehicleType)
	return err
}

// GetVehicle retrieves a vehicle by ID
func (db *Database) GetVehicle(id string) (*models.Vehicle, error) {
	query := `SELECT id, name, license_plate, vehicle_type, created_at FROM vehicles WHERE id = ?`
	
	var v models.Vehicle
	err := db.conn.QueryRow(query, id).Scan(&v.ID, &v.Name, &v.LicensePlate, &v.VehicleType, &v.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ListVehicles returns all vehicles
func (db *Database) ListVehicles() ([]models.Vehicle, error) {
	query := `SELECT id, name, license_plate, vehicle_type, created_at FROM vehicles ORDER BY name`
	
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vehicles []models.Vehicle
	for rows.Next() {
		var v models.Vehicle
		if err := rows.Scan(&v.ID, &v.Name, &v.LicensePlate, &v.VehicleType, &v.CreatedAt); err != nil {
			return nil, err
		}
		vehicles = append(vehicles, v)
	}
	return vehicles, rows.Err()
}

// InsertTelemetry adds a single telemetry record
func (db *Database) InsertTelemetry(t *models.TelemetryData) error {
	query := `
		INSERT INTO telemetry 
		(vehicle_id, timestamp, latitude, longitude, speed, heading, engine_rpm, 
		 fuel_level, odometer_km, engine_temp, battery_volt, diagnostic_code)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := db.conn.Exec(query,
		t.VehicleID, t.Timestamp, t.Latitude, t.Longitude, t.Speed, t.Heading,
		t.EngineRPM, t.FuelLevel, t.OdometerKM, t.EngineTemp, t.BatteryVolt, t.DiagnosticCode,
	)
	if err != nil {
		return err
	}
	
	id, _ := result.LastInsertId()
	t.ID = id
	return nil
}

// InsertTelemetryBatch efficiently inserts multiple telemetry records
func (db *Database) InsertTelemetryBatch(records []models.TelemetryData) (int64, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO telemetry 
		(vehicle_id, timestamp, latitude, longitude, speed, heading, engine_rpm, 
		 fuel_level, odometer_km, engine_temp, battery_volt, diagnostic_code)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	var count int64
	for _, t := range records {
		_, err := stmt.Exec(
			t.VehicleID, t.Timestamp, t.Latitude, t.Longitude, t.Speed, t.Heading,
			t.EngineRPM, t.FuelLevel, t.OdometerKM, t.EngineTemp, t.BatteryVolt, t.DiagnosticCode,
		)
		if err != nil {
			return count, err
		}
		count++
	}

	return count, tx.Commit()
}

// QueryTelemetry retrieves telemetry data based on query parameters
func (db *Database) QueryTelemetry(q models.TelemetryQuery) ([]models.TelemetryData, error) {
	var conditions []string
	var args []interface{}

	baseQuery := `
		SELECT id, vehicle_id, timestamp, latitude, longitude, speed, heading,
		       engine_rpm, fuel_level, odometer_km, engine_temp, battery_volt, diagnostic_code
		FROM telemetry
	`

	if q.VehicleID != "" {
		conditions = append(conditions, "vehicle_id = ?")
		args = append(args, q.VehicleID)
	}
	if !q.StartTime.IsZero() {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, q.StartTime)
	}
	if !q.EndTime.IsZero() {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, q.EndTime)
	}
	if q.MinSpeed > 0 {
		conditions = append(conditions, "speed >= ?")
		args = append(args, q.MinSpeed)
	}
	if q.MaxSpeed > 0 {
		conditions = append(conditions, "speed <= ?")
		args = append(args, q.MaxSpeed)
	}

	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	baseQuery += " ORDER BY timestamp DESC"

	if q.Limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT %d", q.Limit)
		if q.Offset > 0 {
			baseQuery += fmt.Sprintf(" OFFSET %d", q.Offset)
		}
	}

	rows, err := db.conn.Query(baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.TelemetryData
	for rows.Next() {
		var t models.TelemetryData
		var diagCode sql.NullString
		
		err := rows.Scan(
			&t.ID, &t.VehicleID, &t.Timestamp, &t.Latitude, &t.Longitude,
			&t.Speed, &t.Heading, &t.EngineRPM, &t.FuelLevel, &t.OdometerKM,
			&t.EngineTemp, &t.BatteryVolt, &diagCode,
		)
		if err != nil {
			return nil, err
		}
		if diagCode.Valid {
			t.DiagnosticCode = diagCode.String
		}
		results = append(results, t)
	}

	return results, rows.Err()
}

// GetLatestTelemetry returns the most recent telemetry for a vehicle
func (db *Database) GetLatestTelemetry(vehicleID string) (*models.TelemetryData, error) {
	query := `
		SELECT id, vehicle_id, timestamp, latitude, longitude, speed, heading,
		       engine_rpm, fuel_level, odometer_km, engine_temp, battery_volt, diagnostic_code
		FROM telemetry
		WHERE vehicle_id = ?
		ORDER BY timestamp DESC
		LIMIT 1
	`

	var t models.TelemetryData
	var diagCode sql.NullString
	
	err := db.conn.QueryRow(query, vehicleID).Scan(
		&t.ID, &t.VehicleID, &t.Timestamp, &t.Latitude, &t.Longitude,
		&t.Speed, &t.Heading, &t.EngineRPM, &t.FuelLevel, &t.OdometerKM,
		&t.EngineTemp, &t.BatteryVolt, &diagCode,
	)
	if err != nil {
		return nil, err
	}
	if diagCode.Valid {
		t.DiagnosticCode = diagCode.String
	}
	return &t, nil
}

// GetTelemetrySummary returns aggregated statistics for a vehicle
func (db *Database) GetTelemetrySummary(vehicleID string) (*models.TelemetrySummary, error) {
	query := `
		SELECT 
			vehicle_id,
			COUNT(*) as total_records,
			AVG(speed) as avg_speed,
			MAX(speed) as max_speed,
			MAX(odometer_km) - MIN(odometer_km) as total_distance,
			AVG(fuel_level) as avg_fuel,
			AVG(engine_temp) as avg_temp
		FROM telemetry
		WHERE vehicle_id = ?
		GROUP BY vehicle_id
	`

	var s models.TelemetrySummary
	err := db.conn.QueryRow(query, vehicleID).Scan(
		&s.VehicleID, &s.TotalRecords, &s.AvgSpeed, &s.MaxSpeed,
		&s.TotalDistanceKM, &s.AvgFuelLevel, &s.AvgEngineTemp,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// GetDiagnosticAlerts returns all records with diagnostic codes
func (db *Database) GetDiagnosticAlerts(vehicleID string, limit int) ([]models.TelemetryData, error) {
	query := `
		SELECT id, vehicle_id, timestamp, latitude, longitude, speed, heading,
		       engine_rpm, fuel_level, odometer_km, engine_temp, battery_volt, diagnostic_code
		FROM telemetry
		WHERE diagnostic_code IS NOT NULL AND diagnostic_code != ''
	`
	
	var args []interface{}
	if vehicleID != "" {
		query += " AND vehicle_id = ?"
		args = append(args, vehicleID)
	}
	
	query += " ORDER BY timestamp DESC"
	
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.TelemetryData
	for rows.Next() {
		var t models.TelemetryData
		err := rows.Scan(
			&t.ID, &t.VehicleID, &t.Timestamp, &t.Latitude, &t.Longitude,
			&t.Speed, &t.Heading, &t.EngineRPM, &t.FuelLevel, &t.OdometerKM,
			&t.EngineTemp, &t.BatteryVolt, &t.DiagnosticCode,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, t)
	}

	return results, rows.Err()
}

// GetRecordCount returns total telemetry records
func (db *Database) GetRecordCount() (int64, error) {
	var count int64
	err := db.conn.QueryRow("SELECT COUNT(*) FROM telemetry").Scan(&count)
	return count, err
}

// GetStats returns database statistics
func (db *Database) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	var totalRecords int64
	db.conn.QueryRow("SELECT COUNT(*) FROM telemetry").Scan(&totalRecords)
	stats["total_telemetry_records"] = totalRecords
	
	var totalVehicles int64
	db.conn.QueryRow("SELECT COUNT(*) FROM vehicles").Scan(&totalVehicles)
	stats["total_vehicles"] = totalVehicles
	
	var alertCount int64
	db.conn.QueryRow("SELECT COUNT(*) FROM telemetry WHERE diagnostic_code IS NOT NULL AND diagnostic_code != ''").Scan(&alertCount)
	stats["diagnostic_alerts"] = alertCount
	
	return stats, nil
}
