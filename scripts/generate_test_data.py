#!/usr/bin/env python3
"""Generate test data for benchmarking"""
import csv
import sys
import random
from datetime import datetime, timedelta

def generate(filename, num_records):
    with open(filename, 'w', newline='') as f:
        writer = csv.writer(f)
        writer.writerow([
            'vehicle_id', 'timestamp', 'latitude', 'longitude', 'speed',
            'heading', 'engine_rpm', 'fuel_level', 'odometer_km',
            'engine_temp', 'battery_volt', 'diagnostic_code'
        ])
        
        base_time = datetime.now()
        for i in range(num_records):
            writer.writerow([
                f'VEH-{(i % 10) + 1:03d}',
                (base_time + timedelta(seconds=i)).isoformat(),
                28.5 + random.random() * 0.1,
                -81.3 + random.random() * 0.1,
                random.random() * 120,
                random.random() * 360,
                1000 + random.randint(0, 5000),
                20 + random.random() * 80,
                50000 + random.randint(0, 50000),
                75 + random.random() * 30,
                11.5 + random.random() * 2,
                random.choice(['', '', '', '', 'P0420', 'P0171'])
            ])
    
    print(f"Generated {num_records} records to {filename}")

if __name__ == '__main__':
    num = int(sys.argv[1]) if len(sys.argv) > 1 else 1000
    filename = sys.argv[2] if len(sys.argv) > 2 else 'data/benchmark_data.csv'
    generate(filename, num)
