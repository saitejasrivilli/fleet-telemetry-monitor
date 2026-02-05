#!/usr/bin/env python3
"""
Generate sample telemetry data for testing Fleet Telemetry Monitor.
Demonstrates the baseline Python parsing speed for comparison with C++.
"""

import csv
import json
import random
import time
import argparse
from datetime import datetime, timedelta

# Diagnostic codes (OBD-II)
DIAGNOSTIC_CODES = [
    "", "", "", "", "", "", "", "",  # 80% chance of no code
    "P0420",  # Catalyst System Efficiency Below Threshold
    "P0171",  # System Too Lean
    "P0300",  # Random/Multiple Cylinder Misfire
    "P0442",  # Evaporative Emission Control System Leak
    "P0128",  # Coolant Thermostat
    "P0455",  # Evaporative Emission System Leak
]

VEHICLE_TYPES = ["Truck", "Van", "Sedan", "SUV", "Bus"]


def generate_telemetry_record(vehicle_id: str, timestamp: datetime, base_lat: float, base_lon: float, odometer: float):
    """Generate a single telemetry record."""
    return {
        "vehicle_id": vehicle_id,
        "timestamp": timestamp.isoformat(),
        "latitude": base_lat + (random.random() - 0.5) * 0.01,
        "longitude": base_lon + (random.random() - 0.5) * 0.01,
        "speed": random.random() * 120,
        "heading": random.random() * 360,
        "engine_rpm": 800 + random.randint(0, 5200),
        "fuel_level": 15 + random.random() * 85,
        "odometer_km": odometer + random.random() * 0.5,
        "engine_temp": 75 + random.random() * 35,
        "battery_volt": 11.5 + random.random() * 2.5,
        "diagnostic_code": random.choice(DIAGNOSTIC_CODES),
    }


def generate_csv(filename: str, num_records: int, num_vehicles: int):
    """Generate CSV telemetry file."""
    print(f"Generating {num_records:,} records to {filename}...")
    start_time = time.time()
    
    base_time = datetime.now() - timedelta(days=1)
    vehicles = [f"VEH-{i:03d}" for i in range(1, num_vehicles + 1)]
    odometers = {v: 50000 + random.randint(0, 100000) for v in vehicles}
    
    with open(filename, 'w', newline='') as f:
        writer = csv.writer(f)
        writer.writerow([
            "vehicle_id", "timestamp", "latitude", "longitude", "speed",
            "heading", "engine_rpm", "fuel_level", "odometer_km",
            "engine_temp", "battery_volt", "diagnostic_code"
        ])
        
        for i in range(num_records):
            vehicle = random.choice(vehicles)
            timestamp = base_time + timedelta(seconds=i)
            odometers[vehicle] += random.random() * 0.02
            
            record = generate_telemetry_record(
                vehicle, timestamp,
                28.5383, -81.3792,  # Orlando, FL
                odometers[vehicle]
            )
            
            writer.writerow([
                record["vehicle_id"],
                record["timestamp"],
                f"{record['latitude']:.6f}",
                f"{record['longitude']:.6f}",
                f"{record['speed']:.2f}",
                f"{record['heading']:.2f}",
                record["engine_rpm"],
                f"{record['fuel_level']:.2f}",
                f"{record['odometer_km']:.2f}",
                f"{record['engine_temp']:.2f}",
                f"{record['battery_volt']:.2f}",
                record["diagnostic_code"],
            ])
    
    elapsed = time.time() - start_time
    print(f"✓ Generated in {elapsed:.2f}s ({num_records/elapsed:,.0f} records/sec)")
    return filename


def generate_json(filename: str, num_records: int, num_vehicles: int):
    """Generate JSON telemetry file."""
    print(f"Generating {num_records:,} JSON records to {filename}...")
    start_time = time.time()
    
    base_time = datetime.now() - timedelta(days=1)
    vehicles = [f"VEH-{i:03d}" for i in range(1, num_vehicles + 1)]
    odometers = {v: 50000 + random.randint(0, 100000) for v in vehicles}
    
    records = []
    for i in range(num_records):
        vehicle = random.choice(vehicles)
        timestamp = base_time + timedelta(seconds=i)
        odometers[vehicle] += random.random() * 0.02
        
        record = generate_telemetry_record(
            vehicle, timestamp, 28.5383, -81.3792, odometers[vehicle]
        )
        records.append(record)
    
    with open(filename, 'w') as f:
        json.dump(records, f, indent=2)
    
    elapsed = time.time() - start_time
    print(f"✓ Generated in {elapsed:.2f}s ({num_records/elapsed:,.0f} records/sec)")
    return filename


def generate_log(filename: str, num_records: int, num_vehicles: int):
    """Generate custom log format file."""
    print(f"Generating {num_records:,} log records to {filename}...")
    start_time = time.time()
    
    base_time = datetime.now() - timedelta(days=1)
    vehicles = [f"VEH-{i:03d}" for i in range(1, num_vehicles + 1)]
    odometers = {v: 50000 + random.randint(0, 100000) for v in vehicles}
    
    with open(filename, 'w') as f:
        f.write("# Fleet Telemetry Log\n")
        f.write("# Format: timestamp|vehicle_id|lat,lon|speed|rpm|fuel|odo|temp|batt|diag\n")
        
        for i in range(num_records):
            vehicle = random.choice(vehicles)
            timestamp = base_time + timedelta(seconds=i)
            odometers[vehicle] += random.random() * 0.02
            
            r = generate_telemetry_record(
                vehicle, timestamp, 28.5383, -81.3792, odometers[vehicle]
            )
            
            line = (f"{r['timestamp']}|{r['vehicle_id']}|"
                   f"{r['latitude']:.6f},{r['longitude']:.6f}|"
                   f"{r['speed']:.2f}|{r['engine_rpm']}|{r['fuel_level']:.2f}|"
                   f"{r['odometer_km']:.2f}|{r['engine_temp']:.2f}|"
                   f"{r['battery_volt']:.2f}|{r['diagnostic_code']}\n")
            f.write(line)
    
    elapsed = time.time() - start_time
    print(f"✓ Generated in {elapsed:.2f}s ({num_records/elapsed:,.0f} records/sec)")
    return filename


def benchmark_python_parser(filename: str):
    """Benchmark Python CSV parsing for comparison with C++."""
    print(f"\nBenchmarking Python CSV parser on {filename}...")
    
    iterations = 3
    total_time = 0
    record_count = 0
    
    for i in range(iterations):
        start = time.time()
        records = []
        
        with open(filename, 'r') as f:
            reader = csv.DictReader(f)
            for row in reader:
                # Parse all fields (mimics C++ parser work)
                record = {
                    'vehicle_id': row['vehicle_id'],
                    'timestamp': row['timestamp'],
                    'latitude': float(row['latitude']),
                    'longitude': float(row['longitude']),
                    'speed': float(row['speed']),
                    'heading': float(row['heading']),
                    'engine_rpm': int(row['engine_rpm']),
                    'fuel_level': float(row['fuel_level']),
                    'odometer_km': float(row['odometer_km']),
                    'engine_temp': float(row['engine_temp']),
                    'battery_volt': float(row['battery_volt']),
                    'diagnostic_code': row['diagnostic_code'],
                }
                records.append(record)
        
        elapsed = time.time() - start
        total_time += elapsed
        record_count = len(records)
        print(f"  Iteration {i+1}: {elapsed*1000:.2f} ms")
    
    avg_time = total_time / iterations
    records_per_sec = record_count / avg_time
    
    print(f"\nPython Results:")
    print(f"  Records:        {record_count:,}")
    print(f"  Average time:   {avg_time*1000:.2f} ms")
    print(f"  Records/second: {records_per_sec:,.0f}")
    
    return avg_time * 1000  # Return ms for comparison


def main():
    parser = argparse.ArgumentParser(description="Generate sample telemetry data")
    parser.add_argument("-n", "--records", type=int, default=10000,
                       help="Number of records to generate")
    parser.add_argument("-v", "--vehicles", type=int, default=10,
                       help="Number of vehicles")
    parser.add_argument("-f", "--format", choices=["csv", "json", "log", "all"],
                       default="csv", help="Output format")
    parser.add_argument("-o", "--output", default="telemetry",
                       help="Output filename (without extension)")
    parser.add_argument("-b", "--benchmark", action="store_true",
                       help="Run Python parsing benchmark")
    
    args = parser.parse_args()
    
    print(f"🚀 Fleet Telemetry Data Generator")
    print(f"   Records:  {args.records:,}")
    print(f"   Vehicles: {args.vehicles}")
    print(f"   Format:   {args.format}\n")
    
    files = []
    
    if args.format in ["csv", "all"]:
        files.append(generate_csv(f"{args.output}.csv", args.records, args.vehicles))
    
    if args.format in ["json", "all"]:
        files.append(generate_json(f"{args.output}.json", args.records, args.vehicles))
    
    if args.format in ["log", "all"]:
        files.append(generate_log(f"{args.output}.log", args.records, args.vehicles))
    
    if args.benchmark and f"{args.output}.csv" in files:
        benchmark_python_parser(f"{args.output}.csv")
    
    print(f"\n✓ Done! Generated files: {', '.join(files)}")


if __name__ == "__main__":
    main()
