package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pterm/pterm"
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
		pterm.DefaultBox.WithTitle("ECU Map Reader").WithTitleTopCenter().Println(
			"Usage: ecu-reader -file <filename> [options]\n\n" +
				"Options:\n" +
				"  -file     Path to ECU binary file\n" +
				"  -map      Map type: fuel, spark, or all (default: all)\n" +
				"  -display  Display mode: symbols or values (default: symbols)\n" +
				"  -v        Verbose mode - show raw hex values\n" +
				"  -scan     Scan file to find potential map locations")
		os.Exit(1)
	}

	if *scan {
		scanForMaps(*filename)
		return
	}

	// Validate display mode
	if *displayMode != "symbols" && *displayMode != "values" {
		pterm.Error.Println("Display mode must be either 'symbols' or 'values'")
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
			Scale:    0.04,
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

	pterm.DefaultHeader.WithFullWidth().
		WithBackgroundStyle(pterm.NewStyle(pterm.BgDarkGray)).
		WithTextStyle(pterm.NewStyle(pterm.FgLightWhite)).
		Println("ECU Map Reader - Motronic")

	pterm.Println()

	// Read and display maps
	for i, cfg := range selectedConfigs {
		if i > 0 {
			pterm.Println()
		}
		ecuMap, err := readMap(*filename, cfg)
		if err != nil {
			pterm.Error.Printf("Error reading %s: %v\n", cfg.Name, err)
			continue
		}
		renderMap(ecuMap, *verbose, *displayMode)
	}
}

func scanForMaps(filename string) {
	spinner, _ := pterm.DefaultSpinner.Start("Scanning file for map locations...")

	f, err := os.Open(filename)
	if err != nil {
		spinner.Fail("Error opening file")
		pterm.Error.Printf("Error: %v\n", err)
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		spinner.Fail("Error reading file")
		pterm.Error.Printf("Error: %v\n", err)
		return
	}

	spinner.Success(fmt.Sprintf("File loaded: %d bytes (0x%X)", len(data), len(data)))

	pterm.Println()
	pterm.DefaultSection.Println("Potential 8x8 Map Locations")

	var results [][]string
	results = append(results, []string{"Offset", "Preview", "Min", "Max", "Avg"})

	for offset := 0; offset < len(data)-64; offset += 0x40 {
		if hasGoodVariance(data[offset : offset+64]) {
			preview := ""
			for i := 0; i < 8; i++ {
				preview += fmt.Sprintf("%02X ", data[offset+i])
			}
			preview += "..."

			min, max, avg := getStats(data[offset : offset+64])
			results = append(results, []string{
				fmt.Sprintf("0x%04X", offset),
				preview,
				fmt.Sprintf("%d", min),
				fmt.Sprintf("%d", max),
				fmt.Sprintf("%.0f", avg),
			})
		}
	}

	pterm.DefaultTable.WithHasHeader().WithData(results).Render()

	pterm.Println()
	pterm.DefaultSection.Println("Hex Dumps of Key Regions")

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
			pterm.Println()
			pterm.DefaultBox.WithTitle(region.name).Println(getHexDump(data, region.start, 128))
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

func getHexDump(data []byte, offset, length int) string {
	var result strings.Builder
	end := offset + length
	if end > len(data) {
		end = len(data)
	}

	for i := offset; i < end; i += 16 {
		result.WriteString(fmt.Sprintf("0x%04X: ", i))

		for j := 0; j < 16 && i+j < end; j++ {
			result.WriteString(fmt.Sprintf("%02X ", data[i+j]))
		}

		result.WriteString(" | ")
		for j := 0; j < 16 && i+j < end; j++ {
			if data[i+j] >= 32 && data[i+j] <= 126 {
				result.WriteString(fmt.Sprintf("%c", data[i+j]))
			} else {
				result.WriteString(".")
			}
		}
		result.WriteString("\n")
	}

	return result.String()
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

func renderMap(m *ECUMap, verbose bool, displayMode string) {
	min, max := findMinMax(m.Data)

	// Create title with info
	title := fmt.Sprintf("%s | Offset: 0x%04X | %dx%d | Range: %.1f-%.1f %s",
		m.Config.Name, m.Config.Offset, m.Config.Rows, m.Config.Cols, min, max, m.Config.Unit)

	pterm.DefaultBox.WithTitle(title).WithTitleTopLeft().Println(buildMapString(m, displayMode, min, max))

	if verbose {
		pterm.Println()
		pterm.Info.Println("Raw hex values:")
		f, err := os.Open("")
		if err == nil {
			f.Seek(m.Config.Offset, io.SeekStart)
			for i := 0; i < m.Config.Rows; i++ {
				fmt.Printf("Row %d: ", i)
				for j := 0; j < m.Config.Cols; j++ {
					var b uint8
					binary.Read(f, binary.LittleEndian, &b)
					fmt.Printf("0x%02X ", b)
				}
				fmt.Println()
			}
			f.Close()
		}
	}
}

func buildMapString(m *ECUMap, displayMode string, min, max float64) string {
	var result strings.Builder

	// Column headers
	if displayMode == "values" {
		result.WriteString("Load/RPM |")
		for j := 0; j < m.Config.Cols; j++ {
			result.WriteString(fmt.Sprintf("%5d", j))
		}
	} else {
		result.WriteString("Load/RPM |")
		for j := 0; j < m.Config.Cols; j++ {
			result.WriteString(fmt.Sprintf("%d", j%10))
		}
	}
	result.WriteString("\n")

	if displayMode == "values" {
		result.WriteString(strings.Repeat("-", 10) + "|" + strings.Repeat("-", m.Config.Cols*5) + "\n")
	} else {
		result.WriteString(strings.Repeat("-", 10) + "|" + strings.Repeat("-", m.Config.Cols) + "\n")
	}

	// Data rows
	for i := 0; i < m.Config.Rows; i++ {
		result.WriteString(fmt.Sprintf("  %4d   |", i))
		for j := 0; j < m.Config.Cols; j++ {
			value := m.Data[i][j]
			if displayMode == "values" {
				color := getColorStyle(value, min, max)
				result.WriteString(color.Sprintf("%5.1f", value))
			} else {
				symbol := getSymbolForValue(value, min, max)
				result.WriteString(symbol)
			}
		}
		result.WriteString("\n")
	}

	// Legend
	if displayMode == "symbols" {
		result.WriteString("\nLegend: ")
		result.WriteString(pterm.FgCyan.Sprint("░") + " Low  ")
		result.WriteString(pterm.FgGreen.Sprint("▒") + " Med  ")
		result.WriteString(pterm.FgYellow.Sprint("▓") + " High  ")
		result.WriteString(pterm.FgRed.Sprint("█") + " Max")
	}

	return result.String()
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
		return pterm.FgGray.Sprint("·")
	}

	normalized := (value - min) / (max - min)

	switch {
	case normalized < 0.25:
		return pterm.FgCyan.Sprint("░")
	case normalized < 0.5:
		return pterm.FgGreen.Sprint("▒")
	case normalized < 0.75:
		return pterm.FgYellow.Sprint("▓")
	default:
		return pterm.FgRed.Sprint("█")
	}
}

func getColorStyle(value, min, max float64) *pterm.Style {
	if max == min {
		return pterm.NewStyle(pterm.FgGray)
	}

	normalized := (value - min) / (max - min)

	switch {
	case normalized < 0.25:
		return pterm.NewStyle(pterm.FgCyan)
	case normalized < 0.5:
		return pterm.NewStyle(pterm.FgGreen)
	case normalized < 0.75:
		return pterm.NewStyle(pterm.FgYellow)
	default:
		return pterm.NewStyle(pterm.FgRed)
	}
}