package editor

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/tosih/motronic-m21-tool/pkg/models"
)

// CreateBackup creates a timestamped backup of the file
func CreateBackup(filename string) (string, error) {
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

// InteractiveEdit provides an interactive menu for editing ECU maps
func InteractiveEdit(filename string, dryRun bool) {
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
		EditRevLimiter(filename, dryRun)
	case "Edit Fuel Map Cell":
		EditMapCell(filename, models.MapConfigs[0])
	case "Edit Ignition Map Cell":
		EditMapCell(filename, models.MapConfigs[1])
	case "Scale Entire Map":
		ScaleMap(filename, dryRun)
	case "Exit":
		pterm.Info.Println("Exiting edit mode.")
		return
	}
}

// EditRevLimiter allows editing the rev limiter value
func EditRevLimiter(filename string, dryRun bool) {
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

	backup, err := CreateBackup(filename)
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

// EditMapCell allows editing a specific cell in a map (CLI version)
func EditMapCell(filename string, cfg models.MapConfig) {
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

	backup, _ := CreateBackup(filename)
	pterm.Success.Printf("Backup created: %s\n", backup)

	data, _ := os.ReadFile(filename)
	data[cellOffset] = newRaw
	os.WriteFile(filename, data, 0644)

	pterm.Success.Println("Cell updated successfully!")
}

// ScaleMap scales an entire map by a multiplier
func ScaleMap(filename string, dryRun bool) {
	pterm.Info.Println("Scale an entire map by a multiplier")
	pterm.Warning.Println("This modifies ALL cells in the selected map!")

	mapNames := []string{}
	for _, cfg := range models.MapConfigs {
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
	var selectedCfg models.MapConfig
	for _, cfg := range models.MapConfigs {
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

	backup, _ := CreateBackup(filename)
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

// ApplyPreset applies a predefined modification preset
func ApplyPreset(filename, presetName string, dryRun bool) {
	pterm.DefaultHeader.WithFullWidth().
		WithBackgroundStyle(pterm.NewStyle(pterm.BgYellow)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack)).
		Println("PRESET MODIFICATION MODE")

	pterm.Warning.Println("Presets apply predefined changes. USE WITH CAUTION!")

	switch presetName {
	case "revlimit":
		EditRevLimiter(filename, dryRun)
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

	backup, _ := CreateBackup(filename)
	pterm.Success.Printf("Backup created: %s\n", backup)

	cfg := models.MapConfigs[0] // Main fuel map
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

// WriteConfigParam writes a configuration parameter value to the ECU file
func WriteConfigParam(filename string, param models.ConfigParam, value float64) error {
	// Convert real value to raw
	var rawValue interface{}
	switch param.DataType {
	case "uint8":
		rawValue = uint8((value - param.Offset2) / param.Scale)
	case "uint16":
		rawValue = uint16((value - param.Offset2) / param.Scale)
	case "int8":
		rawValue = int8((value - param.Offset2) / param.Scale)
	case "int16":
		rawValue = int16((value - param.Offset2) / param.Scale)
	default:
		return fmt.Errorf("unsupported data type: %s", param.DataType)
	}

	// Read file
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	// Check bounds
	if int(param.Offset) >= len(data) {
		return fmt.Errorf("offset 0x%X out of bounds", param.Offset)
	}

	// Write value
	switch v := rawValue.(type) {
	case uint8:
		data[param.Offset] = v
	case uint16:
		binary.LittleEndian.PutUint16(data[param.Offset:], v)
	case int8:
		data[param.Offset] = byte(v)
	case int16:
		binary.LittleEndian.PutUint16(data[param.Offset:], uint16(v))
	}

	// Write back to file
	return os.WriteFile(filename, data, 0644)
}

// EditMapCellDirect edits a specific map cell without prompts (for GUI use)
func EditMapCellDirect(filename string, cfg models.MapConfig, row, col int, newValue float64) error {
	if row < 0 || row >= cfg.Rows || col < 0 || col >= cfg.Cols {
		return fmt.Errorf("invalid cell coordinates: [%d,%d]", row, col)
	}

	// Read file
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	// Calculate offset
	cellOffset := cfg.Offset + int64(row*cfg.Cols+col)
	if int(cellOffset) >= len(data) {
		return fmt.Errorf("cell offset out of bounds")
	}

	// Convert value to raw
	var newRaw uint8
	if cfg.DataType == "uint8" {
		newRaw = uint8((newValue - cfg.Offset2) / cfg.Scale)
		data[cellOffset] = newRaw
	} else if cfg.DataType == "uint16" {
		newRaw16 := uint16((newValue - cfg.Offset2) / cfg.Scale)
		binary.LittleEndian.PutUint16(data[cellOffset:], newRaw16)
	}

	// Write back
	return os.WriteFile(filename, data, 0644)
}

// ExportMapToCSV exports a map to a CSV file
func ExportMapToCSV(ecuMap *models.ECUMap, exportPath, mapName string) error {
	// This is a placeholder - implement CSV export logic
	// You can import the export package and call its functions
	return fmt.Errorf("CSV export not yet implemented in GUI")
}
