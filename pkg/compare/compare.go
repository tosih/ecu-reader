package compare

import (
	"fmt"
	"strings"

	"github.com/pterm/pterm"
	"github.com/tosih/ecu-reader/pkg/models"
)

// CompareFiles compares maps between two ECU files
func CompareFiles(file1, file2, mapType string, readMap func(string, models.MapConfig) (*models.ECUMap, error)) {
	pterm.DefaultHeader.WithFullWidth().Println("ECU File Comparison")

	var selectedConfigs []models.MapConfig
	if mapType == "all" {
		selectedConfigs = models.MapConfigs
	} else {
		for _, cfg := range models.MapConfigs {
			if strings.Contains(strings.ToLower(cfg.Name), strings.ToLower(mapType)) {
				selectedConfigs = append(selectedConfigs, cfg)
			}
		}
	}

	for _, cfg := range selectedConfigs {
		pterm.Println()
		pterm.DefaultSection.Printf("Comparing: %s\n", cfg.Name)

		map1, err1 := readMap(file1, cfg)
		map2, err2 := readMap(file2, cfg)

		if err1 != nil || err2 != nil {
			pterm.Error.Println("Failed to read one or both maps")
			continue
		}

		// Calculate differences
		differences := compareMapData(map1.Data, map2.Data)
		displayComparison(map1, map2, differences, cfg)
	}
}

func compareMapData(data1, data2 [][]float64) [][]float64 {
	rows := len(data1)
	cols := len(data1[0])
	diff := make([][]float64, rows)

	for i := 0; i < rows; i++ {
		diff[i] = make([]float64, cols)
		for j := 0; j < cols; j++ {
			diff[i][j] = data2[i][j] - data1[i][j]
		}
	}

	return diff
}

func displayComparison(map1, map2 *models.ECUMap, diff [][]float64, cfg models.MapConfig) {
	// Show statistics
	var totalDiff, maxDiff, minDiff float64
	changedCells := 0

	for i := 0; i < cfg.Rows; i++ {
		for j := 0; j < cfg.Cols; j++ {
			d := diff[i][j]
			if d != 0 {
				changedCells++
				totalDiff += d
				if d > maxDiff {
					maxDiff = d
				}
				if d < minDiff {
					minDiff = d
				}
			}
		}
	}

	avgDiff := totalDiff / float64(changedCells)

	pterm.Info.Printf("Changed cells: %d / %d (%.1f%%)\n",
		changedCells, cfg.Rows*cfg.Cols,
		float64(changedCells)/float64(cfg.Rows*cfg.Cols)*100)
	pterm.Info.Printf("Average change: %.2f %s\n", avgDiff, cfg.Unit)
	pterm.Info.Printf("Max increase: %.2f %s\n", maxDiff, cfg.Unit)
	pterm.Info.Printf("Max decrease: %.2f %s\n", minDiff, cfg.Unit)

	// Visualize differences
	pterm.Println("\nDifference Map (File2 - File1):")
	visualizeDifferences(diff, cfg)
}

func visualizeDifferences(diff [][]float64, cfg models.MapConfig) {
	var result strings.Builder

	// Find max absolute difference for scaling
	maxAbs := 0.0
	for i := 0; i < cfg.Rows; i++ {
		for j := 0; j < cfg.Cols; j++ {
			abs := diff[i][j]
			if abs < 0 {
				abs = -abs
			}
			if abs > maxAbs {
				maxAbs = abs
			}
		}
	}

	// RPM header
	rpmStep := 8000 / cfg.Cols
	result.WriteString("    RPM → |")
	for j := 0; j < cfg.Cols; j++ {
		result.WriteString(fmt.Sprintf("%-6d", j*rpmStep))
	}
	result.WriteString("\n")
	result.WriteString("  Load%  |" + strings.Repeat("-", cfg.Cols*6) + "\n")

	// Data rows
	loadStep := 100 / cfg.Rows
	for i := 0; i < cfg.Rows; i++ {
		result.WriteString(fmt.Sprintf("   %3d ↓ |", i*loadStep))
		for j := 0; j < cfg.Cols; j++ {
			val := diff[i][j]
			symbol := getDiffSymbol(val, maxAbs)
			result.WriteString(symbol)
		}
		result.WriteString("\n")
	}

	// Legend
	result.WriteString("\nLegend: ")
	result.WriteString(pterm.FgBlue.Sprint("▼▼") + " Large Decrease  ")
	result.WriteString(pterm.FgCyan.Sprint("▼ ") + " Small Decrease  ")
	result.WriteString(pterm.FgGray.Sprint("··") + " No Change  ")
	result.WriteString(pterm.FgYellow.Sprint("▲ ") + " Small Increase  ")
	result.WriteString(pterm.FgRed.Sprint("▲▲") + " Large Increase")

	pterm.DefaultBox.Println(result.String())
}

func getDiffSymbol(val, maxAbs float64) string {
	if val == 0 {
		return pterm.FgGray.Sprint("·· ")
	}

	normalized := val / maxAbs

	if normalized < -0.5 {
		return pterm.FgBlue.Sprint("▼▼ ")
	} else if normalized < -0.1 {
		return pterm.FgCyan.Sprint("▼  ")
	} else if normalized > 0.5 {
		return pterm.FgRed.Sprint("▲▲ ")
	} else if normalized > 0.1 {
		return pterm.FgYellow.Sprint("▲  ")
	}

	return pterm.FgGray.Sprint("·  ")
}
