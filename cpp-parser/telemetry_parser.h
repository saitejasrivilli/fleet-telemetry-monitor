#ifndef TELEMETRY_PARSER_H
#define TELEMETRY_PARSER_H

#include <string>
#include <vector>
#include <cstdint>
#include <chrono>
#include <optional>
#include <functional>
#include <fstream>
#include <sstream>
#include <memory>

namespace fleet {

// Telemetry data structure - packed for memory efficiency
struct TelemetryData {
    std::string vehicle_id;
    int64_t timestamp;          // Unix timestamp in milliseconds
    double latitude;
    double longitude;
    double speed;               // km/h
    double heading;             // degrees
    int engine_rpm;
    double fuel_level;          // percentage
    double odometer_km;
    double engine_temp;         // Celsius
    double battery_volt;
    std::string diagnostic_code;
    
    // Validation
    bool is_valid() const;
    std::string to_csv() const;
    std::string to_json() const;
};

// Parser statistics
struct ParseStats {
    size_t total_lines = 0;
    size_t valid_records = 0;
    size_t invalid_records = 0;
    size_t bytes_processed = 0;
    double parse_time_ms = 0;
    double records_per_second = 0;
};

// Parser configuration
struct ParserConfig {
    bool validate = true;
    bool skip_invalid = true;
    size_t batch_size = 10000;
    char delimiter = ',';
    bool has_header = true;
    size_t buffer_size = 1024 * 1024;  // 1MB buffer for reading
};

// High-performance telemetry parser
class TelemetryParser {
public:
    explicit TelemetryParser(const ParserConfig& config = ParserConfig());
    ~TelemetryParser() = default;
    
    // Parse entire file
    std::vector<TelemetryData> parse_file(const std::string& filename);
    
    // Parse with callback (for streaming large files)
    void parse_file_streaming(
        const std::string& filename,
        std::function<void(TelemetryData&&)> callback
    );
    
    // Parse a single line
    std::optional<TelemetryData> parse_line(const std::string& line);
    
    // Parse binary format (custom high-performance format)
    std::vector<TelemetryData> parse_binary(const std::string& filename);
    
    // Parse log format: timestamp|vehicle_id|lat,lon|speed|rpm|fuel|odo|temp|batt|diag
    std::vector<TelemetryData> parse_log(const std::string& filename);
    
    // Get parse statistics
    const ParseStats& get_stats() const { return stats_; }
    
    // Reset statistics
    void reset_stats();

private:
    ParserConfig config_;
    ParseStats stats_;
    
    // Fast string to double conversion
    double fast_stod(const char* str, size_t len);
    
    // Fast string to int conversion
    int fast_stoi(const char* str, size_t len);
    
    // Parse timestamp
    int64_t parse_timestamp(const std::string& str);
    
    // Split line by delimiter (optimized)
    void split_line(const std::string& line, std::vector<std::string_view>& fields);
    
    // Map header to column indices
    void parse_header(const std::string& header);
    
    // Column indices from header
    int col_vehicle_id_ = 0;
    int col_timestamp_ = 1;
    int col_latitude_ = 2;
    int col_longitude_ = 3;
    int col_speed_ = 4;
    int col_heading_ = 5;
    int col_engine_rpm_ = 6;
    int col_fuel_level_ = 7;
    int col_odometer_km_ = 8;
    int col_engine_temp_ = 9;
    int col_battery_volt_ = 10;
    int col_diagnostic_code_ = 11;
};

// Binary file format for maximum performance
class BinaryWriter {
public:
    explicit BinaryWriter(const std::string& filename);
    ~BinaryWriter();
    
    void write(const TelemetryData& data);
    void write_batch(const std::vector<TelemetryData>& data);
    void flush();
    
    size_t records_written() const { return records_written_; }

private:
    std::ofstream file_;
    size_t records_written_ = 0;
    static constexpr uint32_t MAGIC = 0x464C4554;  // "FLET"
    static constexpr uint8_t VERSION = 1;
};

// Utility functions
std::string format_stats(const ParseStats& stats);
void benchmark_parser(const std::string& filename, int iterations = 5);

}  // namespace fleet

#endif  // TELEMETRY_PARSER_H
