package gui

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/tosih/motronic-m21-tool/pkg/editor"
)

// onMapClicked handles mouse clicks on the map for editing
func (mw *MainWindow) onMapClicked(gesture *gtk.GestureClick, nPress int, x, y float64) {
	if mw.currentMap == nil || mw.currentFile == "" {
		return
	}

	// Get widget dimensions
	width := mw.mapDrawArea.AllocatedWidth()
	height := mw.mapDrawArea.AllocatedHeight()

	// Determine which cell was clicked
	row, col, valid := mw.getCellAtPosition(x, y, width, height)
	if !valid {
		return
	}

	// Show edit dialog
	mw.showCellEditDialog(row, col)
}

// showCellEditDialog displays a dialog to edit a single cell value
func (mw *MainWindow) showCellEditDialog(row, col int) {
	currentValue := mw.currentMap.Data[row][col]

	dialog := gtk.NewDialog()
	dialog.SetTransientFor(&mw.window.Window)
	dialog.SetModal(true)
	dialog.SetTitle("Edit Cell Value")
	dialog.SetDefaultSize(400, 200)

	// Content area
	contentArea := dialog.ContentArea()
	contentArea.SetSpacing(10)
	contentArea.SetMarginStart(20)
	contentArea.SetMarginEnd(20)
	contentArea.SetMarginTop(20)
	contentArea.SetMarginBottom(20)

	// Info label
	infoLabel := gtk.NewLabel(fmt.Sprintf(
		"Map: %s\nPosition: Row %d, Column %d\nCurrent Value: %.2f %s",
		mw.currentMap.Config.Name,
		row, col,
		currentValue,
		mw.currentMap.Config.Unit,
	))
	infoLabel.SetXAlign(0)
	contentArea.Append(infoLabel)

	// Warning label
	warningLabel := gtk.NewLabel("⚠️  Modifying ECU values can damage your engine!")
	warningLabel.AddCSSClass("warning-text")
	warningLabel.SetXAlign(0)
	contentArea.Append(warningLabel)

	// Entry for new value
	entryBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	entryLabel := gtk.NewLabel("New Value:")
	entryLabel.SetXAlign(0)
	entryBox.Append(entryLabel)

	entry := gtk.NewEntry()
	entry.SetText(fmt.Sprintf("%.2f", currentValue))
	entry.SetHExpand(true)
	entryBox.Append(entry)

	unitLabel := gtk.NewLabel(mw.currentMap.Config.Unit)
	entryBox.Append(unitLabel)
	contentArea.Append(entryBox)

	// Buttons
	dialog.AddButton("Cancel", int(gtk.ResponseCancel))
	saveButton := dialog.AddButton("Save", int(gtk.ResponseAccept))
	saveButton.AddCSSClass("suggested-action")

	dialog.ConnectResponse(func(responseID int) {
		if responseID == int(gtk.ResponseAccept) {
			newValueStr := entry.Text()
			var newValue float64
			if _, err := fmt.Sscanf(newValueStr, "%f", &newValue); err != nil {
				mw.showErrorDialog(fmt.Sprintf("Invalid value: %v", err))
				dialog.Destroy()
				return
			}

			// Show confirmation dialog
			mw.confirmAndSaveEdit(row, col, newValue, dialog)
		} else {
			dialog.Destroy()
		}
	})

	dialog.Show()
}

// confirmAndSaveEdit shows a confirmation dialog before saving
func (mw *MainWindow) confirmAndSaveEdit(row, col int, newValue float64, editDialog *gtk.Dialog) {
	confirmDialog := gtk.NewMessageDialog(
		&mw.window.Window,
		gtk.DialogModal,
		gtk.MessageWarning,
		gtk.ButtonsNone,
		"Confirm ECU Modification",
	)

	confirmDialog.SetProperty("secondary-text",
		fmt.Sprintf("This will modify the ECU binary file.\nA backup will be created automatically.\n\nProceed with caution!"))

	confirmDialog.AddButton("Cancel", int(gtk.ResponseCancel))
	confirmButton := confirmDialog.AddButton("Save Changes", int(gtk.ResponseAccept))
	confirmButton.AddCSSClass("destructive-action")

	confirmDialog.ConnectResponse(func(responseID int) {
		if responseID == int(gtk.ResponseAccept) {
			mw.saveCellEdit(row, col, newValue)
			editDialog.Destroy()
		}
		confirmDialog.Destroy()
	})

	confirmDialog.Show()
}

// saveCellEdit saves a cell edit to the ECU file
func (mw *MainWindow) saveCellEdit(row, col int, newValue float64) {
	// Create backup first
	if err := editor.CreateBackup(mw.currentFile); err != nil {
		mw.showErrorDialog(fmt.Sprintf("Failed to create backup: %v", err))
		return
	}

	// Update the cell
	err := editor.EditMapCellDirect(mw.currentFile, mw.currentMap.Config, row, col, newValue)
	if err != nil {
		mw.showErrorDialog(fmt.Sprintf("Failed to save edit: %v", err))
		return
	}

	// Update local data
	mw.currentMap.Data[row][col] = newValue

	// Redraw
	mw.mapDrawArea.QueueDraw()

	// Update status
	mw.statusBar.SetText(fmt.Sprintf("Cell [%d,%d] updated to %.2f %s", row, col, newValue, mw.currentMap.Config.Unit))

	// Show success message
	mw.showInfoDialog("Edit saved successfully! Backup created.")
}

// showInfoDialog displays an informational message
func (mw *MainWindow) showInfoDialog(message string) {
	dialog := gtk.NewMessageDialog(
		&mw.window.Window,
		gtk.DialogModal,
		gtk.MessageInfo,
		gtk.ButtonsOK,
		"%s",
		message,
	)
	dialog.ConnectResponse(func(response int) {
		dialog.Destroy()
	})
	dialog.Show()
}

// openCompareDialog opens a dialog to select a second file for comparison
func (mw *MainWindow) openCompareDialog() {
	if mw.currentFile == "" {
		mw.showErrorDialog("Please open an ECU file first")
		return
	}

	dialog := gtk.NewFileChooserDialog(
		"Select ECU File to Compare",
		&mw.window.Window,
		gtk.FileChooserActionOpen,
	)

	dialog.AddButton("Cancel", int(gtk.ResponseCancel))
	dialog.AddButton("Compare", int(gtk.ResponseAccept))
	dialog.SetModal(true)

	// Add file filter
	filter := gtk.NewFileFilter()
	filter.SetName("ECU Binary Files (*.bin)")
	filter.AddPattern("*.bin")
	filter.AddPattern("*.BIN")
	dialog.AddFilter(filter)

	dialog.ConnectResponse(func(responseID int) {
		if responseID == int(gtk.ResponseAccept) {
			file := dialog.File()
			if file != nil {
				path := file.Path()
				mw.compareFile = path
				mw.loadCurrentMap() // Reload to load comparison map
				mw.statusBar.SetText(fmt.Sprintf("Comparing with: %s", path))
			}
		}
		dialog.Destroy()
	})

	dialog.Show()
}

// exportDialog shows a dialog for exporting maps to CSV
func (mw *MainWindow) exportDialog() {
	if mw.currentFile == "" {
		mw.showErrorDialog("Please open an ECU file first")
		return
	}

	dialog := gtk.NewFileChooserDialog(
		"Select Export Directory",
		&mw.window.Window,
		gtk.FileChooserActionSelectFolder,
	)

	dialog.AddButton("Cancel", int(gtk.ResponseCancel))
	dialog.AddButton("Export", int(gtk.ResponseAccept))
	dialog.SetModal(true)

	dialog.ConnectResponse(func(responseID int) {
		if responseID == int(gtk.ResponseAccept) {
			file := dialog.File()
			if file != nil {
				exportPath := file.Path()
				mw.performExport(exportPath)
			}
		}
		dialog.Destroy()
	})

	dialog.Show()
}

// performExport exports the current map to CSV
func (mw *MainWindow) performExport(exportPath string) {
	// For now, export just the current map
	// You can extend this to export all maps
	err := editor.ExportMapToCSV(mw.currentMap, exportPath, mw.currentMap.Config.Name)
	if err != nil {
		mw.showErrorDialog(fmt.Sprintf("Export failed: %v", err))
		return
	}

	mw.showInfoDialog(fmt.Sprintf("Map exported successfully to:\n%s", exportPath))
}
