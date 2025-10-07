package export

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pterm/pterm"
	"github.com/tosih/motronic-m21-tool/pkg/models"
)

// ExportMapsToCSV exports selected maps to CSV files
func ExportMapsToCSV(filename, exportPath, mapType string, readMap func(string, models.MapConfig) (*models.ECUMap, error)) {
	// Create export directory if it doesn't exist
	if err := os.MkdirAll(exportPath, 0755); err != nil {
		pterm.Error.Printf("Failed to create export directory: %v\n", err)
		return
	}

	var selectedConfigs []models.MapConfig
	if mapType == "all" {
		selectedConfigs = models.MapConfigs
	} else {
		// Select specific map
		for _, cfg := range models.MapConfigs {
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

func exportMapToCSV(m *models.ECUMap, filename string) error {
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

// ImportMapFromCSV imports a map from a CSV file
func ImportMapFromCSV(ecuFilename, csvFilename string) {
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
