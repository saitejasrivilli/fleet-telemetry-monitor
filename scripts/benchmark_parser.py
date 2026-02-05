#!/usr/bin/env python3
"""Python parser for benchmark comparison with C++"""
import csv
import sys
import time
import json

def parse_telemetry(filename):
    start = time.perf_counter()
    records = []
    
    with open(filename, 'r') as f:
        reader = csv.DictReader(f)
        for row in reader:
            record = {
                'vehicle_id': row['vehicle_id'],
                'latitude': float(row['latitude']),
                'longitude': float(row['longitude']),
                'speed': float(row['speed']),
                'engine_rpm': int(row['engine_rpm']),
                'fuel_level': float(row['fuel_level']),
            }
            records.append(record)
    
    elapsed_ms = (time.perf_counter() - start) * 1000
    
    return {
        'parser': 'Python',
        'records_parsed': len(records),
        'time_ms': round(elapsed_ms, 2),
        'records_per_sec': int(len(records) / (elapsed_ms / 1000)) if elapsed_ms > 0 else 0
    }

if __name__ == '__main__':
    if len(sys.argv) < 2:
        print(json.dumps({'error': 'No file provided'}))
        sys.exit(1)
    
    result = parse_telemetry(sys.argv[1])
    print(json.dumps(result))
