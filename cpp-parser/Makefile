# Fleet Telemetry Parser Makefile

CXX = g++
CXXFLAGS = -std=c++17 -O3 -march=native -Wall -Wextra
DEBUG_FLAGS = -std=c++17 -g -O0 -Wall -Wextra -DDEBUG

SRCS = telemetry_parser.cpp main.cpp
OBJS = $(SRCS:.cpp=.o)
TARGET = fleet_parser

.PHONY: all clean debug benchmark install

all: $(TARGET)

$(TARGET): $(OBJS)
	$(CXX) $(CXXFLAGS) -o $@ $^

%.o: %.cpp telemetry_parser.h
	$(CXX) $(CXXFLAGS) -c -o $@ $<

debug: CXXFLAGS = $(DEBUG_FLAGS)
debug: clean $(TARGET)

# Static library
libfleet_parser.a: telemetry_parser.o
	ar rcs $@ $^

# Run benchmark
benchmark: $(TARGET)
	@echo "Generating test data..."
	@python3 -c "import csv; import random; f=open('test_data.csv','w'); w=csv.writer(f); \
		w.writerow(['vehicle_id','timestamp','latitude','longitude','speed','heading','engine_rpm','fuel_level','odometer_km','engine_temp','battery_volt','diagnostic_code']); \
		[w.writerow(['VEH-'+str(i%10).zfill(3), 1704067200+i, 28.5+random.random()*0.1, -81.3+random.random()*0.1, \
		random.random()*120, random.random()*360, 1000+random.randint(0,5000), 20+random.random()*80, \
		50000+random.randint(0,100000), 80+random.random()*30, 12+random.random()*2, '' if random.random()>0.1 else 'P0420']) \
		for i in range(100000)]; f.close()" 2>/dev/null || echo "Skipping Python data generation"
	@if [ -f test_data.csv ]; then \
		echo "Running benchmark..."; \
		./$(TARGET) -B 5 test_data.csv; \
	fi

clean:
	rm -f $(OBJS) $(TARGET) libfleet_parser.a test_data.csv

install: $(TARGET)
	install -d $(DESTDIR)/usr/local/bin
	install -m 755 $(TARGET) $(DESTDIR)/usr/local/bin/
