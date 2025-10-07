package renderer

import (
	"fmt"
	"strings"

	"github.com/pterm/pterm"
	"github.com/tosih/motronic-m21-tool/pkg/models"
)

// RenderMap displays a map with optional verbose output and display mode
func RenderMap(m *models.ECUMap, verbose bool, displayMode string, min, max float64) {
	title := fmt.Sprintf("%s | Offset: 0x%04X | %dx%d | Range: %.2f-%.2f %s",
		m.Config.Name, m.Config.Offset, m.Config.Rows, m.Config.Cols, min, max, m.Config.Unit)

	pterm.Info.Println(m.Config.Description)
	pterm.DefaultBox.WithTitle(title).WithTitleTopLeft().Println(BuildMapString(m, displayMode, min, max))
}

// BuildMapString creates a formatted string representation of the map
func BuildMapString(m *models.ECUMap, displayMode string, min, max float64) string {
	var result strings.Builder

	rpmStep := 8000 / m.Config.Cols
	loadStep := 100 / m.Config.Rows

	// Header
	result.WriteString("    RPM → |")
	for j := 0; j < m.Config.Cols; j++ {
		rpm := j * rpmStep
		if displayMode == "values" {
			result.WriteString(fmt.Sprintf("%6d", rpm))
		} else {
			result.WriteString(fmt.Sprintf("%-4d", rpm))
		}
	}
	result.WriteString("\n")

	// Separator
	sep := 6
	if displayMode != "values" {
		sep = 4
	}
	result.WriteString("  Load%  |" + strings.Repeat("-", m.Config.Cols*sep) + "\n")

	// Data rows
	for i := 0; i < m.Config.Rows; i++ {
		loadPct := i * loadStep
		result.WriteString(fmt.Sprintf("   %3d ↓ |", loadPct))
		for j := 0; j < m.Config.Cols; j++ {
			value := m.Data[i][j]
			if displayMode == "values" {
				color := getColorStyle(value, min, max)
				result.WriteString(color.Sprintf("%6.2f", value))
			} else if displayMode == "heatmap" {
				result.WriteString(getHeatmapBlock(value, min, max))
			} else {
				symbol := getSymbolForValue(value, min, max)
				result.WriteString(symbol + symbol + symbol + symbol)
			}
		}
		result.WriteString("\n")
	}

	// Legend
	if displayMode == "heatmap" {
		result.WriteString("\n" + getHeatmapLegend())
	} else if displayMode == "symbols" {
		result.WriteString("\nLegend: ")
		result.WriteString(pterm.FgCyan.Sprint("░") + " Low  ")
		result.WriteString(pterm.FgGreen.Sprint("▒") + " Med  ")
		result.WriteString(pterm.FgYellow.Sprint("▓") + " High  ")
		result.WriteString(pterm.FgRed.Sprint("█") + " Max")
	}

	return result.String()
}

func getHeatmapBlock(value, min, max float64) string {
	if max == min {
		return pterm.BgGray.Sprint("  ")
	}

	normalized := (value - min) / (max - min)

	switch {
	case normalized < 0.2:
		return pterm.NewStyle(pterm.BgBlue, pterm.FgWhite).Sprint("▄▄")
	case normalized < 0.4:
		return pterm.NewStyle(pterm.BgCyan, pterm.FgBlack).Sprint("▄▄")
	case normalized < 0.6:
		return pterm.NewStyle(pterm.BgGreen, pterm.FgBlack).Sprint("▄▄")
	case normalized < 0.8:
		return pterm.NewStyle(pterm.BgYellow, pterm.FgBlack).Sprint("▄▄")
	default:
		return pterm.NewStyle(pterm.BgRed, pterm.FgWhite).Sprint("▄▄")
	}
}

func getHeatmapLegend() string {
	var result strings.Builder
	result.WriteString("Heatmap: ")
	result.WriteString(pterm.NewStyle(pterm.BgBlue, pterm.FgWhite).Sprint("▄▄") + " Very Low  ")
	result.WriteString(pterm.NewStyle(pterm.BgCyan, pterm.FgBlack).Sprint("▄▄") + " Low  ")
	result.WriteString(pterm.NewStyle(pterm.BgGreen, pterm.FgBlack).Sprint("▄▄") + " Medium  ")
	result.WriteString(pterm.NewStyle(pterm.BgYellow, pterm.FgBlack).Sprint("▄▄") + " High  ")
	result.WriteString(pterm.NewStyle(pterm.BgRed, pterm.FgWhite).Sprint("▄▄") + " Very High")
	return result.String()
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

// ListAvailableMaps displays all available ECU maps in a table
func ListAvailableMaps() {
	pterm.DefaultHeader.WithFullWidth().Println("Available ECU Maps")

	data := [][]string{
		{"Name", "Offset", "Size", "Unit", "Description"},
	}

	for _, cfg := range models.MapConfigs {
		data = append(data, []string{
			cfg.Name,
			fmt.Sprintf("0x%04X", cfg.Offset),
			fmt.Sprintf("%dx%d", cfg.Rows, cfg.Cols),
			cfg.Unit,
			cfg.Description,
		})
	}

	pterm.DefaultTable.WithHasHeader().WithData(data).Render()
}

// DisplayMaps reads and displays the selected maps
func DisplayMaps(filename, mapType string, verbose bool, displayMode string, readMap func(string, models.MapConfig) (*models.ECUMap, error)) {
	// Select which maps to display
	var selectedConfigs []models.MapConfig
	switch mapType {
	case "fuel":
		selectedConfigs = []models.MapConfig{models.MapConfigs[0]}
	case "spark", "ignition":
		selectedConfigs = []models.MapConfig{models.MapConfigs[1]}
	case "lambda":
		selectedConfigs = []models.MapConfig{models.MapConfigs[2]}
	case "boost":
		selectedConfigs = []models.MapConfig{models.MapConfigs[3]}
	case "coldstart":
		selectedConfigs = []models.MapConfig{models.MapConfigs[4]}
	case "all":
		selectedConfigs = models.MapConfigs
	default:
		pterm.Error.Printf("Unknown map type: %s\n", mapType)
		return
	}

	pterm.DefaultHeader.WithFullWidth().
		WithBackgroundStyle(pterm.NewStyle(pterm.BgDarkGray)).
		WithTextStyle(pterm.NewStyle(pterm.FgLightWhite)).
		Println("ECU Map Reader - Motronic M2.1")

	pterm.Println()

	// Read and display maps
	for i, cfg := range selectedConfigs {
		if i > 0 {
			pterm.Println()
		}
		ecuMap, err := readMap(filename, cfg)
		if err != nil {
			pterm.Error.Printf("Error reading %s: %v\n", cfg.Name, err)
			continue
		}

		min, max := findMinMax(ecuMap.Data)
		RenderMap(ecuMap, verbose, displayMode, min, max)
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
