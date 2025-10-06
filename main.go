package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

// MapConfig defines the structure of a map in the ECU file
type MapConfig struct {
	Name     string
	Offset   int64
	Rows     int
	Cols     int
	DataType string
	Scale    float64
	Offset2  float64
	Unit     string
}

// ECUMap represents a 2D map from the ECU
type ECUMap struct {
	Config MapConfig
	Data   [][]float64
}

func main() {
	filename := flag.String("file", "", "ECU binary file to read")
	mapType := flag.String("map", "all", "Map type to display: fuel, spark, or all (default: all)")
	verbose := flag.Bool("v", false, "Verbose output showing raw values")
	scan := flag.Bool("scan", false, "Scan file for potential map locations")
	displayMode := flag.String("display", "symbols", "Display mode: symbols or values")
	flag.Parse()

	if *filename == "" {
		fmt.Println("Usage: ecu-reader -file <filename> [-map fuel|spark|all] [-display symbols|values] [-v] [-scan]")
		fmt.Println("\nOptions:")
		fmt.Println("  -file     Path to ECU binary file")
		fmt.Println("  -map      Map type to display: fuel, spark, or all (default: all)")
		fmt.Println("  -display  Display mode: symbols or values (default: symbols)")
		fmt.Println("  -v        Verbose mode - show raw hex values")
		fmt.Println("  -scan     Scan file to find potential map locations")
		os.Exit(1)
	}

	if *scan {
		scanForMaps(*filename)
		return
	}

	// Validate display mode
	if *displayMode != "symbols" && *displayMode != "values" {
		fmt.Println("Error: -display must be either 'symbols' or 'values'")
		os.Exit(1)
	}

	// Motronic map locations (identified from bin file scan)
	configs := []MapConfig{
		{
			Name:     "Main Fuel Map (Injection Time)",
			Offset:   0x6700,
			Rows:     8,
			Cols:     16,
			DataType: "uint8",
			Scale:    0.04, // Typical injection time scaling
			Offset2:  0,
			Unit:     "ms",
		},
		{
			Name:     "Fuel Map 2 (Secondary)",
			Offset:   0x6740,
			Rows:     8,
			Cols:     16,
			DataType: "uint8",
			Scale:    0.04,
			Offset2:  0,
			Unit:     "ms",
		},
		{
			Name:     "Ignition Timing Map",
			Offset:   0x6780,
			Rows:     8,
			Cols:     8,
			DataType: "uint8",
			Scale:    0.75,
			Offset2:  -24.0,
			Unit:     "°BTDC",
		},
		{
			Name:     "Idle/Low Load Map",
			Offset:   0x6800,
			Rows:     8,
			Cols:     16,
			DataType: "uint8",
			Scale:    0.04,
			Offset2:  0,
			Unit:     "ms",
		},
		{
			Name:     "Cold Start Enrichment",
			Offset:   0x6880,
			Rows:     8,
			Cols:     8,
			DataType: "uint8",
			Scale:    1.0,
			Offset2:  0,
			Unit:     "%",
		},
		{
			Name:     "Warmup Enrichment",
			Offset:   0x68C0,
			Rows:     8,
			Cols:     8,
			DataType: "uint8",
			Scale:    0.5,
			Offset2:  0,
			Unit:     "%",
		},
		{
			Name:     "Air/Fuel Ratio Target",
			Offset:   0x6D00,
			Rows:     8,
			Cols:     8,
			DataType: "uint8",
			Scale:    0.1,
			Offset2:  10.0,
			Unit:     "AFR",
		},
		{
			Name:     "Boost/Pressure Map",
			Offset:   0x6E00,
			Rows:     8,
			Cols:     8,
			DataType: "uint8",
			Scale:    1.0,
			Offset2:  0,
			Unit:     "kPa",
		},
		{
			Name:     "Throttle Position Map",
			Offset:   0x6F00,
			Rows:     8,
			Cols:     8,
			DataType: "uint8",
			Scale:    0.5,
			Offset2:  0,
			Unit:     "%",
		},
		{
			Name:     "Rev Limiter/Fuel Cut",
			Offset:   0x7000,
			Rows:     8,
			Cols:     8,
			DataType: "uint8",
			Scale:    50.0,
			Offset2:  0,
			Unit:     "RPM",
		},
	}

	// Filter configs based on map type
	var selectedConfigs []MapConfig
	for _, cfg := range configs {
		if *mapType == "all" ||
			(*mapType == "fuel" && strings.Contains(strings.ToLower(cfg.Name), "fuel")) ||
			(*mapType == "spark" && strings.Contains(strings.ToLower(cfg.Name), "ignition")) {
			selectedConfigs = append(selectedConfigs, cfg)
		}
	}

	// Read and display maps
	for i, cfg := range selectedConfigs {
		if i > 0 {
			fmt.Println()
		}
		ecuMap, err := readMap(*filename, cfg)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", cfg.Name, err)
			continue
		}
		renderMap(ecuMap, *verbose, *displayMode)
	}
}

func scanForMaps(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	fmt.Printf("File size: %d bytes (0x%X)\n\n", len(data), len(data))
	fmt.Println("Scanning for potential 8x8 map locations...")
	fmt.Println("Looking for areas with good variance (not all same values)\n")

	// Scan for potential 8x8 maps
	for offset := 0; offset < len(data)-64; offset += 0x40 {
		if hasGoodVariance(data[offset : offset+64]) {
			fmt.Printf("Offset 0x%04X: ", offset)

			// Show first row as preview
			for i := 0; i < 8; i++ {
				fmt.Printf("%02X ", data[offset+i])
			}
			fmt.Printf("...")

			// Calculate some stats
			min, max, avg := getStats(data[offset : offset+64])
			fmt.Printf(" [Min:%d Max:%d Avg:%.0f]\n", min, max, avg)
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("Hex dump of interesting regions:")
	fmt.Println(strings.Repeat("=", 70))

	// Show some specific regions known to contain maps in Motronic
	regions := []struct {
		start int
		name  string
	}{
		{0x6600, "Region 1 (0x6600)"},
		{0x6700, "Region 2 (0x6700)"},
		{0x6800, "Region 3 (0x6800)"},
		{0x7900, "Region 4 (0x7900)"},
		{0x7A00, "Region 5 (0x7A00)"},
	}

	for _, region := range regions {
		if region.start+128 <= len(data) {
			fmt.Printf("\n%s:\n", region.name)
			printHexDump(data, region.start, 128)
		}
	}
}

func hasGoodVariance(data []byte) bool {
	if len(data) < 2 {
		return false
	}

	min := data[0]
	max := data[0]

	for _, b := range data {
		if b < min {
			min = b
		}
		if b > max {
			max = b
		}
	}

	// Good variance if range is at least 10 and not all zeros
	return (max-min) >= 10 && max > 0
}

func getStats(data []byte) (uint8, uint8, float64) {
	if len(data) == 0 {
		return 0, 0, 0
	}

	min := data[0]
	max := data[0]
	sum := 0

	for _, b := range data {
		if b < min {
			min = b
		}
		if b > max {
			max = b
		}
		sum += int(b)
	}

	avg := float64(sum) / float64(len(data))
	return min, max, avg
}

func printHexDump(data []byte, offset, length int) {
	end := offset + length
	if end > len(data) {
		end = len(data)
	}

	for i := offset; i < end; i += 16 {
		fmt.Printf("  0x%04X: ", i)

		// Hex values
		for j := 0; j < 16 && i+j < end; j++ {
			fmt.Printf("%02X ", data[i+j])
		}

		// ASCII representation
		fmt.Print(" | ")
		for j := 0; j < 16 && i+j < end; j++ {
			if data[i+j] >= 32 && data[i+j] <= 126 {
				fmt.Printf("%c", data[i+j])
			} else {
				fmt.Print(".")
			}
		}
		fmt.Println()
	}
}

func readMap(filename string, cfg MapConfig) (*ECUMap, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	_, err = f.Seek(cfg.Offset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	data := make([][]float64, cfg.Rows)
	for i := 0; i < cfg.Rows; i++ {
		data[i] = make([]float64, cfg.Cols)
		for j := 0; j < cfg.Cols; j++ {
			var value float64

			if cfg.DataType == "uint8" {
				var rawValue uint8
				err := binary.Read(f, binary.LittleEndian, &rawValue)
				if err != nil {
					return nil, err
				}
				value = float64(rawValue)*cfg.Scale + cfg.Offset2
			} else {
				var rawValue uint16
				err := binary.Read(f, binary.LittleEndian, &rawValue)
				if err != nil {
					return nil, err
				}
				value = float64(rawValue)*cfg.Scale + cfg.Offset2
			}

			data[i][j] = value
		}
	}

	return &ECUMap{
		Config: cfg,
		Data:   data,
	}, nil
}

func renderMap(m *ECUMap, verbose bool) {
	width := m.Config.Cols*3 + 10
	if width < 40 {
		width = 40
	}

	fmt.Printf("╔" + strings.Repeat("═", width) + "╗\n")
	fmt.Printf("║ %-"+fmt.Sprintf("%d", width-2)+"s ║\n", m.Config.Name)
	fmt.Printf("║ Offset: 0x%04X | Size: %dx%d | Type: %s %-"+fmt.Sprintf("%d", width-45)+"s ║\n",
		m.Config.Offset, m.Config.Rows, m.Config.Cols, m.Config.DataType, "")
	fmt.Printf("╠" + strings.Repeat("═", width) + "╣\n")

	min, max := findMinMax(m.Data)

	if m.Config.Cols > 1 {
		fmt.Print("║ Load/RPM │")
		for j := 0; j < m.Config.Cols; j++ {
			fmt.Printf("%3d", j)
		}
		fmt.Println(" ║")
		fmt.Printf("╠══════════╪" + strings.Repeat("═", m.Config.Cols*3) + "═╣\n")
	} else {
		fmt.Print("║   RPM    │Val║\n")
		fmt.Printf("╠══════════╪═══╣\n")
	}

	for i := 0; i < m.Config.Rows; i++ {
		fmt.Printf("║   %4d   │", i)
		for j := 0; j < m.Config.Cols; j++ {
			value := m.Data[i][j]
			symbol := getSymbolForValue(value, min, max)
			fmt.Printf(" %s ", symbol)
		}
		fmt.Println(" ║")
	}

	fmt.Printf("╚══════════╧" + strings.Repeat("═", m.Config.Cols*3) + "═╝\n")
	fmt.Printf("Range: %.2f - %.2f %s\n", min, max, m.Config.Unit)
	fmt.Printf("Legend: \033[34m░\033[0m Low  \033[32m▒\033[0m Med  \033[33m▓\033[0m High  \033[31m█\033[0m Max\n")

	if verbose {
		fmt.Println("\nRaw Values:")
		for i := 0; i < m.Config.Rows; i++ {
			fmt.Printf("Row %d: ", i)
			for j := 0; j < m.Config.Cols; j++ {
				fmt.Printf("%.2f ", m.Data[i][j])
			}
			fmt.Println()
		}
	}
}

func findMinMax(data [][]float64) (float64, float64) {
	min := data[0][0]
	max := data[0][0]

	for _, row := range data {
		for _, val := range row {
			if val < min {
				min = val
			}
			if val > max {
				max = val
			}
		}
	}

	return min, max
}

func getSymbolForValue(value, min, max float64) string {
	if max == min {
		return "\033[37m·\033[0m" // Gray dot if all values are the same
	}

	normalized := (value - min) / (max - min)

	switch {
	case normalized < 0.25:
		return "\033[34m░\033[0m"
	case normalized < 0.5:
		return "\033[32m▒\033[0m"
	case normalized < 0.75:
		return "\033[33m▓\033[0m"
	default:
		return "\033[31m█\033[0m"
	}
}

func getColorCode(value, min, max float64) string {
	if max == min {
		return "\033[37m" // Gray if all values are the same
	}

	normalized := (value - min) / (max - min)

	switch {
	case normalized < 0.25:
		return "\033[34m" // Blue (low)
	case normalized < 0.5:
		return "\033[32m" // Green (medium-low)
	case normalized < 0.75:
		return "\033[33m" // Yellow (medium-high)
	default:
		return "\033[31m" // Red (high)
	}
}
