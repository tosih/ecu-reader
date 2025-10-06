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
	edit := flag.Bool("edit", false, "Enter interactive edit mode")
	preset := flag.String("preset", "", "Apply preset modification: revlimit, boost, etc.")
	dryRun := flag.Bool("dry-run", false, "Preview changes without writing")
	flag.Parse()

	if *filename == "" {
		pterm.DefaultBox.WithTitle("ECU Map Reader").WithTitleTopCenter().Println(
			"Usage: ecu-reader -file <filename> [options]\n\n" +
				"Options:\n" +
				"  -file     Path to ECU binary file\n" +
				"  -map      Map type: fuel, spark, or all (default: all)\n" +
				"  -display  Display mode: symbols or values (default: symbols)\n" +
				"  -v        Verbose mode - show raw hex values\n" +
				"  -scan     Scan file to find potential map locations\n" +
				"  -edit     Enter interactive edit mode\n" +
				"  -preset   Apply preset: revlimit, boost, fuel-enrich\n" +
				"  -dry-run  Preview changes without writing to file")
		os.Exit(1)
	}

	if *scan {
		scanForMaps(*filename)
		return
	}

	if *edit {
		interactiveEdit(*filename, *dryRun)
		return
	}

	if *preset != "" {
		applyPreset(*filename, *preset, *dryRun)
		return
	}

	// Validate display mode
	if *displayMode != "symbols" && *displayMode != "values" {
		pterm.Error.Println("Display mode must be either 'symbols' or 'values'")
		os.Exit(1)
	}

	// Motronic M2.1 map locations (verified from file scan)
	configs := []MapConfig{
		{
			Name:     "Main Fuel Map",
			Offset:   0x6700,
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
			Cols:     16,
			DataType: "uint8",
			Scale:    0.75,
			Offset2:  -24.0,
			Unit:     "°BTDC",
		},
		{
			Name:     "Idle Fuel Map",
			Offset:   0x6800,
			Rows:     8,
			Cols:     16,
			DataType: "uint8",
			Scale:    0.04,
			Offset2:  0,
			Unit:     "ms",
		},
		{
			Name:     "Warmup/Enrichment Table",
			Offset:   0x6880,
			Rows:     8,
			Cols:     8,
			DataType: "uint8",
			Scale:    0.5,
			Offset2:  0,
			Unit:     "%",
		},
		{
			Name:     "Lambda/AFR Map",
			Offset:   0x6D00,
			Rows:     8,
			Cols:     8,
			DataType: "uint8",
			Scale:    0.1,
			Offset2:  10.0,
			Unit:     "AFR",
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
		Println("ECU Map Reader - Motronic M2.1")

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

	// Generate RPM axis labels (0-8000 RPM)
	rpmStep := 8000 / m.Config.Cols
	// Generate Load axis labels (0-100%)
	loadStep := 100 / m.Config.Rows

	// Column headers with RPM values
	result.WriteString("    RPM → |")
	for j := 0; j < m.Config.Cols; j++ {
		rpm := j * rpmStep
		if displayMode == "values" {
			result.WriteString(fmt.Sprintf("%4d", rpm))
		} else {
			result.WriteString(fmt.Sprintf("%-4d", rpm))
		}
	}
	result.WriteString("\n")

	// Separator
	if displayMode == "values" {
		result.WriteString("  Load%  |" + strings.Repeat("-", m.Config.Cols*4) + "\n")
	} else {
		result.WriteString("  Load%  |" + strings.Repeat("-", m.Config.Cols*4) + "\n")
	}

	// Data rows with load percentage labels
	for i := 0; i < m.Config.Rows; i++ {
		loadPct := i * loadStep
		result.WriteString(fmt.Sprintf("   %3d ↓ |", loadPct))
		for j := 0; j < m.Config.Cols; j++ {
			value := m.Data[i][j]
			if displayMode == "values" {
				color := getColorStyle(value, min, max)
				result.WriteString(color.Sprintf("%4.1f", value))
			} else {
				// 1x4 cell: 4 characters wide
				symbol := getSymbolForValue(value, min, max)
				result.WriteString(symbol + symbol + symbol + symbol)
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

// createBackup creates a timestamped backup of the original file
func createBackup(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}

	backupName := filename + ".backup"
	err = os.WriteFile(backupName, data, 0644)
	if err != nil {
		return "", err
	}

	return backupName, nil
}

// interactiveEdit provides an interactive menu to edit map values
func interactiveEdit(filename string, dryRun bool) {
	pterm.DefaultHeader.WithFullWidth().
		WithBackgroundStyle(pterm.NewStyle(pterm.BgRed)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack)).
		Println("⚠️  INTERACTIVE EDIT MODE - USE WITH EXTREME CAUTION  ⚠️")

	pterm.Warning.Println("Modifying ECU calibration can cause:\n" +
		"  • Engine damage or failure\n" +
		"  • Unsafe driving conditions\n" +
		"  • Warranty void\n" +
		"  • Legal issues (emissions)\n")

	result, _ := pterm.DefaultInteractiveConfirm.Show("Do you understand the risks and want to proceed?")
	if !result {
		pterm.Info.Println("Edit cancelled.")
		return
	}

	pterm.Println()

	options := []string{
		"Edit Rev Limiter",
		"Edit Fuel Map Cell",
		"Edit Ignition Map Cell",
		"Scale Entire Map (multiply)",
		"Exit",
	}

	selectedOption, _ := pterm.DefaultInteractiveSelect.
		WithOptions(options).
		Show("Select what to edit:")

	switch selectedOption {
	case "Edit Rev Limiter":
		editRevLimiter(filename, dryRun)
	case "Edit Fuel Map Cell":
		editMapCell(filename, 0x6700, "Fuel Map", 8, 16, 0.04, 0, dryRun)
	case "Edit Ignition Map Cell":
		editMapCell(filename, 0x6780, "Ignition Map", 8, 16, 0.75, -24.0, dryRun)
	case "Scale Entire Map (multiply)":
		scaleMap(filename, dryRun)
	case "Exit":
		pterm.Info.Println("Exiting edit mode.")
		return
	}
}

func editRevLimiter(filename string, dryRun bool) {
	pterm.Info.Println("M2.1 Rev Limiter location varies by application")
	pterm.Warning.Println("Setting too high can cause catastrophic engine damage!")
	pterm.Warning.Println("M2.1 typically has hardware-based limiters in addition to software")

	currentValue, _ := pterm.DefaultInteractiveTextInput.Show("Enter new RPM limit (e.g., 6500)")

	rpm := 0
	fmt.Sscanf(currentValue, "%d", &rpm)

	if rpm < 3000 || rpm > 7500 {
		pterm.Error.Println("Invalid RPM range for M2.1. Must be between 3000-7500.")
		return
	}

	pterm.Info.Printf("Will set rev limiter to: %d RPM\n", rpm)
	pterm.Warning.Println("Note: M2.1 may also have hardware limiters that override software settings")

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
	pterm.Info.Println("Verify with datalogger before driving!")
}

func editMapCell(filename string, offset int64, mapName string, rows, cols int, scale, offset2 float64, dryRun bool) {
	pterm.Info.Printf("Editing %s (%dx%d)\n", mapName, rows, cols)

	rowStr, _ := pterm.DefaultInteractiveTextInput.Show("Enter row (0-" + fmt.Sprintf("%d", rows-1) + ")")
	colStr, _ := pterm.DefaultInteractiveTextInput.Show("Enter column (0-" + fmt.Sprintf("%d", cols-1) + ")")

	row, col := 0, 0
	fmt.Sscanf(rowStr, "%d", &row)
	fmt.Sscanf(colStr, "%d", &col)

	if row < 0 || row >= rows || col < 0 || col >= cols {
		pterm.Error.Println("Invalid cell coordinates")
		return
	}

	f, err := os.Open(filename)
	if err != nil {
		pterm.Error.Printf("Error opening file: %v\n", err)
		return
	}
	cellOffset := offset + int64(row*cols+col)
	f.Seek(cellOffset, io.SeekStart)
	var currentRaw uint8
	binary.Read(f, binary.LittleEndian, &currentRaw)
	f.Close()

	currentValue := float64(currentRaw)*scale + offset2
	pterm.Info.Printf("Current value at [%d,%d]: %.2f (raw: 0x%02X)\n", row, col, currentValue, currentRaw)

	newValueStr, _ := pterm.DefaultInteractiveTextInput.Show("Enter new value")
	newValue := 0.0
	fmt.Sscanf(newValueStr, "%f", &newValue)

	newRaw := uint8((newValue - offset2) / scale)
	pterm.Info.Printf("New value: %.2f (raw: 0x%02X)\n", newValue, newRaw)

	if dryRun {
		pterm.Warning.Println("DRY RUN - No changes made")
		return
	}

	result, _ := pterm.DefaultInteractiveConfirm.Show("Write this change?")
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
	data[cellOffset] = newRaw
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		pterm.Error.Printf("Failed to write: %v\n", err)
		return
	}

	pterm.Success.Println("Cell updated successfully!")
}

func scaleMap(filename string, dryRun bool) {
	pterm.Info.Println("Scale an entire map by a multiplier")
	pterm.Warning.Println("This modifies ALL cells in the selected map!")

	options := []string{
		"Main Fuel Map (0x6700)",
		"Ignition Map (0x6780)",
		"Cancel",
	}

	selectedOption, _ := pterm.DefaultInteractiveSelect.
		WithOptions(options).
		Show("Select map to scale:")

	if selectedOption == "Cancel" {
		return
	}

	multiplierStr, _ := pterm.DefaultInteractiveTextInput.Show("Enter multiplier (e.g., 1.1 for +10%, 0.9 for -10%)")
	multiplier := 1.0
	fmt.Sscanf(multiplierStr, "%f", &multiplier)

	if multiplier < 0.5 || multiplier > 2.0 {
		pterm.Error.Println("Multiplier out of safe range (0.5-2.0)")
		return
	}

	var offset int64
	var rows, cols int
	if strings.Contains(selectedOption, "Fuel") {
		offset = 0x6700
		rows, cols = 8, 16
	} else {
		offset = 0x6780
		rows, cols = 8, 16
	}

	pterm.Info.Printf("Will multiply all values in %s by %.2f\n", selectedOption, multiplier)

	if dryRun {
		pterm.Warning.Println("DRY RUN - No changes made")
		return
	}

	result, _ := pterm.DefaultInteractiveConfirm.Show("Apply this scaling?")
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
	for i := 0; i < rows*cols; i++ {
		cellOffset := int(offset) + i
		oldVal := data[cellOffset]
		newVal := uint8(float64(oldVal) * multiplier)
		data[cellOffset] = newVal
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		pterm.Error.Printf("Failed to write: %v\n", err)
		return
	}

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
		applyRevLimitPreset(filename, dryRun)
	case "boost":
		pterm.Error.Println("Boost preset not yet implemented")
	case "fuel-enrich":
		applyFuelEnrichPreset(filename, dryRun)
	default:
		pterm.Error.Printf("Unknown preset: %s\n", presetName)
		pterm.Info.Println("Available presets: revlimit, fuel-enrich")
	}
}

func applyRevLimitPreset(filename string, dryRun bool) {
	pterm.Info.Println("Rev Limit Preset: Raises limiter by 500 RPM")

	if dryRun {
		pterm.Warning.Println("DRY RUN - Would increase rev limit by 500 RPM")
		return
	}

	result, _ := pterm.DefaultInteractiveConfirm.Show("Apply +500 RPM to rev limiter?")
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
	data[0x7000] += 10

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		pterm.Error.Printf("Failed to write: %v\n", err)
		return
	}

	pterm.Success.Println("Preset applied successfully!")
}

func applyFuelEnrichPreset(filename string, dryRun bool) {
	pterm.Info.Println("Fuel Enrichment Preset: +5% across entire fuel map")
	pterm.Warning.Println("This can cause rich running conditions!")

	if dryRun {
		pterm.Warning.Println("DRY RUN - Would enrich fuel map by 5%")
		return
	}

	result, _ := pterm.DefaultInteractiveConfirm.Show("Apply +5% fuel enrichment?")
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
	offset := 0x6700
	rows, cols := 8, 16

	for i := 0; i < rows*cols; i++ {
		cellOffset := offset + i
		oldVal := data[cellOffset]
		newVal := uint8(float64(oldVal) * 1.05)
		data[cellOffset] = newVal
	}

	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		pterm.Error.Printf("Failed to write: %v\n", err)
		return
	}

	pterm.Success.Println("Fuel enrichment applied!")
}
