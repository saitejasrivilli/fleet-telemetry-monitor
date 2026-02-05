package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"fleet-telemetry-monitor/internal/db"
	"fleet-telemetry-monitor/internal/models"
	"fleet-telemetry-monitor/internal/parser"

	"github.com/gorilla/mux"
)

// Server represents the API server
type Server struct {
	db     *db.Database
	router *mux.Router
}

// NewServer creates a new API server
func NewServer(database *db.Database) *Server {
	s := &Server{
		db:     database,
		router: mux.NewRouter(),
	}
	s.setupRoutes()
	return s
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	
	// Vehicle endpoints
	s.router.HandleFunc("/api/v1/vehicles", s.handleListVehicles).Methods("GET")
	s.router.HandleFunc("/api/v1/vehicles", s.handleCreateVehicle).Methods("POST")
	s.router.HandleFunc("/api/v1/vehicles/{id}", s.handleGetVehicle).Methods("GET")
	
	// Telemetry endpoints
	s.router.HandleFunc("/api/v1/telemetry", s.handleQueryTelemetry).Methods("GET")
	s.router.HandleFunc("/api/v1/telemetry", s.handleCreateTelemetry).Methods("POST")
	s.router.HandleFunc("/api/v1/telemetry/batch", s.handleBatchTelemetry).Methods("POST")
	s.router.HandleFunc("/api/v1/telemetry/latest/{vehicle_id}", s.handleLatestTelemetry).Methods("GET")
	s.router.HandleFunc("/api/v1/telemetry/summary/{vehicle_id}", s.handleTelemetrySummary).Methods("GET")
	
	// Diagnostics endpoints
	s.router.HandleFunc("/api/v1/diagnostics", s.handleGetDiagnostics).Methods("GET")
	s.router.HandleFunc("/api/v1/diagnostics/{vehicle_id}", s.handleGetVehicleDiagnostics).Methods("GET")
	
	// Stats endpoint
	s.router.HandleFunc("/api/v1/stats", s.handleStats).Methods("GET")

	// Add middleware
	s.router.Use(loggingMiddleware)
	s.router.Use(jsonMiddleware)
}

// Router returns the configured router
func (s *Server) Router() *mux.Router {
	return s.router
}

// Middleware
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func jsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// Response helpers
type apiResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    *meta       `json:"meta,omitempty"`
}

type meta struct {
	Total   int    `json:"total,omitempty"`
	Limit   int    `json:"limit,omitempty"`
	Offset  int    `json:"offset,omitempty"`
	QueryMs int64  `json:"query_ms,omitempty"`
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(apiResponse{Success: true, Data: data})
}

func respondError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(apiResponse{Success: false, Error: message})
}

func respondWithMeta(w http.ResponseWriter, data interface{}, m *meta) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(apiResponse{Success: true, Data: data, Meta: m})
}

// Handlers
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"status": "healthy"})
}

func (s *Server) handleListVehicles(w http.ResponseWriter, r *http.Request) {
	vehicles, err := s.db.ListVehicles()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, vehicles)
}

func (s *Server) handleCreateVehicle(w http.ResponseWriter, r *http.Request) {
	var v models.Vehicle
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if v.ID == "" || v.Name == "" || v.LicensePlate == "" {
		respondError(w, http.StatusBadRequest, "id, name, and license_plate are required")
		return
	}

	if err := s.db.InsertVehicle(&v); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, v)
}

func (s *Server) handleGetVehicle(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	vehicle, err := s.db.GetVehicle(id)
	if err != nil {
		respondError(w, http.StatusNotFound, "vehicle not found")
		return
	}

	respondJSON(w, http.StatusOK, vehicle)
}

func (s *Server) handleQueryTelemetry(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	q := models.TelemetryQuery{
		VehicleID: r.URL.Query().Get("vehicle_id"),
		Limit:     100, // default
	}

	if v := r.URL.Query().Get("limit"); v != "" {
		q.Limit, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		q.Offset, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("start_time"); v != "" {
		q.StartTime, _ = time.Parse(time.RFC3339, v)
	}
	if v := r.URL.Query().Get("end_time"); v != "" {
		q.EndTime, _ = time.Parse(time.RFC3339, v)
	}
	if v := r.URL.Query().Get("min_speed"); v != "" {
		q.MinSpeed, _ = strconv.ParseFloat(v, 64)
	}
	if v := r.URL.Query().Get("max_speed"); v != "" {
		q.MaxSpeed, _ = strconv.ParseFloat(v, 64)
	}

	results, err := s.db.QueryTelemetry(q)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	queryMs := time.Since(start).Milliseconds()
	respondWithMeta(w, results, &meta{
		Total:   len(results),
		Limit:   q.Limit,
		Offset:  q.Offset,
		QueryMs: queryMs,
	})
}

func (s *Server) handleCreateTelemetry(w http.ResponseWriter, r *http.Request) {
	var t models.TelemetryData
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if errs := parser.ValidateTelemetry(&t); len(errs) > 0 {
		respondError(w, http.StatusBadRequest, errs[0])
		return
	}

	if t.Timestamp.IsZero() {
		t.Timestamp = time.Now()
	}

	if err := s.db.InsertTelemetry(&t); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, t)
}

func (s *Server) handleBatchTelemetry(w http.ResponseWriter, r *http.Request) {
	var records []models.TelemetryData
	if err := json.NewDecoder(r.Body).Decode(&records); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON array")
		return
	}

	if len(records) == 0 {
		respondError(w, http.StatusBadRequest, "empty array")
		return
	}

	// Set timestamps for records without one
	now := time.Now()
	for i := range records {
		if records[i].Timestamp.IsZero() {
			records[i].Timestamp = now
		}
	}

	count, err := s.db.InsertTelemetryBatch(records)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, map[string]int64{"inserted": count})
}

func (s *Server) handleLatestTelemetry(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	vehicleID := vars["vehicle_id"]

	telemetry, err := s.db.GetLatestTelemetry(vehicleID)
	if err != nil {
		respondError(w, http.StatusNotFound, "no telemetry found for vehicle")
		return
	}

	queryMs := time.Since(start).Milliseconds()
	respondWithMeta(w, telemetry, &meta{QueryMs: queryMs})
}

func (s *Server) handleTelemetrySummary(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	vehicleID := vars["vehicle_id"]

	summary, err := s.db.GetTelemetrySummary(vehicleID)
	if err != nil {
		respondError(w, http.StatusNotFound, "no data found for vehicle")
		return
	}

	queryMs := time.Since(start).Milliseconds()
	respondWithMeta(w, summary, &meta{QueryMs: queryMs})
}

func (s *Server) handleGetDiagnostics(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		limit, _ = strconv.Atoi(v)
	}

	alerts, err := s.db.GetDiagnosticAlerts("", limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, alerts)
}

func (s *Server) handleGetVehicleDiagnostics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	vehicleID := vars["vehicle_id"]

	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		limit, _ = strconv.Atoi(v)
	}

	alerts, err := s.db.GetDiagnosticAlerts(vehicleID, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, alerts)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.db.GetStats()
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, stats)
}
