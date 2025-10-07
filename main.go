package main

import (
	"flag"
	"os"

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

	if *filename == "" && !*list {
		pterm.Error.Println("Please specify an ECU file with -file flag")
		flag.Usage()
		os.Exit(1)
	}

	// List available maps
	if *list {
		renderer.ListAvailableMaps()
		return
	}

	// Web interface mode
	if *webMode {
		var server *web.Server
		if *compareFile != "" {
			server = web.NewCompareServer(*filename, *compareFile, *port)
		} else {
			server = web.NewServer(*filename, *port)
		}
		if err := server.Start(); err != nil {
			pterm.Error.Printf("Web server error: %v\n", err)
			os.Exit(1)
		}
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
