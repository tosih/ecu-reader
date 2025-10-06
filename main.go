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
	Name   string
	Offset int64
	Rows   int
	Cols   int
	Scale  float64
	Unit   string
}

// ECUMap represents a 2D map from the ECU
type ECUMap struct {
	Config MapConfig
	Data   [][]float64
}

func main() {
	filename := flag.String("file", "", "ECU binary file to read")
	mapType := flag.String("map", "all", "Map type: fuel, spark, or all")
	flag.Parse()

	if *filename == "" {
		fmt.Println("Usage: ecu-reader -file <filename> [-map fuel|spark|all]")
		os.Exit(1)
	}

	// Define map configurations (customize these for your ECU)
	configs := []MapConfig{
		{
			Name:   "Fuel Map (AFR)",
			Offset: 0x1000, // Example offset
			Rows:   16,
			Cols:   16,
			Scale:  0.1, // Scale factor to convert raw values
			Unit:   "AFR",
		},
		{
			Name:   "Spark Advance Map",
			Offset: 0x2000, // Example offset
			Rows:   16,
			Cols:   16,
			Scale:  0.5,
			Unit:   "°",
		},
	}

	// Filter configs based on map type
	var selectedConfigs []MapConfig
	for _, cfg := range configs {
		if *mapType == "all" ||
			(*mapType == "fuel" && strings.Contains(strings.ToLower(cfg.Name), "fuel")) ||
			(*mapType == "spark" && strings.Contains(strings.ToLower(cfg.Name), "spark")) {
			selectedConfigs = append(selectedConfigs, cfg)
		}
	}

	// Read and display maps
	for _, cfg := range selectedConfigs {
		ecuMap, err := readMap(*filename, cfg)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", cfg.Name, err)
			continue
		}
		renderMap(ecuMap)
		fmt.Println()
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
			var rawValue uint16
			err := binary.Read(f, binary.LittleEndian, &rawValue)
			if err != nil {
				return nil, err
			}
			data[i][j] = float64(rawValue) * cfg.Scale
		}
	}

	return &ECUMap{
		Config: cfg,
		Data:   data,
	}, nil
}

func renderMap(m *ECUMap) {
	fmt.Printf("╔═══════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║ %-57s ║\n", m.Config.Name)
	fmt.Printf("╠═══════════════════════════════════════════════════════════╣\n")

	// Find min and max for color scaling
	min, max := findMinMax(m.Data)

	// Render column headers
	fmt.Print("║ RPM/Load │")
	for j := 0; j < m.Config.Cols; j++ {
		fmt.Printf("%3d", j)
	}
	fmt.Println(" ║")
	fmt.Println("╠══════════╪" + strings.Repeat("═", m.Config.Cols*3) + "═╣")

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
	fmt.Printf("Range: %.1f - %.1f %s\n", min, max, m.Config.Unit)
	fmt.Printf("Legend: \033[34m░\033[0m Low  \033[32m▒\033[0m Med  \033[33m▓\033[0m High  \033[31m█\033[0m Max\n")
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
