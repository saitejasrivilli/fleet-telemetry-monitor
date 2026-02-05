#include "telemetry_parser.h"
#include <algorithm>
#include <cstring>
#include <cmath>
#include <iomanip>
#include <iostream>
#include <stdexcept>

namespace fleet {

// ============================================================================
// TelemetryData implementation
// ============================================================================

bool TelemetryData::is_valid() const {
    if (vehicle_id.empty()) return false;
    if (latitude < -90.0 || latitude > 90.0) return false;
    if (longitude < -180.0 || longitude > 180.0) return false;
    if (speed < 0) return false;
    if (fuel_level < 0 || fuel_level > 100) return false;
    if (engine_rpm < 0) return false;
    return true;
}

std::string TelemetryData::to_csv() const {
    std::ostringstream oss;
    oss << vehicle_id << ","
        << timestamp << ","
        << std::fixed << std::setprecision(6) << latitude << ","
        << longitude << ","
        << std::setprecision(2) << speed << ","
        << heading << ","
        << engine_rpm << ","
        << fuel_level << ","
        << odometer_km << ","
        << engine_temp << ","
        << battery_volt << ","
        << diagnostic_code;
    return oss.str();
}

std::string TelemetryData::to_json() const {
    std::ostringstream oss;
    oss << "{\"vehicle_id\":\"" << vehicle_id << "\","
        << "\"timestamp\":" << timestamp << ","
        << "\"latitude\":" << std::fixed << std::setprecision(6) << latitude << ","
        << "\"longitude\":" << longitude << ","
        << "\"speed\":" << std::setprecision(2) << speed << ","
        << "\"heading\":" << heading << ","
        << "\"engine_rpm\":" << engine_rpm << ","
        << "\"fuel_level\":" << fuel_level << ","
        << "\"odometer_km\":" << odometer_km << ","
        << "\"engine_temp\":" << engine_temp << ","
        << "\"battery_volt\":" << battery_volt;
    if (!diagnostic_code.empty()) {
        oss << ",\"diagnostic_code\":\"" << diagnostic_code << "\"";
    }
    oss << "}";
    return oss.str();
}

// ============================================================================
// TelemetryParser implementation
// ============================================================================

TelemetryParser::TelemetryParser(const ParserConfig& config) : config_(config) {
    reset_stats();
}

void TelemetryParser::reset_stats() {
    stats_ = ParseStats();
}

// Optimized string to double conversion
double TelemetryParser::fast_stod(const char* str, size_t len) {
    if (len == 0) return 0.0;
    
    double result = 0.0;
    double sign = 1.0;
    size_t i = 0;
    
    // Handle sign
    if (str[0] == '-') {
        sign = -1.0;
        i = 1;
    } else if (str[0] == '+') {
        i = 1;
    }
    
    // Integer part
    while (i < len && str[i] >= '0' && str[i] <= '9') {
        result = result * 10.0 + (str[i] - '0');
        i++;
    }
    
    // Decimal part
    if (i < len && str[i] == '.') {
        i++;
        double factor = 0.1;
        while (i < len && str[i] >= '0' && str[i] <= '9') {
            result += (str[i] - '0') * factor;
            factor *= 0.1;
            i++;
        }
    }
    
    return result * sign;
}

// Optimized string to int conversion
int TelemetryParser::fast_stoi(const char* str, size_t len) {
    if (len == 0) return 0;
    
    int result = 0;
    int sign = 1;
    size_t i = 0;
    
    if (str[0] == '-') {
        sign = -1;
        i = 1;
    } else if (str[0] == '+') {
        i = 1;
    }
    
    while (i < len && str[i] >= '0' && str[i] <= '9') {
        result = result * 10 + (str[i] - '0');
        i++;
    }
    
    return result * sign;
}

int64_t TelemetryParser::parse_timestamp(const std::string& str) {
    if (str.empty()) return 0;
    
    // Fast path: Unix timestamp (all digits)
    bool all_digits = true;
    for (char c : str) {
        if (c < '0' || c > '9') { all_digits = false; break; }
    }
    if (all_digits && str.length() <= 13) {
        int64_t result = 0;
        for (char c : str) result = result * 10 + (c - '0');
        return result;
    }
    
    // Fast ISO 8601 parser: 2024-01-15T10:30:00
    // Format: YYYY-MM-DDTHH:MM:SS (length 19)
    if (str.length() >= 19 && (str[10] == 'T' || str[10] == ' ')) {
        int year = (str[0]-'0')*1000 + (str[1]-'0')*100 + (str[2]-'0')*10 + (str[3]-'0');
        int month = (str[5]-'0')*10 + (str[6]-'0');
        int day = (str[8]-'0')*10 + (str[9]-'0');
        int hour = (str[11]-'0')*10 + (str[12]-'0');
        int min = (str[14]-'0')*10 + (str[15]-'0');
        int sec = (str[17]-'0')*10 + (str[18]-'0');
        
        // Simplified days calculation (approximate, good enough for telemetry)
        int64_t days = (year - 1970) * 365 + (year - 1969) / 4;
        static const int month_days[] = {0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334};
        days += month_days[month - 1] + day - 1;
        if (month > 2 && (year % 4 == 0)) days++;
        
        return (days * 86400 + hour * 3600 + min * 60 + sec) * 1000;
    }
    
    return 0;
}

void TelemetryParser::split_line(const std::string& line, std::vector<std::string_view>& fields) {
    fields.clear();
    size_t start = 0;
    size_t pos = 0;
    
    while (pos <= line.size()) {
        if (pos == line.size() || line[pos] == config_.delimiter) {
            fields.emplace_back(line.data() + start, pos - start);
            start = pos + 1;
        }
        pos++;
    }
}

void TelemetryParser::parse_header(const std::string& header) {
    std::vector<std::string_view> fields;
    split_line(header, fields);
    
    for (size_t i = 0; i < fields.size(); i++) {
        std::string field(fields[i]);
        // Convert to lowercase for comparison
        std::transform(field.begin(), field.end(), field.begin(), ::tolower);
        
        if (field == "vehicle_id") col_vehicle_id_ = i;
        else if (field == "timestamp") col_timestamp_ = i;
        else if (field == "latitude") col_latitude_ = i;
        else if (field == "longitude") col_longitude_ = i;
        else if (field == "speed") col_speed_ = i;
        else if (field == "heading") col_heading_ = i;
        else if (field == "engine_rpm") col_engine_rpm_ = i;
        else if (field == "fuel_level") col_fuel_level_ = i;
        else if (field == "odometer_km") col_odometer_km_ = i;
        else if (field == "engine_temp") col_engine_temp_ = i;
        else if (field == "battery_volt") col_battery_volt_ = i;
        else if (field == "diagnostic_code") col_diagnostic_code_ = i;
    }
}

std::optional<TelemetryData> TelemetryParser::parse_line(const std::string& line) {
    if (line.empty()) return std::nullopt;
    
    std::vector<std::string_view> fields;
    split_line(line, fields);
    
    if (fields.size() < 11) return std::nullopt;
    
    TelemetryData data;
    
    auto get_field = [&](int idx) -> std::string_view {
        if (idx >= 0 && idx < static_cast<int>(fields.size())) {
            return fields[idx];
        }
        return std::string_view();
    };
    
    data.vehicle_id = std::string(get_field(col_vehicle_id_));
    data.timestamp = parse_timestamp(std::string(get_field(col_timestamp_)));
    
    auto lat = get_field(col_latitude_);
    auto lon = get_field(col_longitude_);
    data.latitude = fast_stod(lat.data(), lat.size());
    data.longitude = fast_stod(lon.data(), lon.size());
    
    auto spd = get_field(col_speed_);
    data.speed = fast_stod(spd.data(), spd.size());
    
    auto hdg = get_field(col_heading_);
    data.heading = fast_stod(hdg.data(), hdg.size());
    
    auto rpm = get_field(col_engine_rpm_);
    data.engine_rpm = fast_stoi(rpm.data(), rpm.size());
    
    auto fuel = get_field(col_fuel_level_);
    data.fuel_level = fast_stod(fuel.data(), fuel.size());
    
    auto odo = get_field(col_odometer_km_);
    data.odometer_km = fast_stod(odo.data(), odo.size());
    
    auto temp = get_field(col_engine_temp_);
    data.engine_temp = fast_stod(temp.data(), temp.size());
    
    auto batt = get_field(col_battery_volt_);
    data.battery_volt = fast_stod(batt.data(), batt.size());
    
    if (col_diagnostic_code_ < static_cast<int>(fields.size())) {
        data.diagnostic_code = std::string(get_field(col_diagnostic_code_));
    }
    
    if (config_.validate && !data.is_valid()) {
        return std::nullopt;
    }
    
    return data;
}

std::vector<TelemetryData> TelemetryParser::parse_file(const std::string& filename) {
    auto start_time = std::chrono::high_resolution_clock::now();
    
    std::ifstream file(filename, std::ios::binary);
    if (!file.is_open()) {
        throw std::runtime_error("Failed to open file: " + filename);
    }
    
    // Get file size
    file.seekg(0, std::ios::end);
    size_t file_size = file.tellg();
    file.seekg(0, std::ios::beg);
    
    std::vector<TelemetryData> results;
    results.reserve(file_size / 100);  // Estimate: ~100 bytes per record
    
    std::string line;
    line.reserve(512);
    
    // Read and parse header
    if (config_.has_header && std::getline(file, line)) {
        parse_header(line);
        stats_.total_lines++;
    }
    
    // Parse data lines
    while (std::getline(file, line)) {
        stats_.total_lines++;
        stats_.bytes_processed += line.size() + 1;
        
        // Remove trailing whitespace
        while (!line.empty() && (line.back() == '\r' || line.back() == '\n' || line.back() == ' ')) {
            line.pop_back();
        }
        
        if (line.empty()) continue;
        
        auto data = parse_line(line);
        if (data.has_value()) {
            results.push_back(std::move(data.value()));
            stats_.valid_records++;
        } else {
            stats_.invalid_records++;
        }
    }
    
    auto end_time = std::chrono::high_resolution_clock::now();
    stats_.parse_time_ms = std::chrono::duration<double, std::milli>(end_time - start_time).count();
    stats_.records_per_second = (stats_.valid_records / stats_.parse_time_ms) * 1000.0;
    
    return results;
}

void TelemetryParser::parse_file_streaming(
    const std::string& filename,
    std::function<void(TelemetryData&&)> callback
) {
    auto start_time = std::chrono::high_resolution_clock::now();
    
    std::ifstream file(filename, std::ios::binary);
    if (!file.is_open()) {
        throw std::runtime_error("Failed to open file: " + filename);
    }
    
    std::string line;
    line.reserve(512);
    
    // Read and parse header
    if (config_.has_header && std::getline(file, line)) {
        parse_header(line);
        stats_.total_lines++;
    }
    
    // Parse and stream data
    while (std::getline(file, line)) {
        stats_.total_lines++;
        stats_.bytes_processed += line.size() + 1;
        
        while (!line.empty() && (line.back() == '\r' || line.back() == '\n' || line.back() == ' ')) {
            line.pop_back();
        }
        
        if (line.empty()) continue;
        
        auto data = parse_line(line);
        if (data.has_value()) {
            callback(std::move(data.value()));
            stats_.valid_records++;
        } else {
            stats_.invalid_records++;
        }
    }
    
    auto end_time = std::chrono::high_resolution_clock::now();
    stats_.parse_time_ms = std::chrono::duration<double, std::milli>(end_time - start_time).count();
    stats_.records_per_second = (stats_.valid_records / stats_.parse_time_ms) * 1000.0;
}

std::vector<TelemetryData> TelemetryParser::parse_log(const std::string& filename) {
    auto start_time = std::chrono::high_resolution_clock::now();
    
    std::ifstream file(filename, std::ios::binary);
    if (!file.is_open()) {
        throw std::runtime_error("Failed to open file: " + filename);
    }
    
    std::vector<TelemetryData> results;
    std::string line;
    
    while (std::getline(file, line)) {
        stats_.total_lines++;
        stats_.bytes_processed += line.size() + 1;
        
        // Skip comments and empty lines
        if (line.empty() || line[0] == '#') continue;
        
        // Parse log format: timestamp|vehicle_id|lat,lon|speed|rpm|fuel|odo|temp|batt|diag
        std::vector<std::string> parts;
        std::istringstream iss(line);
        std::string part;
        while (std::getline(iss, part, '|')) {
            parts.push_back(part);
        }
        
        if (parts.size() < 10) {
            stats_.invalid_records++;
            continue;
        }
        
        TelemetryData data;
        data.timestamp = parse_timestamp(parts[0]);
        data.vehicle_id = parts[1];
        
        // Parse lat,lon
        size_t comma = parts[2].find(',');
        if (comma != std::string::npos) {
            data.latitude = std::stod(parts[2].substr(0, comma));
            data.longitude = std::stod(parts[2].substr(comma + 1));
        }
        
        data.speed = std::stod(parts[3]);
        data.engine_rpm = std::stoi(parts[4]);
        data.fuel_level = std::stod(parts[5]);
        data.odometer_km = std::stod(parts[6]);
        data.engine_temp = std::stod(parts[7]);
        data.battery_volt = std::stod(parts[8]);
        
        if (parts.size() > 9 && !parts[9].empty()) {
            data.diagnostic_code = parts[9];
        }
        
        if (!config_.validate || data.is_valid()) {
            results.push_back(std::move(data));
            stats_.valid_records++;
        } else {
            stats_.invalid_records++;
        }
    }
    
    auto end_time = std::chrono::high_resolution_clock::now();
    stats_.parse_time_ms = std::chrono::duration<double, std::milli>(end_time - start_time).count();
    stats_.records_per_second = (stats_.valid_records / stats_.parse_time_ms) * 1000.0;
    
    return results;
}

std::vector<TelemetryData> TelemetryParser::parse_binary(const std::string& filename) {
    auto start_time = std::chrono::high_resolution_clock::now();
    
    std::ifstream file(filename, std::ios::binary);
    if (!file.is_open()) {
        throw std::runtime_error("Failed to open file: " + filename);
    }
    
    // Read and verify header
    uint32_t magic;
    uint8_t version;
    file.read(reinterpret_cast<char*>(&magic), sizeof(magic));
    file.read(reinterpret_cast<char*>(&version), sizeof(version));
    
    if (magic != 0x464C4554 || version != 1) {
        throw std::runtime_error("Invalid binary file format");
    }
    
    std::vector<TelemetryData> results;
    
    while (file.peek() != EOF) {
        TelemetryData data;
        
        // Read vehicle_id (length-prefixed string)
        uint8_t vid_len;
        file.read(reinterpret_cast<char*>(&vid_len), sizeof(vid_len));
        data.vehicle_id.resize(vid_len);
        file.read(data.vehicle_id.data(), vid_len);
        
        // Read fixed-size fields
        file.read(reinterpret_cast<char*>(&data.timestamp), sizeof(data.timestamp));
        file.read(reinterpret_cast<char*>(&data.latitude), sizeof(data.latitude));
        file.read(reinterpret_cast<char*>(&data.longitude), sizeof(data.longitude));
        file.read(reinterpret_cast<char*>(&data.speed), sizeof(data.speed));
        file.read(reinterpret_cast<char*>(&data.heading), sizeof(data.heading));
        file.read(reinterpret_cast<char*>(&data.engine_rpm), sizeof(data.engine_rpm));
        file.read(reinterpret_cast<char*>(&data.fuel_level), sizeof(data.fuel_level));
        file.read(reinterpret_cast<char*>(&data.odometer_km), sizeof(data.odometer_km));
        file.read(reinterpret_cast<char*>(&data.engine_temp), sizeof(data.engine_temp));
        file.read(reinterpret_cast<char*>(&data.battery_volt), sizeof(data.battery_volt));
        
        // Read diagnostic_code (length-prefixed string)
        uint8_t diag_len;
        file.read(reinterpret_cast<char*>(&diag_len), sizeof(diag_len));
        if (diag_len > 0) {
            data.diagnostic_code.resize(diag_len);
            file.read(data.diagnostic_code.data(), diag_len);
        }
        
        stats_.total_lines++;
        if (!config_.validate || data.is_valid()) {
            results.push_back(std::move(data));
            stats_.valid_records++;
        } else {
            stats_.invalid_records++;
        }
    }
    
    auto end_time = std::chrono::high_resolution_clock::now();
    stats_.parse_time_ms = std::chrono::duration<double, std::milli>(end_time - start_time).count();
    stats_.records_per_second = (stats_.valid_records / stats_.parse_time_ms) * 1000.0;
    
    return results;
}

// ============================================================================
// BinaryWriter implementation
// ============================================================================

BinaryWriter::BinaryWriter(const std::string& filename) 
    : file_(filename, std::ios::binary) {
    if (!file_.is_open()) {
        throw std::runtime_error("Failed to create file: " + filename);
    }
    
    // Write header
    file_.write(reinterpret_cast<const char*>(&MAGIC), sizeof(MAGIC));
    file_.write(reinterpret_cast<const char*>(&VERSION), sizeof(VERSION));
}

BinaryWriter::~BinaryWriter() {
    flush();
}

void BinaryWriter::write(const TelemetryData& data) {
    // Write vehicle_id (length-prefixed)
    uint8_t vid_len = static_cast<uint8_t>(std::min(data.vehicle_id.size(), size_t(255)));
    file_.write(reinterpret_cast<const char*>(&vid_len), sizeof(vid_len));
    file_.write(data.vehicle_id.data(), vid_len);
    
    // Write fixed-size fields
    file_.write(reinterpret_cast<const char*>(&data.timestamp), sizeof(data.timestamp));
    file_.write(reinterpret_cast<const char*>(&data.latitude), sizeof(data.latitude));
    file_.write(reinterpret_cast<const char*>(&data.longitude), sizeof(data.longitude));
    file_.write(reinterpret_cast<const char*>(&data.speed), sizeof(data.speed));
    file_.write(reinterpret_cast<const char*>(&data.heading), sizeof(data.heading));
    file_.write(reinterpret_cast<const char*>(&data.engine_rpm), sizeof(data.engine_rpm));
    file_.write(reinterpret_cast<const char*>(&data.fuel_level), sizeof(data.fuel_level));
    file_.write(reinterpret_cast<const char*>(&data.odometer_km), sizeof(data.odometer_km));
    file_.write(reinterpret_cast<const char*>(&data.engine_temp), sizeof(data.engine_temp));
    file_.write(reinterpret_cast<const char*>(&data.battery_volt), sizeof(data.battery_volt));
    
    // Write diagnostic_code (length-prefixed)
    uint8_t diag_len = static_cast<uint8_t>(std::min(data.diagnostic_code.size(), size_t(255)));
    file_.write(reinterpret_cast<const char*>(&diag_len), sizeof(diag_len));
    if (diag_len > 0) {
        file_.write(data.diagnostic_code.data(), diag_len);
    }
    
    records_written_++;
}

void BinaryWriter::write_batch(const std::vector<TelemetryData>& data) {
    for (const auto& d : data) {
        write(d);
    }
}

void BinaryWriter::flush() {
    file_.flush();
}

// ============================================================================
// Utility functions
// ============================================================================

std::string format_stats(const ParseStats& stats) {
    std::ostringstream oss;
    oss << "Parse Statistics:\n"
        << "  Total lines:      " << stats.total_lines << "\n"
        << "  Valid records:    " << stats.valid_records << "\n"
        << "  Invalid records:  " << stats.invalid_records << "\n"
        << "  Bytes processed:  " << stats.bytes_processed << "\n"
        << "  Parse time:       " << std::fixed << std::setprecision(2) 
        << stats.parse_time_ms << " ms\n"
        << "  Records/second:   " << std::fixed << std::setprecision(0) 
        << stats.records_per_second;
    return oss.str();
}

void benchmark_parser(const std::string& filename, int iterations) {
    std::cout << "Benchmarking parser on: " << filename << "\n";
    std::cout << "Iterations: " << iterations << "\n\n";
    
    double total_time = 0;
    size_t total_records = 0;
    
    for (int i = 0; i < iterations; i++) {
        TelemetryParser parser;
        auto results = parser.parse_file(filename);
        
        total_time += parser.get_stats().parse_time_ms;
        total_records = results.size();
        
        std::cout << "  Iteration " << (i + 1) << ": " 
                  << std::fixed << std::setprecision(2)
                  << parser.get_stats().parse_time_ms << " ms\n";
    }
    
    double avg_time = total_time / iterations;
    double records_per_sec = (total_records / avg_time) * 1000.0;
    
    std::cout << "\nResults:\n"
              << "  Records:          " << total_records << "\n"
              << "  Average time:     " << std::fixed << std::setprecision(2) 
              << avg_time << " ms\n"
              << "  Records/second:   " << std::fixed << std::setprecision(0) 
              << records_per_sec << "\n";
}

}  // namespace fleet
