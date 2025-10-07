package scanner

import (
	"fmt"
	"io"
	"os"

	"github.com/pterm/pterm"
)

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

	var results [][]string
	results = append(results, []string{"Offset", "Size", "Preview", "Min", "Max", "Variance"})

	// Scan for 8x8, 8x16, and 16x16 patterns
	sizes := []struct{ rows, cols int }{
		{8, 8},
		{8, 16},
		{16, 16},
	}

	for _, size := range sizes {
		cellCount := size.rows * size.cols
		for offset := 0; offset < len(data)-cellCount; offset += 0x40 {
			if hasGoodVariance(data[offset : offset+cellCount]) {
				preview := ""
				for i := 0; i < 8 && i < cellCount; i++ {
					preview += fmt.Sprintf("%02X ", data[offset+i])
				}

				min, max, variance := getDetailedStats(data[offset : offset+cellCount])
				results = append(results, []string{
					fmt.Sprintf("0x%04X", offset),
					fmt.Sprintf("%dx%d", size.rows, size.cols),
					preview + "...",
					fmt.Sprintf("%d", min),
					fmt.Sprintf("%d", max),
					fmt.Sprintf("%.1f", variance),
				})
			}
		}
	}

	pterm.DefaultTable.WithHasHeader().WithData(results).Render()
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

func getDetailedStats(data []byte) (uint8, uint8, float64) {
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

	// Calculate variance
	variance := 0.0
	for _, b := range data {
		diff := float64(b) - avg
		variance += diff * diff
	}
	variance /= float64(len(data))

	return min, max, variance
}
