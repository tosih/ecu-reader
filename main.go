package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"

	"github.com/pterm/pterm"
	"github.com/tosih/motronic-m21-tool/pkg/compare"
	"github.com/tosih/motronic-m21-tool/pkg/editor"
	"github.com/tosih/motronic-m21-tool/pkg/export"
	"github.com/tosih/motronic-m21-tool/pkg/reader"
	"github.com/tosih/motronic-m21-tool/pkg/renderer"
	"github.com/tosih/motronic-m21-tool/pkg/scanner"
	"github.com/tosih/motronic-m21-tool/pkg/web"
)

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
	compareFile := flag.String("compare", "", "Compare current file with another ECU file")
	list := flag.Bool("list", false, "List all available maps")
	webMode := flag.Bool("web", false, "Launch web interface for interactive visualization")
	port := flag.Int("port", 8080, "Port for web server (default: 8080)")

	flag.Parse()

	// List available maps
	if *list {
		renderer.ListAvailableMaps()
		return
	}

	// Web interface mode
	if *webMode {
		var server *web.Server
		fileOrDir := *filename

		// If no file specified, use bins/ directory
		if fileOrDir == "" {
			fileOrDir = "bins"
		}

		if *compareFile != "" {
			server = web.NewCompareServer(fileOrDir, *compareFile, *port)
		} else {
			server = web.NewServer(fileOrDir, *port)
		}
		if err := server.Start(); err != nil {
			pterm.Error.Printf("Web server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// If no file specified, scan bins/ directory and list available files
	if *filename == "" {
		binFiles := findBinFiles("bins")
		if len(binFiles) == 0 {
			pterm.Error.Println("No .bin files found in bins/ directory")
			pterm.Info.Println("Please specify a file with -file flag or place .bin files in the bins/ directory")
			os.Exit(1)
		}

		// Show available bin files
		pterm.DefaultHeader.WithFullWidth().Println("Available ECU Binary Files")
		pterm.DefaultTable.WithHasHeader().WithData(pterm.TableData{
			{"#", "Filename", "Size", "Path"},
		}).Render()

		tableData := pterm.TableData{{"#", "Filename", "Size", "Path"}}
		for i, file := range binFiles {
			info, _ := os.Stat(file)
			size := formatFileSize(info.Size())
			tableData = append(tableData, []string{
				pterm.Sprintf("%d", i+1),
				filepath.Base(file),
				size,
				file,
			})
		}
		pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()

		pterm.Info.Printf("\nFound %d .bin file(s) in bins/ directory\n", len(binFiles))
		pterm.Info.Println("Use -file <path> to analyze a specific file")
		pterm.Info.Println("Use -web to launch web interface with all files")
		return
	}

	// Export maps to CSV
	if *exportPath != "" {
		export.ExportMapsToCSV(*filename, *exportPath, *mapType, reader.ReadMap)
		return
	}

	// Import map from CSV
	if *importFile != "" {
		export.ImportMapFromCSV(*filename, *importFile)
		return
	}

	// Compare two files
	if *compareFile != "" {
		compare.CompareFiles(*filename, *compareFile, *mapType, reader.ReadMap)
		return
	}

	// File scanning mode
	if *scan {
		scanner.ScanForMaps(*filename)
		return
	}

	// Interactive edit mode
	if *edit {
		editor.InteractiveEdit(*filename, false)
		return
	}

	// Apply preset modifications
	if *preset != "" {
		editor.ApplyPreset(*filename, *preset, false)
		return
	}

	// Normal display mode
	renderer.DisplayMaps(*filename, *mapType, *verbose, *displayMode, reader.ReadMap)
}

// findBinFiles scans a directory for .bin files
func findBinFiles(dir string) []string {
	var binFiles []string

	files, err := os.ReadDir(dir)
	if err != nil {
		return binFiles
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".bin") {
			binFiles = append(binFiles, filepath.Join(dir, file.Name()))
		}
	}

	return binFiles
}

// formatFileSize formats a file size in bytes to a human-readable string
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return pterm.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return pterm.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
