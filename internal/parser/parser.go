package parser

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"fleet-telemetry-monitor/internal/models"
)

// Parser handles parsing of telemetry data files
type Parser struct {
	format string
}

// NewParser creates a new parser with the specified format
func NewParser(format string) *Parser {
	return &Parser{format: format}
}

// ParseFile parses a telemetry data file
func (p *Parser) ParseFile(filename string) ([]models.TelemetryData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	switch strings.ToLower(p.format) {
	case "csv":
		return p.parseCSV(file)
	case "json":
		return p.parseJSON(file)
	case "log":
		return p.parseLog(file)
	default:
		return nil, fmt.Errorf("unsupported format: %s", p.format)
	}
}

// parseCSV parses CSV formatted telemetry data
func (p *Parser) parseCSV(r io.Reader) ([]models.TelemetryData, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1 // Allow variable fields

	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	// Map header indices
	indices := make(map[string]int)
	for i, h := range header {
		indices[strings.ToLower(strings.TrimSpace(h))] = i
	}

	var results []models.TelemetryData
	lineNum := 1

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return results, fmt.Errorf("error at line %d: %w", lineNum, err)
		}
		lineNum++

		data, err := p.recordToTelemetry(record, indices)
		if err != nil {
			// Log error but continue parsing
			fmt.Printf("Warning: line %d: %v\n", lineNum, err)
			continue
		}
		results = append(results, data)
	}

	return results, nil
}

// recordToTelemetry converts a CSV record to TelemetryData
func (p *Parser) recordToTelemetry(record []string, indices map[string]int) (models.TelemetryData, error) {
	var t models.TelemetryData
	var err error

	getValue := func(key string) string {
		if idx, ok := indices[key]; ok && idx < len(record) {
			return strings.TrimSpace(record[idx])
		}
		return ""
	}

	t.VehicleID = getValue("vehicle_id")
	if t.VehicleID == "" {
		return t, fmt.Errorf("missing vehicle_id")
	}

	// Parse timestamp
	tsStr := getValue("timestamp")
	if tsStr != "" {
		t.Timestamp, err = parseTimestamp(tsStr)
		if err != nil {
			return t, fmt.Errorf("invalid timestamp: %w", err)
		}
	}

	// Parse numeric fields
	t.Latitude, _ = strconv.ParseFloat(getValue("latitude"), 64)
	t.Longitude, _ = strconv.ParseFloat(getValue("longitude"), 64)
	t.Speed, _ = strconv.ParseFloat(getValue("speed"), 64)
	t.Heading, _ = strconv.ParseFloat(getValue("heading"), 64)
	t.EngineRPM, _ = strconv.Atoi(getValue("engine_rpm"))
	t.FuelLevel, _ = strconv.ParseFloat(getValue("fuel_level"), 64)
	t.OdometerKM, _ = strconv.ParseFloat(getValue("odometer_km"), 64)
	t.EngineTemp, _ = strconv.ParseFloat(getValue("engine_temp"), 64)
	t.BatteryVolt, _ = strconv.ParseFloat(getValue("battery_volt"), 64)
	t.DiagnosticCode = getValue("diagnostic_code")

	return t, nil
}

// parseJSON parses JSON formatted telemetry data
func (p *Parser) parseJSON(r io.Reader) ([]models.TelemetryData, error) {
	var results []models.TelemetryData
	decoder := json.NewDecoder(r)

	// Try to decode as array first
	if err := decoder.Decode(&results); err == nil {
		return results, nil
	}

	// Reset and try line-by-line JSON
	return p.parseJSONLines(r)
}

// parseJSONLines parses newline-delimited JSON
func (p *Parser) parseJSONLines(r io.Reader) ([]models.TelemetryData, error) {
	var results []models.TelemetryData
	scanner := bufio.NewScanner(r)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line == "[" || line == "]" {
			continue
		}

		// Remove trailing comma if present
		line = strings.TrimSuffix(line, ",")

		var t models.TelemetryData
		if err := json.Unmarshal([]byte(line), &t); err != nil {
			fmt.Printf("Warning: line %d: %v\n", lineNum, err)
			continue
		}
		results = append(results, t)
	}

	return results, scanner.Err()
}

// parseLog parses custom log format: timestamp|vehicle_id|lat,lon|speed|rpm|fuel|odo|temp|batt|diag
func (p *Parser) parseLog(r io.Reader) ([]models.TelemetryData, error) {
	var results []models.TelemetryData
	scanner := bufio.NewScanner(r)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 10 {
			fmt.Printf("Warning: line %d: insufficient fields\n", lineNum)
			continue
		}

		var t models.TelemetryData
		var err error

		t.Timestamp, err = parseTimestamp(parts[0])
		if err != nil {
			fmt.Printf("Warning: line %d: invalid timestamp\n", lineNum)
			continue
		}

		t.VehicleID = parts[1]

		// Parse lat,lon
		coords := strings.Split(parts[2], ",")
		if len(coords) == 2 {
			t.Latitude, _ = strconv.ParseFloat(coords[0], 64)
			t.Longitude, _ = strconv.ParseFloat(coords[1], 64)
		}

		t.Speed, _ = strconv.ParseFloat(parts[3], 64)
		t.EngineRPM, _ = strconv.Atoi(parts[4])
		t.FuelLevel, _ = strconv.ParseFloat(parts[5], 64)
		t.OdometerKM, _ = strconv.ParseFloat(parts[6], 64)
		t.EngineTemp, _ = strconv.ParseFloat(parts[7], 64)
		t.BatteryVolt, _ = strconv.ParseFloat(parts[8], 64)
		
		if len(parts) > 9 {
			t.DiagnosticCode = parts[9]
		}

		results = append(results, t)
	}

	return results, scanner.Err()
}

// parseTimestamp tries multiple timestamp formats
func parseTimestamp(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05",
		"01/02/2006 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	// Try Unix timestamp
	if ts, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(ts, 0), nil
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", s)
}

// ValidateTelemetry validates telemetry data
func ValidateTelemetry(t *models.TelemetryData) []string {
	var errors []string

	if t.VehicleID == "" {
		errors = append(errors, "vehicle_id is required")
	}
	if t.Latitude < -90 || t.Latitude > 90 {
		errors = append(errors, "latitude must be between -90 and 90")
	}
	if t.Longitude < -180 || t.Longitude > 180 {
		errors = append(errors, "longitude must be between -180 and 180")
	}
	if t.Speed < 0 {
		errors = append(errors, "speed cannot be negative")
	}
	if t.FuelLevel < 0 || t.FuelLevel > 100 {
		errors = append(errors, "fuel_level must be between 0 and 100")
	}
	if t.EngineRPM < 0 {
		errors = append(errors, "engine_rpm cannot be negative")
	}

	return errors
}
