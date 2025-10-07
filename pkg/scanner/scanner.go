package scanner

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/pterm/pterm"
)

// ScanResult holds information about a potential map location
type ScanResult struct {
	Offset     int
	Rows       int
	Cols       int
	DataType   string
	Endianness string
	Min        float64
	Max        float64
	Variance   float64
	Preview    string
}

// ScanForMaps scans a binary file for potential map locations
func ScanForMaps(filename string) {
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
	pterm.DefaultSection.Println("Potential Map Locations")

	var results []ScanResult

	// Scan for 8x8, 8x16, and 16x16 patterns
	sizes := []struct{ rows, cols int }{
		{8, 8},
		{8, 16},
		{16, 16},
	}

	// Try uint8 and uint16 with both endiannesses
	for _, size := range sizes {
		cellCount := size.rows * size.cols

		// Scan for uint8 values
		for offset := 0; offset < len(data)-cellCount; offset += 0x40 {
			if result := scanUint8(data, offset, size.rows, size.cols); result != nil {
				results = append(results, *result)
			}
		}

		// Scan for uint16 values (need 2 bytes per cell)
		byteCount := cellCount * 2
		for offset := 0; offset < len(data)-byteCount; offset += 0x40 {
			// Try little-endian
			if result := scanUint16(data, offset, size.rows, size.cols, binary.LittleEndian, "LE"); result != nil {
				results = append(results, *result)
			}
			// Try big-endian
			if result := scanUint16(data, offset, size.rows, size.cols, binary.BigEndian, "BE"); result != nil {
				results = append(results, *result)
			}
		}
	}

	// Display results in table
	displayResults(results)
}

func scanUint8(data []byte, offset int, rows int, cols int) *ScanResult {
	cellCount := rows * cols
	if offset+cellCount > len(data) {
		return nil
	}

	values := make([]float64, cellCount)
	for i := 0; i < cellCount; i++ {
		values[i] = float64(data[offset+i])
	}

	min, max, variance := calculateStats(values)

	// Check if variance is good enough
	if (max-min) < 10 || max == 0 {
		return nil
	}

	// Create preview
	preview := ""
	for i := 0; i < 8 && i < cellCount; i++ {
		preview += fmt.Sprintf("%02X ", data[offset+i])
	}

	return &ScanResult{
		Offset:     offset,
		Rows:       rows,
		Cols:       cols,
		DataType:   "uint8",
		Endianness: "N/A",
		Min:        min,
		Max:        max,
		Variance:   variance,
		Preview:    preview + "...",
	}
}

func scanUint16(data []byte, offset int, rows int, cols int, byteOrder binary.ByteOrder, endianness string) *ScanResult {
	cellCount := rows * cols
	byteCount := cellCount * 2
	if offset+byteCount > len(data) {
		return nil
	}

	values := make([]float64, cellCount)
	for i := 0; i < cellCount; i++ {
		val := byteOrder.Uint16(data[offset+i*2 : offset+i*2+2])
		values[i] = float64(val)
	}

	min, max, variance := calculateStats(values)

	// Check if variance is good enough (higher threshold for uint16)
	if (max-min) < 100 || max == 0 {
		return nil
	}

	// Create preview
	preview := ""
	for i := 0; i < 4 && i < cellCount; i++ {
		val := byteOrder.Uint16(data[offset+i*2 : offset+i*2+2])
		preview += fmt.Sprintf("%04X ", val)
	}

	return &ScanResult{
		Offset:     offset,
		Rows:       rows,
		Cols:       cols,
		DataType:   "uint16",
		Endianness: endianness,
		Min:        min,
		Max:        max,
		Variance:   variance,
		Preview:    preview + "...",
	}
}

func calculateStats(values []float64) (float64, float64, float64) {
	if len(values) == 0 {
		return 0, 0, 0
	}

	min := values[0]
	max := values[0]
	sum := 0.0

	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
		sum += v
	}

	avg := sum / float64(len(values))

	// Calculate variance
	variance := 0.0
	for _, v := range values {
		diff := v - avg
		variance += diff * diff
	}
	variance /= float64(len(values))

	return min, max, variance
}

func displayResults(results []ScanResult) {
	if len(results) == 0 {
		pterm.Info.Println("No potential maps found")
		return
	}

	tableData := pterm.TableData{
		{"Offset", "Size", "Type", "Endian", "Min", "Max", "Variance", "Preview"},
	}

	for _, result := range results {
		tableData = append(tableData, []string{
			fmt.Sprintf("0x%04X", result.Offset),
			fmt.Sprintf("%dx%d", result.Rows, result.Cols),
			result.DataType,
			result.Endianness,
			fmt.Sprintf("%.0f", result.Min),
			fmt.Sprintf("%.0f", result.Max),
			fmt.Sprintf("%.1f", result.Variance),
			result.Preview,
		})
	}

	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
	pterm.Info.Printf("\nFound %d potential map(s)\n", len(results))
}
