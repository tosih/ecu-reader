package gui

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/tosih/motronic-m21-tool/pkg/scanner"
)

// buildScannerView creates the scanner tab
func (mw *MainWindow) buildScannerView() *gtk.Box {
	box := gtk.NewBox(gtk.OrientationVertical, 10)
	box.SetMarginStart(20)
	box.SetMarginEnd(20)
	box.SetMarginTop(20)
	box.SetMarginBottom(20)

	// Header
	headerLabel := gtk.NewLabel("Binary Scanner - Find Unknown Maps")
	headerLabel.AddCSSClass("scanner-header")
	headerLabel.SetXAlign(0)
	box.Append(headerLabel)

	// Description
	descLabel := gtk.NewLabel("Scan the ECU binary file for potential map locations based on data patterns.")
	descLabel.SetXAlign(0)
	descLabel.SetWrap(true)
	box.Append(descLabel)

	// Scan parameters
	paramsBox := gtk.NewBox(gtk.OrientationHorizontal, 15)

	// Min variance
	minVarBox := gtk.NewBox(gtk.OrientationHorizontal, 5)
	minVarLabel := gtk.NewLabel("Min Variance:")
	minVarBox.Append(minVarLabel)

	minVarEntry := gtk.NewEntry()
	minVarEntry.SetText("10")
	minVarEntry.SetSizeRequest(80, -1)
	minVarEntry.SetName("min_variance")
	minVarBox.Append(minVarEntry)
	paramsBox.Append(minVarBox)

	// Dimension filter
	dimBox := gtk.NewBox(gtk.OrientationHorizontal, 5)
	dimLabel := gtk.NewLabel("Dimensions:")
	dimBox.Append(dimLabel)

	dimCombo := gtk.NewComboBoxText()
	dimCombo.AppendText("All (8x8, 8x16, 16x16)")
	dimCombo.AppendText("8x8 only")
	dimCombo.AppendText("8x16 only")
	dimCombo.AppendText("16x16 only")
	dimCombo.SetActive(0)
	dimCombo.SetName("dimension_filter")
	dimBox.Append(dimCombo)
	paramsBox.Append(dimBox)

	box.Append(paramsBox)

	// Scan button
	scanButton := gtk.NewButtonWithLabel("Scan File")
	scanButton.AddCSSClass("suggested-action")
	scanButton.ConnectClicked(func() {
		mw.performScan(box, minVarEntry, dimCombo)
	})
	box.Append(scanButton)

	// Results area (initially empty)
	resultsLabel := gtk.NewLabel("")
	resultsLabel.SetName("scan_results")
	resultsLabel.SetXAlign(0)
	resultsLabel.SetYAlign(0)
	resultsLabel.SetSelectable(true)

	resultsScrolled := gtk.NewScrolledWindow()
	resultsScrolled.SetVExpand(true)
	resultsScrolled.SetPolicy(gtk.PolicyAutomatic, gtk.PolicyAutomatic)
	resultsScrolled.SetChild(resultsLabel)
	box.Append(resultsScrolled)

	return box
}

// performScan executes the binary scan
func (mw *MainWindow) performScan(containerBox *gtk.Box, minVarEntry *gtk.Entry, dimCombo *gtk.ComboBoxText) {
	if mw.currentFile == "" {
		mw.showErrorDialog("Please open an ECU file first")
		return
	}

	// Parse min variance
	minVarStr := minVarEntry.Text()
	var minVariance float64
	if _, err := fmt.Sscanf(minVarStr, "%f", &minVariance); err != nil {
		minVariance = 10.0
	}

	// Get dimension filter
	dimText := dimCombo.ActiveText()

	mw.statusBar.SetText("Scanning file... This may take a moment.")

	// Perform scan
	results := scanner.ScanFile(mw.currentFile, minVariance)

	// Filter by dimensions if needed
	filteredResults := []scanner.ScanResult{}
	for _, result := range results {
		include := true

		switch dimText {
		case "8x8 only":
			include = (result.Rows == 8 && result.Cols == 8)
		case "8x16 only":
			include = (result.Rows == 8 && result.Cols == 16)
		case "16x16 only":
			include = (result.Rows == 16 && result.Cols == 16)
		}

		if include {
			filteredResults = append(filteredResults, result)
		}
	}

	// Display results
	mw.displayScanResults(containerBox, filteredResults)

	mw.statusBar.SetText(fmt.Sprintf("Scan complete. Found %d potential maps.", len(filteredResults)))
}

// displayScanResults shows scan results in the UI
func (mw *MainWindow) displayScanResults(containerBox *gtk.Box, results []scanner.ScanResult) {
	// Find the results label
	resultsLabel := mw.findChildByName(containerBox, "scan_results")
	if resultsLabel == nil {
		return
	}

	label, ok := resultsLabel.(*gtk.Label)
	if !ok {
		return
	}

	if len(results) == 0 {
		label.SetText("No potential maps found with the current criteria.")
		return
	}

	// Build results text
	resultsText := fmt.Sprintf("Found %d potential maps:\n\n", len(results))

	for i, result := range results {
		resultsText += fmt.Sprintf("%d. Offset: 0x%04X (%dx%d)\n", i+1, result.Offset, result.Rows, result.Cols)
		resultsText += fmt.Sprintf("   Min: %.2f, Max: %.2f, Variance: %.1f\n", result.Min, result.Max, result.Variance)
		resultsText += fmt.Sprintf("   Mean: %.2f, StdDev: %.2f\n\n", result.Mean, result.StdDev)
	}

	label.SetText(resultsText)
}
