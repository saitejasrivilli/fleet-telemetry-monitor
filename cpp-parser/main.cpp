#include "telemetry_parser.h"
#include <iostream>
#include <iomanip>
#include <cstring>
#include <getopt.h>

void print_usage(const char* program) {
    std::cout << "Fleet Telemetry Parser - High-Performance C++ Data Parser\n\n"
              << "Usage: " << program << " [options] <input_file>\n\n"
              << "Options:\n"
              << "  -f, --format <type>   Input format: csv, log, binary (default: csv)\n"
              << "  -o, --output <file>   Output file (JSON format)\n"
              << "  -b, --binary <file>   Convert to binary format for faster future parsing\n"
              << "  -v, --validate        Enable strict validation\n"
              << "  -n, --no-header       Input file has no header row\n"
              << "  -d, --delimiter <c>   Field delimiter (default: comma)\n"
              << "  -s, --stats           Show detailed statistics\n"
              << "  -B, --benchmark <n>   Benchmark with n iterations\n"
              << "  -h, --help            Show this help message\n\n"
              << "Examples:\n"
              << "  " << program << " telemetry.csv\n"
              << "  " << program << " -f log -o output.json sensor_data.log\n"
              << "  " << program << " -b fast_data.fbin telemetry.csv\n"
              << "  " << program << " -B 5 large_dataset.csv\n";
}

int main(int argc, char* argv[]) {
    // Options
    std::string format = "csv";
    std::string output_file;
    std::string binary_output;
    bool validate = false;
    bool has_header = true;
    char delimiter = ',';
    bool show_stats = false;
    int benchmark_iterations = 0;
    
    // Command line options
    static struct option long_options[] = {
        {"format",    required_argument, 0, 'f'},
        {"output",    required_argument, 0, 'o'},
        {"binary",    required_argument, 0, 'b'},
        {"validate",  no_argument,       0, 'v'},
        {"no-header", no_argument,       0, 'n'},
        {"delimiter", required_argument, 0, 'd'},
        {"stats",     no_argument,       0, 's'},
        {"benchmark", required_argument, 0, 'B'},
        {"help",      no_argument,       0, 'h'},
        {0, 0, 0, 0}
    };
    
    int opt;
    while ((opt = getopt_long(argc, argv, "f:o:b:vnd:sB:h", long_options, nullptr)) != -1) {
        switch (opt) {
            case 'f': format = optarg; break;
            case 'o': output_file = optarg; break;
            case 'b': binary_output = optarg; break;
            case 'v': validate = true; break;
            case 'n': has_header = false; break;
            case 'd': delimiter = optarg[0]; break;
            case 's': show_stats = true; break;
            case 'B': benchmark_iterations = std::stoi(optarg); break;
            case 'h':
                print_usage(argv[0]);
                return 0;
            default:
                print_usage(argv[0]);
                return 1;
        }
    }
    
    if (optind >= argc) {
        std::cerr << "Error: No input file specified\n\n";
        print_usage(argv[0]);
        return 1;
    }
    
    std::string input_file = argv[optind];
    
    // Benchmark mode
    if (benchmark_iterations > 0) {
        fleet::benchmark_parser(input_file, benchmark_iterations);
        return 0;
    }
    
    try {
        // Configure parser
        fleet::ParserConfig config;
        config.validate = validate;
        config.has_header = has_header;
        config.delimiter = delimiter;
        
        fleet::TelemetryParser parser(config);
        
        std::cout << "🚀 Fleet Telemetry Parser\n";
        std::cout << "   Input:  " << input_file << "\n";
        std::cout << "   Format: " << format << "\n\n";
        
        // Parse file
        std::vector<fleet::TelemetryData> data;
        
        if (format == "csv") {
            data = parser.parse_file(input_file);
        } else if (format == "log") {
            data = parser.parse_log(input_file);
        } else if (format == "binary") {
            data = parser.parse_binary(input_file);
        } else {
            std::cerr << "Error: Unknown format '" << format << "'\n";
            return 1;
        }
        
        const auto& stats = parser.get_stats();
        
        std::cout << "✓ Parsed " << stats.valid_records << " records in "
                  << std::fixed << std::setprecision(2) << stats.parse_time_ms << " ms\n";
        std::cout << "  Speed: " << std::fixed << std::setprecision(0) 
                  << stats.records_per_second << " records/second\n\n";
        
        // Show detailed stats
        if (show_stats) {
            std::cout << fleet::format_stats(stats) << "\n\n";
        }
        
        // Write JSON output
        if (!output_file.empty()) {
            std::ofstream out(output_file);
            if (!out.is_open()) {
                std::cerr << "Error: Cannot create output file\n";
                return 1;
            }
            
            out << "[\n";
            for (size_t i = 0; i < data.size(); i++) {
                out << "  " << data[i].to_json();
                if (i < data.size() - 1) out << ",";
                out << "\n";
            }
            out << "]\n";
            
            std::cout << "✓ Wrote JSON output to: " << output_file << "\n";
        }
        
        // Write binary output
        if (!binary_output.empty()) {
            fleet::BinaryWriter writer(binary_output);
            writer.write_batch(data);
            
            std::cout << "✓ Wrote binary output to: " << binary_output 
                      << " (" << writer.records_written() << " records)\n";
        }
        
        // Show sample if no output specified
        if (output_file.empty() && binary_output.empty() && !data.empty()) {
            std::cout << "Sample records (first 5):\n";
            for (size_t i = 0; i < std::min(size_t(5), data.size()); i++) {
                const auto& r = data[i];
                std::cout << "  [" << r.timestamp << "] " << r.vehicle_id 
                          << " | " << std::fixed << std::setprecision(4)
                          << r.latitude << "," << r.longitude
                          << " | " << std::setprecision(1) << r.speed << " km/h"
                          << " | RPM: " << r.engine_rpm
                          << " | Fuel: " << r.fuel_level << "%";
                if (!r.diagnostic_code.empty()) {
                    std::cout << " | ⚠️ " << r.diagnostic_code;
                }
                std::cout << "\n";
            }
        }
        
    } catch (const std::exception& e) {
        std::cerr << "Error: " << e.what() << "\n";
        return 1;
    }
    
    return 0;
}
