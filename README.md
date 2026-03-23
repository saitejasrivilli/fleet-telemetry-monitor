# 🚚 Fleet Telemetry Monitor

[![CI/CD](https://github.com/YOUR_USERNAME/fleet-telemetry-monitor/actions/workflows/ci.yml/badge.svg)](https://github.com/YOUR_USERNAME/fleet-telemetry-monitor/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org/)
[![C++](https://img.shields.io/badge/C++-17-00599C?logo=cplusplus)](https://isocpp.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

> A high-performance vehicle fleet telemetry monitoring system built with **Go**, **C++**, **REST APIs**, and **SQLite**.

🔗 **[Live Demo](https://fleet-telemetry-api.onrender.com)** | 📚 **[API Docs](#-rest-api-endpoints)** | 🐳 **[Docker](#-docker-deployment)**

---

## ✨ Features

| Feature | Description |
|---------|-------------|
| 🚗 **Real-time Telemetry** | GPS, speed, diagnostics, and sensor data ingestion |
| ⚡ **Sub-100ms Queries** | Optimized SQLite with strategic indexes |
| 🔥 **5x Faster Parsing** | C++ parser vs Python baseline |
| 🌐 **REST API** | Full CRUD with batch operations |
| 📊 **Live Dashboard** | Real-time monitoring UI |
| 🐳 **Docker Ready** | One-command deployment |

---

## 🚀 Quick Start

### Option 1: Docker (Recommended)

```bash
# Clone the repository
git clone https://github.com/YOUR_USERNAME/fleet-telemetry-monitor.git
cd fleet-telemetry-monitor

# Start with Docker Compose
docker-compose up -d

# Initialize with sample data
docker-compose --profile init up init-data

# Open dashboard at http://localhost:3000
# API available at http://localhost:8080
```

### Option 2: Local Development

```bash
# Prerequisites: Go 1.21+, g++, SQLite3

# Build Go CLI
go build -o fleet-monitor ./cmd/

# Build C++ parser
cd cpp-parser && make && cd ..

# Generate sample data
./fleet-monitor generate -c 10000 -n 10

# Start API server
./fleet-monitor server --port 8080
```

### Option 3: VS Code

1. Open the project folder in VS Code
2. Install recommended extensions (prompt will appear)
3. Press `F5` to run with debugging
4. Use the REST Client extension with `.vscode/api-tests.http`

---

## 📁 Project Structure

```
fleet-telemetry-monitor/
├── 📂 cmd/                    # Go CLI application
│   └── main.go
├── 📂 internal/               # Go packages
│   ├── api/                   # REST API handlers
│   ├── db/                    # SQLite database layer
│   ├── models/                # Data models
│   └── parser/                # Go file parser
├── 📂 cpp-parser/             # High-performance C++ parser
│   ├── telemetry_parser.h
│   ├── telemetry_parser.cpp
│   ├── main.cpp
│   └── Makefile
├── 📂 web/                    # Dashboard UI
├── 📂 .vscode/                # VS Code configuration
├── 📂 .github/workflows/      # CI/CD pipelines
├── 🐳 Dockerfile
├── 🐳 docker-compose.yml
└── 📄 render.yaml             # Render.com deployment
```

---

## 🔌 REST API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/api/v1/stats` | Database statistics |
| `GET` | `/api/v1/vehicles` | List all vehicles |
| `POST` | `/api/v1/vehicles` | Create vehicle |
| `GET` | `/api/v1/vehicles/{id}` | Get vehicle |
| `GET` | `/api/v1/telemetry` | Query telemetry |
| `POST` | `/api/v1/telemetry` | Insert record |
| `POST` | `/api/v1/telemetry/batch` | Batch insert |
| `GET` | `/api/v1/telemetry/latest/{id}` | Latest reading |
| `GET` | `/api/v1/telemetry/summary/{id}` | Aggregated stats |
| `GET` | `/api/v1/diagnostics` | Diagnostic alerts |

### Example Requests

```bash
# Create a vehicle
curl -X POST http://localhost:8080/api/v1/vehicles \
  -H "Content-Type: application/json" \
  -d '{"id":"VEH-001","name":"Truck 1","license_plate":"FL-1234","vehicle_type":"Truck"}'

# Insert telemetry
curl -X POST http://localhost:8080/api/v1/telemetry \
  -H "Content-Type: application/json" \
  -d '{
    "vehicle_id": "VEH-001",
    "latitude": 28.5383,
    "longitude": -81.3792,
    "speed": 65.5,
    "engine_rpm": 2800,
    "fuel_level": 72.5
  }'

# Query with filters
curl "http://localhost:8080/api/v1/telemetry?vehicle_id=VEH-001&limit=50"
```

---

## 🏎️ Performance Benchmarks

### C++ Parser vs Python

| Parser | 50K Records | Speed | Improvement |
|--------|-------------|-------|-------------|
| Python | 191 ms | 261K rec/sec | baseline |
| **C++** | **36 ms** | **1.39M rec/sec** | **5.3x faster** |

### Query Performance

| Query Type | Records | Response Time |
|------------|---------|---------------|
| By Vehicle ID | 10K | < 15ms |
| Time Range | 50K | < 45ms |
| With Filters | 100K | < 85ms |

Run benchmarks yourself:
```bash
# Generate data
python3 scripts/generate_data.py -n 100000 -f csv -o data/benchmark

# C++ benchmark
./cpp-parser/fleet_parser -B 5 data/benchmark.csv

# Python benchmark
python3 scripts/generate_data.py -n 100000 -b
```

---

## 🐳 Docker Deployment

### Docker Compose (Development)

```bash
docker-compose up -d                    # Start services
docker-compose --profile init up        # Initialize data
docker-compose logs -f api              # View logs
docker-compose down                     # Stop services
```

### Production Deployment

**Render.com** (Free tier available):
1. Fork this repository
2. Connect to [Render](https://render.com)
3. Create new "Blueprint" from `render.yaml`
4. Deploy!

**Railway**:
```bash
railway login
railway init
railway up
```

---

## 💻 VS Code Setup

This project includes full VS Code integration:

### Recommended Extensions
- Go (golang.go)
- C/C++ (ms-vscode.cpptools)
- REST Client (humao.rest-client)
- Docker (ms-azuretools.vscode-docker)
- SQLTools + SQLite (mtxr.sqltools)

### Available Tasks (`Ctrl+Shift+B`)
- Build Go CLI
- Build C++ Parser
- Run Tests
- Start Server
- Docker Build/Run

### Debug Configurations (`F5`)
- Go: Launch Server
- Go: Generate Data
- C++: Debug Parser
- C++: Benchmark

### API Testing
Open `.vscode/api-tests.http` and click "Send Request" above any request.

---

## 🧪 Testing

```bash
# Go tests
go test -v ./...

# Go tests with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# C++ parser test
./cpp-parser/fleet_parser -s data/sample_telemetry.csv
```

---

## 📊 Data Formats

### CSV
```csv
vehicle_id,timestamp,latitude,longitude,speed,heading,engine_rpm,fuel_level,odometer_km,engine_temp,battery_volt,diagnostic_code
VEH-001,2024-01-15T08:00:00,28.5383,-81.3792,65.5,180.0,2800,72.5,52341.2,85.2,12.8,
```

### JSON
```json
{
  "vehicle_id": "VEH-001",
  "timestamp": "2024-01-15T08:00:00",
  "latitude": 28.5383,
  "speed": 65.5,
  "diagnostic_code": "P0420"
}
```

---

## 🛠️ Tech Stack

| Component | Technology |
|-----------|------------|
| API Server | Go 1.21, Gorilla Mux |
| Database | SQLite (WAL mode) |
| Parser | C++17 (optimized) |
| Dashboard | HTML, Tailwind CSS, Chart.js |
| CI/CD | GitHub Actions |
| Containers | Docker, Docker Compose |
| Deployment | Render, Railway, Fly.io |

---

## 📝 License

MIT License - see [LICENSE](LICENSE) for details.

---

## 👤 Author

**Your Name**
- GitHub: [@saitejasrivilli](https://github.com/saitejasrivilli)
- LinkedIn: [saiteja-srivillibhutturu](https://www.linkedin.com/in/saitejasrivillibhutturu/)

---

<p align="center">
  <b>⭐ Star this repo if you found it useful!</b>
</p>
