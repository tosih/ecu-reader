# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go CLI tool for reading, analyzing, and editing Motronic M2.1 ECU binary files. The application reads calibration maps (fuel, ignition timing, lambda, boost control, cold start enrichment) from binary files at predefined offsets and provides visualization, comparison, and editing capabilities.

## Development Commands

### Build and Run
```bash
# Build the application
go build -o motronic-m21-tool main.go

# Run directly with Go
go run main.go -file <path-to-binary>

# List available maps
go run main.go -list

# Display specific map types
go run main.go -file bins/file.bin -map fuel
go run main.go -file bins/file.bin -map spark
go run main.go -file bins/file.bin -map lambda
go run main.go -file bins/file.bin -map boost
go run main.go -file bins/file.bin -map coldstart
go run main.go -file bins/file.bin -map all

# Display modes
go run main.go -file bins/file.bin -display heatmap
go run main.go -file bins/file.bin -display symbols
go run main.go -file bins/file.bin -display values

# Scan file for potential map locations
go run main.go -file bins/file.bin -scan

# Export maps to CSV
go run main.go -file bins/file.bin -export ./output -map all

# Compare two ECU files
go run main.go -file bins/file1.bin -compare bins/file2.bin -map all

# Interactive edit mode (with warnings)
go run main.go -file bins/file.bin -edit

# Apply presets
go run main.go -file bins/file.bin -preset revlimit
go run main.go -file bins/file.bin -preset fuel-enrich
```

### Dependencies
```bash
# Install/update dependencies
go mod download
go mod tidy

# Main dependency: github.com/pterm/pterm for terminal UI
```

## Architecture

### Single-File Design
The entire application is contained in `main.go` (~1063 lines). This is intentional for simplicity - there are no packages or modules to navigate.

### Core Data Structures

**MapConfig** (line 18): Defines map metadata including:
- Offset: Memory location in binary file
- Dimensions: Rows x Cols
- DataType: uint8 or uint16
- Scale/Offset: Conversion factors from raw to real values
- Unit: Physical unit (ms, deg, λ, bar, %)

**ECUMap** (line 31): Runtime representation of a map with config and parsed float64 data.

**mapConfigs** (line 38): Global slice containing hardcoded configurations for 5 Motronic M2.1 maps:
1. Main Fuel Map (0x6700, 8x16, uint8)
2. Ignition Timing Map (0x6780, 8x16, uint8)
3. Lambda Target Map (0x6800, 8x16, uint8)
4. Boost Control Map (0x7900, 8x8, uint8)
5. Cold Start Enrichment (0x7A00, 8x8, uint8)

### Key Functions

**readMap()** (line 601): Core binary reading logic
- Opens file, seeks to offset
- Reads raw bytes as uint8/uint16
- Applies scale and offset transformations
- Returns populated ECUMap

**renderMap()** (line 645): Visualization dispatcher
- Calls buildMapString() with display mode
- Uses pterm for colored terminal output
- Supports heatmap, symbols, or numeric values modes

**displayMaps()** (line 183): Main display flow
- Filters mapConfigs based on user selection
- Reads and renders each selected map

**scanForMaps()** (line 491): Discovery tool
- Scans binary file in 0x40 byte increments
- Looks for patterns with good variance (min-max range ≥ 10)
- Tests 8x8, 8x16, and 16x16 dimensions
- Displays potential map locations with statistics

**compareFiles()** (line 340): Diff functionality
- Reads same map from two files
- Calculates cell-by-cell differences
- Visualizes changes with colored symbols

**Editing Functions** (lines 817-1062):
- `interactiveEdit()`: Menu-driven editor with safety confirmations
- `editRevLimiter()`: Modifies single-byte rev limit at 0x7000
- `editMapCell()`: Allows editing individual map cells
- `scaleMap()`: Multiplies entire map by factor
- `createBackup()`: Timestamped backup creation
- All edits require user confirmation and create backups

### Binary File Format
- Little-endian byte order
- Fixed memory offsets for known maps
- Raw values stored as uint8 or uint16
- Real values calculated as: `real = raw * scale + offset`
- Example: Fuel map raw value 100 → 100 * 0.04 + 0 = 4.0 ms

### Display Visualization
Three modes implemented:
1. **Heatmap**: Color-coded blocks using pterm background colors
2. **Symbols**: ASCII characters (░▒▓█) representing intensity
3. **Values**: Numeric display with color coding

All visualizations include:
- RPM axis (0-8000, divided by columns)
- Load axis (0-100%, divided by rows)
- Color legends

## Safety Considerations

This tool modifies ECU calibration data that directly controls engine behavior. The code includes multiple safety features:
- Interactive confirmation prompts before any write
- Automatic timestamped backups before modifications
- Range validation on inputs (e.g., RPM 3000-7500)
- Prominent warning headers in edit modes
- Dry-run capability (though not fully implemented)

When working on editing features, maintain these safety patterns and never bypass user confirmations.

## Binary File Locations

Sample ECU binaries are expected in the `bins/` directory (gitignored). The `scratch/` directory exists for temporary working files.
