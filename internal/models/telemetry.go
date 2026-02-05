package models

import "time"

// TelemetryData represents a single telemetry reading from a vehicle
type TelemetryData struct {
	ID            int64     `json:"id"`
	VehicleID     string    `json:"vehicle_id"`
	Timestamp     time.Time `json:"timestamp"`
	Latitude      float64   `json:"latitude"`
	Longitude     float64   `json:"longitude"`
	Speed         float64   `json:"speed"`          // km/h
	Heading       float64   `json:"heading"`        // degrees
	EngineRPM     int       `json:"engine_rpm"`
	FuelLevel     float64   `json:"fuel_level"`     // percentage
	OdometerKM    float64   `json:"odometer_km"`
	EngineTemp    float64   `json:"engine_temp"`    // Celsius
	BatteryVolt   float64   `json:"battery_volt"`
	DiagnosticCode string   `json:"diagnostic_code,omitempty"`
}

// Vehicle represents a fleet vehicle
type Vehicle struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	LicensePlate string   `json:"license_plate"`
	VehicleType string    `json:"vehicle_type"`
	CreatedAt   time.Time `json:"created_at"`
}

// TelemetryQuery represents query parameters for telemetry searches
type TelemetryQuery struct {
	VehicleID   string
	StartTime   time.Time
	EndTime     time.Time
	MinSpeed    float64
	MaxSpeed    float64
	Limit       int
	Offset      int
}

// TelemetrySummary provides aggregated statistics
type TelemetrySummary struct {
	VehicleID       string  `json:"vehicle_id"`
	TotalRecords    int     `json:"total_records"`
	AvgSpeed        float64 `json:"avg_speed"`
	MaxSpeed        float64 `json:"max_speed"`
	TotalDistanceKM float64 `json:"total_distance_km"`
	AvgFuelLevel    float64 `json:"avg_fuel_level"`
	AvgEngineTemp   float64 `json:"avg_engine_temp"`
}

// DiagnosticAlert represents a diagnostic warning
type DiagnosticAlert struct {
	VehicleID      string    `json:"vehicle_id"`
	Timestamp      time.Time `json:"timestamp"`
	DiagnosticCode string    `json:"diagnostic_code"`
	Description    string    `json:"description"`
	Severity       string    `json:"severity"`
}
