cmake_minimum_required(VERSION 3.14)
project(fleet_telemetry_parser VERSION 1.0.0 LANGUAGES CXX)

# C++17 required for std::optional, std::string_view
set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Optimization flags for release builds
set(CMAKE_CXX_FLAGS_RELEASE "-O3 -march=native -DNDEBUG")
set(CMAKE_CXX_FLAGS_DEBUG "-g -O0 -Wall -Wextra")

# Default to Release build
if(NOT CMAKE_BUILD_TYPE)
    set(CMAKE_BUILD_TYPE Release)
endif()

# Source files
set(SOURCES
    telemetry_parser.cpp
    main.cpp
)

# Main executable
add_executable(fleet_parser ${SOURCES})

# Library for embedding in other projects
add_library(fleet_parser_lib STATIC telemetry_parser.cpp)
target_include_directories(fleet_parser_lib PUBLIC ${CMAKE_CURRENT_SOURCE_DIR})

# Install targets
install(TARGETS fleet_parser DESTINATION bin)
install(TARGETS fleet_parser_lib DESTINATION lib)
install(FILES telemetry_parser.h DESTINATION include/fleet)

# Benchmark executable (optional)
option(BUILD_BENCHMARK "Build benchmark tool" OFF)
if(BUILD_BENCHMARK)
    add_executable(fleet_benchmark benchmark.cpp telemetry_parser.cpp)
endif()

# Print build info
message(STATUS "Build type: ${CMAKE_BUILD_TYPE}")
message(STATUS "C++ Compiler: ${CMAKE_CXX_COMPILER}")
message(STATUS "C++ Flags: ${CMAKE_CXX_FLAGS_${CMAKE_BUILD_TYPE}}")
