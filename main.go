package main

import (
	"encoding/binary"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pterm/pterm"
)

// MapConfig defines the structure of a map in the ECU file
type MapConfig struct {
	Name        string
	Offset      int64
	Rows        int
	Cols        int
	DataType    string
	Scale       float64
	Offset2     float64
	Unit        string
	Description string
}

// ECUMap represents a 2D map from the ECU
type ECUMap struct {
	Config MapConfig
	Data   [][]float64
}

// Predefined map configurations for Motronic M2.1
var mapConfigs = []MapConfig{
	{
		Name:        "Main Fuel Map",
		Offset:      0x6700,
		Rows:        8,
		Cols:        16,
		DataType:    "uint8",
		Scale:       0.04,
		Offset2:     0,
		Unit:        "ms",
		Description: "Primary fuel injection duration map",
	},
	{
		Name:        "Ignition Timing Map",
		Offset:      0x6780,
		Rows:        8,
		Cols:        16,
		DataType:    "uint8",
		Scale:       0.75,
		Offset2:     -24.0,
		Unit:        "deg",
		Description: "Spark advance timing map",
	},
	{
		Name:        "Lambda Target Map",
		Offset:      0x6800,
		Rows:        8,
		Cols:        16,
		DataType:    "uint8",
		Scale:       0.01,
		Offset2:     0.5,
		Unit:        "λ",
		Description: "Target air-fuel ratio map",
	},
	{
		Name:        "Boost Control Map",
		Offset:      0x7900,
		Rows:        8,
		Cols:        8,
		DataType:    "uint8",
		Scale:       0.1,
		Offset2:     0,
		Unit:        "bar",
		Description: "Wastegate duty cycle / boost target",
	},
	{
		Name:        "Cold Start Enrichment",
		Offset:      0x7A00,
		Rows:        8,
		Cols:        8,
		DataType:    "uint8",
		Scale:       0.02,
		Offset2:     0,
		Unit:        "%",
		Description: "Cold start fuel enrichment multiplier",
	},
}

func main() {
	filename := flag.String("file", "", "ECU binary file to read")
	mapType := flag.String("map", "all", "Map type to display: fuel, spark, lambda, boost, coldstart, or all")
	verbose := flag.Bool("v", false, "Verbose output showing raw values")
	scan := flag.Bool("scan", false, "Scan file for potential map locations")
	displayMode := flag.String("display", "heatmap", "Display mode: heatmap, symbols, or values")
	edit := flag.Bool("edit", false, "Enter interactive edit mode")
	preset := flag.String("preset", "", "Apply preset modification: revlimit, boost, etc.")
	exportPath := flag.String("export", "", "Export maps to CSV files in specified directory")
	importFile := flag.String("import", "", "Import map from CSV file")
	compare := flag.String("compare", "", "Compare current file with another ECU file")
	list := flag.Bool("list", false, "List all available maps")

	flag.Parse()

	if *filename == "" && !*list {
		pterm.Error.Println("Please specify an ECU file with -file flag")
		flag.Usage()
		os.Exit(1)
	}

	// List available maps
	if *list {
		listAvailableMaps()
		return
	}

	// Export maps to CSV
	if *exportPath != "" {
		exportMapsToCSV(*filename, *exportPath, *mapType)
		return
	}

	// Import map from CSV
	if *importFile != "" {
		importMapFromCSV(*filename, *importFile)
		return
	}

	// Compare two files
	if *compare != "" {
		compareFiles(*filename, *compare, *mapType)
		return
	}

	// File scanning mode
	if *scan {
		scanForMaps(*filename)
		return
	}

	// Interactive edit mode
	if *edit {
		interactiveEdit(*filename, false)
		return
	}

	// Apply preset modifications
	if *preset != "" {
		applyPreset(*filename, *preset, false)
		return
	}

	// Normal display mode
	displayMaps(*filename, *mapType, *verbose, *displayMode)
}

func listAvailableMaps() {
	pterm.DefaultHeader.WithFullWidth().Println("Available ECU Maps")

	data := [][]string{
		{"Name", "Offset", "Size", "Unit", "Description"},
	}

	for _, cfg := range mapConfigs {
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

func displayMaps(filename, mapType string, verbose bool, displayMode string) {
	// Select which maps to display
	var selectedConfigs []MapConfig
	switch mapType {
	case "fuel":
		selectedConfigs = []MapConfig{mapConfigs[0]}
	case "spark", "ignition":
		selectedConfigs = []MapConfig{mapConfigs[1]}
	case "lambda":
		selectedConfigs = []MapConfig{mapConfigs[2]}
	case "boost":
		selectedConfigs = []MapConfig{mapConfigs[3]}
	case "coldstart":
		selectedConfigs = []MapConfig{mapConfigs[4]}
	case "all":
		selectedConfigs = mapConfigs
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
		renderMap(ecuMap, verbose, displayMode)
	}
}

func exportMapsToCSV(filename, exportPath, mapType string) {
	// Create export directory if it doesn't exist
	if err := os.MkdirAll(exportPath, 0755); err != nil {
		pterm.Error.Printf("Failed to create export directory: %v\n", err)
		return
	}

	var selectedConfigs []MapConfig
	if mapType == "all" {
		selectedConfigs = mapConfigs
	} else {
		// Select specific map
		for _, cfg := range mapConfigs {
			if strings.Contains(strings.ToLower(cfg.Name), strings.ToLower(mapType)) {
				selectedConfigs = append(selectedConfigs, cfg)
			}
		}
	}

	spinner, _ := pterm.DefaultSpinner.Start("Exporting maps to CSV...")

	for _, cfg := range selectedConfigs {
		ecuMap, err := readMap(filename, cfg)
		if err != nil {
			spinner.Warning(fmt.Sprintf("Failed to read %s", cfg.Name))
			continue
		}

		// Create CSV filename
		csvFilename := filepath.Join(exportPath,
			strings.ReplaceAll(strings.ToLower(cfg.Name), " ", "_")+".csv")

		if err := exportMapToCSV(ecuMap, csvFilename); err != nil {
			spinner.Warning(fmt.Sprintf("Failed to export %s", cfg.Name))
			continue
		}
	}

	spinner.Success(fmt.Sprintf("Maps exported to %s", exportPath))
}

func exportMapToCSV(m *ECUMap, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write metadata as comments
	writer.Write([]string{fmt.Sprintf("# %s", m.Config.Name)})
	writer.Write([]string{fmt.Sprintf("# Offset: 0x%04X", m.Config.Offset)})
	writer.Write([]string{fmt.Sprintf("# Size: %dx%d", m.Config.Rows, m.Config.Cols)})
	writer.Write([]string{fmt.Sprintf("# Unit: %s", m.Config.Unit)})
	writer.Write([]string{""})

	// Write RPM header (column indices)
	rpmStep := 8000 / m.Config.Cols
	header := []string{"Load\\RPM"}
	for j := 0; j < m.Config.Cols; j++ {
		header = append(header, fmt.Sprintf("%d", j*rpmStep))
	}
	writer.Write(header)

	// Write data rows with load percentages
	loadStep := 100 / m.Config.Rows
	for i := 0; i < m.Config.Rows; i++ {
		row := []string{fmt.Sprintf("%d%%", i*loadStep)}
		for j := 0; j < m.Config.Cols; j++ {
			row = append(row, fmt.Sprintf("%.2f", m.Data[i][j]))
		}
		writer.Write(row)
	}

	return nil
}

func importMapFromCSV(ecuFilename, csvFilename string) {
	pterm.Info.Printf("Importing map from %s\n", csvFilename)

	// Read CSV file
	file, err := os.Open(csvFilename)
	if err != nil {
		pterm.Error.Printf("Failed to open CSV file: %v\n", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		pterm.Error.Printf("Failed to read CSV file: %v\n", err)
		return
	}

	// Parse CSV and find data start
	dataStart := 0
	for i, record := range records {
		if len(record) > 0 && strings.HasPrefix(record[0], "Load\\RPM") {
			dataStart = i + 1
			break
		}
	}

	if dataStart == 0 {
		pterm.Error.Println("Invalid CSV format: couldn't find data header")
		return
	}

	// TODO: Implement full CSV import with map identification
	pterm.Warning.Println("CSV import is under development")
}

func compareFiles(file1, file2, mapType string) {
	pterm.DefaultHeader.WithFullWidth().Println("ECU File Comparison")

	var selectedConfigs []MapConfig
	if mapType == "all" {
		selectedConfigs = mapConfigs
	} else {
		for _, cfg := range mapConfigs {
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

func displayComparison(map1, map2 *ECUMap, diff [][]float64, cfg MapConfig) {
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

func visualizeDifferences(diff [][]float64, cfg MapConfig) {
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

	title := fmt.Sprintf("%s | Offset: 0x%04X | %dx%d | Range: %.2f-%.2f %s",
		m.Config.Name, m.Config.Offset, m.Config.Rows, m.Config.Cols, min, max, m.Config.Unit)

	pterm.Info.Println(m.Config.Description)
	pterm.DefaultBox.WithTitle(title).WithTitleTopLeft().Println(buildMapString(m, displayMode, min, max))
}

func buildMapString(m *ECUMap, displayMode string, min, max float64) string {
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

func createBackup(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}

	timestamp := time.Now().Format("20060102_150405")
	backupName := filename + ".backup_" + timestamp
	err = os.WriteFile(backupName, data, 0644)
	if err != nil {
		return "", err
	}

	return backupName, nil
}

func interactiveEdit(filename string, dryRun bool) {
	pterm.DefaultHeader.WithFullWidth().
		WithBackgroundStyle(pterm.NewStyle(pterm.BgRed)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack)).
		Println("⚠️  INTERACTIVE EDIT MODE - USE WITH EXTREME CAUTION  ⚠️")

	pterm.Warning.Println("Modifying ECU calibration can cause engine damage, unsafe driving conditions, warranty void, and legal issues.")

	result, _ := pterm.DefaultInteractiveConfirm.Show("Do you understand the risks and want to proceed?")
	if !result {
		pterm.Info.Println("Edit cancelled.")
		return
	}

	options := []string{
		"Edit Rev Limiter",
		"Edit Fuel Map Cell",
		"Edit Ignition Map Cell",
		"Scale Entire Map",
		"Exit",
	}

	selectedOption, _ := pterm.DefaultInteractiveSelect.
		WithOptions(options).
		Show("Select what to edit:")

	switch selectedOption {
	case "Edit Rev Limiter":
		editRevLimiter(filename, dryRun)
	case "Edit Fuel Map Cell":
		editMapCell(filename, mapConfigs[0])
	case "Edit Ignition Map Cell":
		editMapCell(filename, mapConfigs[1])
	case "Scale Entire Map":
		scaleMap(filename, dryRun)
	case "Exit":
		pterm.Info.Println("Exiting edit mode.")
		return
	}
}

func editRevLimiter(filename string, dryRun bool) {
	pterm.Info.Println("Rev Limiter Editor")
	pterm.Warning.Println("Setting too high can cause catastrophic engine damage!")

	currentValue, _ := pterm.DefaultInteractiveTextInput.Show("Enter new RPM limit (e.g., 6500)")

	rpm := 0
	fmt.Sscanf(currentValue, "%d", &rpm)

	if rpm < 3000 || rpm > 7500 {
		pterm.Error.Println("Invalid RPM range. Must be between 3000-7500.")
		return
	}

	if dryRun {
		pterm.Warning.Println("DRY RUN - No changes made")
		return
	}

	result, _ := pterm.DefaultInteractiveConfirm.Show("Write this change to file?")
	if !result {
		pterm.Info.Println("Cancelled.")
		return
	}

	backup, err := createBackup(filename)
	if err != nil {
		pterm.Error.Printf("Failed to create backup: %v\n", err)
		return
	}
	pterm.Success.Printf("Backup created: %s\n", backup)

	data, _ := os.ReadFile(filename)
	scaled := uint8(rpm / 50)
	if len(data) > 0x7000 {
		data[0x7000] = scaled
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		pterm.Error.Printf("Failed to write: %v\n", err)
		return
	}

	pterm.Success.Println("Rev limiter updated successfully!")
}

func editMapCell(filename string, cfg MapConfig) {
	pterm.Info.Printf("Editing %s (%dx%d)\n", cfg.Name, cfg.Rows, cfg.Cols)

	rowStr, _ := pterm.DefaultInteractiveTextInput.Show(fmt.Sprintf("Enter row (0-%d)", cfg.Rows-1))
	colStr, _ := pterm.DefaultInteractiveTextInput.Show(fmt.Sprintf("Enter column (0-%d)", cfg.Cols-1))

	row, _ := strconv.Atoi(rowStr)
	col, _ := strconv.Atoi(colStr)

	if row < 0 || row >= cfg.Rows || col < 0 || col >= cfg.Cols {
		pterm.Error.Println("Invalid cell coordinates")
		return
	}

	f, _ := os.Open(filename)
	cellOffset := cfg.Offset + int64(row*cfg.Cols+col)
	f.Seek(cellOffset, io.SeekStart)
	var currentRaw uint8
	binary.Read(f, binary.LittleEndian, &currentRaw)
	f.Close()

	currentValue := float64(currentRaw)*cfg.Scale + cfg.Offset2
	pterm.Info.Printf("Current value at [%d,%d]: %.2f %s (raw: 0x%02X)\n", row, col, currentValue, cfg.Unit, currentRaw)

	newValueStr, _ := pterm.DefaultInteractiveTextInput.Show("Enter new value")
	newValue, _ := strconv.ParseFloat(newValueStr, 64)

	newRaw := uint8((newValue - cfg.Offset2) / cfg.Scale)
	pterm.Info.Printf("New value: %.2f %s (raw: 0x%02X)\n", newValue, cfg.Unit, newRaw)

	result, _ := pterm.DefaultInteractiveConfirm.Show("Write this change?")
	if !result {
		pterm.Info.Println("Cancelled.")
		return
	}

	backup, _ := createBackup(filename)
	pterm.Success.Printf("Backup created: %s\n", backup)

	data, _ := os.ReadFile(filename)
	data[cellOffset] = newRaw
	os.WriteFile(filename, data, 0644)

	pterm.Success.Println("Cell updated successfully!")
}

func scaleMap(filename string, dryRun bool) {
	pterm.Info.Println("Scale an entire map by a multiplier")
	pterm.Warning.Println("This modifies ALL cells in the selected map!")

	mapNames := []string{}
	for _, cfg := range mapConfigs {
		mapNames = append(mapNames, fmt.Sprintf("%s (0x%04X)", cfg.Name, cfg.Offset))
	}
	mapNames = append(mapNames, "Cancel")

	selectedOption, _ := pterm.DefaultInteractiveSelect.
		WithOptions(mapNames).
		Show("Select map to scale:")

	if selectedOption == "Cancel" {
		return
	}

	multiplierStr, _ := pterm.DefaultInteractiveTextInput.Show("Enter multiplier (e.g., 1.1 for +10%, 0.9 for -10%)")
	multiplier, _ := strconv.ParseFloat(multiplierStr, 64)

	if multiplier < 0.5 || multiplier > 2.0 {
		pterm.Error.Println("Multiplier out of safe range (0.5-2.0)")
		return
	}

	// Find selected config
	var selectedCfg MapConfig
	for _, cfg := range mapConfigs {
		if strings.Contains(selectedOption, cfg.Name) {
			selectedCfg = cfg
			break
		}
	}

	pterm.Info.Printf("Will multiply all values in %s by %.2f\n", selectedCfg.Name, multiplier)

	if dryRun {
		pterm.Warning.Println("DRY RUN - No changes made")
		return
	}

	result, _ := pterm.DefaultInteractiveConfirm.Show("Apply this scaling?")
	if !result {
		pterm.Info.Println("Cancelled.")
		return
	}

	backup, _ := createBackup(filename)
	pterm.Success.Printf("Backup created: %s\n", backup)

	data, _ := os.ReadFile(filename)
	for i := 0; i < selectedCfg.Rows*selectedCfg.Cols; i++ {
		cellOffset := int(selectedCfg.Offset) + i
		oldVal := data[cellOffset]
		newVal := uint8(float64(oldVal) * multiplier)
		data[cellOffset] = newVal
	}

	os.WriteFile(filename, data, 0644)
	pterm.Success.Println("Map scaled successfully!")
}

func applyPreset(filename, presetName string, dryRun bool) {
	pterm.DefaultHeader.WithFullWidth().
		WithBackgroundStyle(pterm.NewStyle(pterm.BgYellow)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack)).
		Println("PRESET MODIFICATION MODE")

	pterm.Warning.Println("Presets apply predefined changes. USE WITH CAUTION!")

	switch presetName {
	case "revlimit":
		editRevLimiter(filename, dryRun)
	case "fuel-enrich":
		applyFuelEnrichPreset(filename, dryRun)
	default:
		pterm.Error.Printf("Unknown preset: %s\n", presetName)
		pterm.Info.Println("Available presets: revlimit, fuel-enrich")
	}
}

func applyFuelEnrichPreset(filename string, dryRun bool) {
	pterm.Info.Println("Fuel Enrichment Preset: +5% across entire fuel map")

	if dryRun {
		pterm.Warning.Println("DRY RUN - Would increase fuel by 5%")
		return
	}

	result, _ := pterm.DefaultInteractiveConfirm.Show("Apply +5% fuel enrichment?")
	if !result {
		pterm.Info.Println("Cancelled.")
		return
	}

	backup, _ := createBackup(filename)
	pterm.Success.Printf("Backup created: %s\n", backup)

	cfg := mapConfigs[0] // Main fuel map
	data, _ := os.ReadFile(filename)

	for i := 0; i < cfg.Rows*cfg.Cols; i++ {
		cellOffset := int(cfg.Offset) + i
		oldVal := data[cellOffset]
		newVal := uint8(float64(oldVal) * 1.05)
		data[cellOffset] = newVal
	}

	os.WriteFile(filename, data, 0644)
	pterm.Success.Println("Fuel enrichment applied!")
}
