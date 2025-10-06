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
	DataType string // "uint8" or "uint16"
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
	mapType := flag.String("map", "all", "Map type: fuel, spark, or all")
	verbose := flag.Bool("v", false, "Verbose output showing raw values")
	flag.Parse()

	if *filename == "" {
		fmt.Println("Usage: ecu-reader -file <filename> [-map fuel|spark|all] [-v]")
		fmt.Println("\nOptions:")
		fmt.Println("  -file    Path to ECU binary file")
		fmt.Println("  -map     Map type to display: fuel, spark, or all (default: all)")
		fmt.Println("  -v       Verbose mode - show raw hex values")
		os.Exit(1)
	}

	// Motronic map configurations
	// These offsets are based on typical M2.7/M2.8 layout
	configs := []MapConfig{
		{
			Name:     "Main Fuel Map",
			Offset:   0x79C0, // Typical main fuel map location
			Rows:     8,
			Cols:     8,
			DataType: "uint8",
			Scale:    0.78125, // 200/256 for fuel maps
			Offset2:  0,
			Unit:     "ms",
		},
		{
			Name:     "Ignition Timing Map",
			Offset:   0x7A00, // Typical ignition map location
			Rows:     8,
			Cols:     8,
			DataType: "uint8",
			Scale:    0.75,
			Offset2:  -24.0, // Offset to get proper degrees
			Unit:     "°BTDC",
		},
		{
			Name:     "Full Load Enrichment",
			Offset:   0x7A40,
			Rows:     8,
			Cols:     1,
			DataType: "uint8",
			Scale:    0.78125,
			Offset2:  0,
			Unit:     "ms",
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
		renderMap(ecuMap, *verbose)
	}
}

func readMap(filename string, cfg MapConfig) (*ECUMap, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Seek to the map offset
	_, err = f.Seek(cfg.Offset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	// Read the map data
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

	// Find min and max for color scaling
	min, max := findMinMax(m.Data)

	// Render column headers
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

	// Render each row
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
	// Normalize value between 0 and 1
	normalized := (value - min) / (max - min)

	// Return colored density symbols
	switch {
	case normalized < 0.25:
		return "\033[34m░\033[0m" // Blue light shade (low)
	case normalized < 0.5:
		return "\033[32m▒\033[0m" // Green medium shade (medium-low)
	case normalized < 0.75:
		return "\033[33m▓\033[0m" // Yellow dark shade (medium-high)
	default:
		return "\033[31m█\033[0m" // Red full block (high)
	}
}

