package gui

import (
	"fmt"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/tosih/motronic-m21-tool/pkg/editor"
	"github.com/tosih/motronic-m21-tool/pkg/models"
	"github.com/tosih/motronic-m21-tool/pkg/reader"
)

// buildConfigView creates the configuration parameters tab
func (mw *MainWindow) buildConfigView() *gtk.Box {
	box := gtk.NewBox(gtk.OrientationVertical, 10)
	box.SetMarginStart(20)
	box.SetMarginEnd(20)
	box.SetMarginTop(20)
	box.SetMarginBottom(20)

	// Header
	headerLabel := gtk.NewLabel("ECU Configuration Parameters")
	headerLabel.AddCSSClass("config-header")
	headerLabel.SetXAlign(0)
	box.Append(headerLabel)

	// Scrolled window for parameter list
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetVExpand(true)
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)

	// List box for parameters
	listBox := gtk.NewListBox()
	listBox.SetSelectionMode(gtk.SelectionNone)
	scrolled.SetChild(listBox)

	// Populate parameters
	for _, param := range models.ConfigParams {
		row := mw.createConfigParamRow(param)
		listBox.Append(row)
	}

	box.Append(scrolled)

	return box
}

// createConfigParamRow creates a row for a single config parameter
func (mw *MainWindow) createConfigParamRow(param models.ConfigParam) *gtk.Box {
	rowBox := gtk.NewBox(gtk.OrientationHorizontal, 15)
	rowBox.SetMarginStart(10)
	rowBox.SetMarginEnd(10)
	rowBox.SetMarginTop(8)
	rowBox.SetMarginBottom(8)

	// Left side - parameter info
	infoBox := gtk.NewBox(gtk.OrientationVertical, 3)
	infoBox.SetHExpand(true)

	nameLabel := gtk.NewLabel(param.Name)
	nameLabel.SetXAlign(0)
	nameLabel.AddCSSClass("param-name")
	infoBox.Append(nameLabel)

	descLabel := gtk.NewLabel(param.Description)
	descLabel.SetXAlign(0)
	descLabel.AddCSSClass("param-description")
	descLabel.SetWrap(true)
	infoBox.Append(descLabel)

	rowBox.Append(infoBox)

	// Middle - current value
	valueLabel := gtk.NewLabel("--")
	valueLabel.AddCSSClass("param-value")
	valueLabel.SetSizeRequest(120, -1)
	valueLabel.SetXAlign(1) // Right align the value
	rowBox.Append(valueLabel)

	// Store the label for later updates
	mw.configValueLabels[param.Name] = valueLabel

	// Right side - edit button
	editButton := gtk.NewButtonWithLabel("Edit")
	editButton.ConnectClicked(func() {
		mw.editConfigParam(param, valueLabel)
	})
	rowBox.Append(editButton)

	return rowBox
}

// refreshConfigValues refreshes all config parameter values from the file
func (mw *MainWindow) refreshConfigValues() {
	if mw.currentFile == "" {
		return
	}

	// Read all config values
	for _, param := range models.ConfigParams {
		value, err := reader.ReadConfigParam(mw.currentFile, param)
		if err != nil {
			// Show error in the label
			if label, ok := mw.configValueLabels[param.Name]; ok {
				label.SetText("Error")
			}
			continue
		}

		// Update the value label
		if label, ok := mw.configValueLabels[param.Name]; ok {
			label.SetText(fmt.Sprintf("%.1f %s", value, param.Unit))
		}
	}
}

// findChildByName recursively finds a widget by name (disabled for now)
func (mw *MainWindow) findChildByName(widget gtk.Widgetter, name string) gtk.Widgetter {
	// Note: This function needs to be reimplemented using a different approach
	// as NextSibling() is not available in this version of gotk4
	// For now, we'll manually track value labels in the refresh function
	return nil
}

// editConfigParam shows a dialog to edit a config parameter
func (mw *MainWindow) editConfigParam(param models.ConfigParam, valueLabel *gtk.Label) {
	if mw.currentFile == "" {
		mw.showErrorDialog("Please open an ECU file first")
		return
	}

	// Read current value
	currentValue, err := reader.ReadConfigParam(mw.currentFile, param)
	if err != nil {
		mw.showErrorDialog(fmt.Sprintf("Failed to read parameter: %v", err))
		return
	}

	dialog := gtk.NewDialog()
	dialog.SetTransientFor(&mw.window.Window)
	dialog.SetModal(true)
	dialog.SetTitle(fmt.Sprintf("Edit %s", param.Name))
	dialog.SetDefaultSize(450, 250)

	// Content area
	contentArea := dialog.ContentArea()
	contentArea.SetSpacing(12)
	contentArea.SetMarginStart(20)
	contentArea.SetMarginEnd(20)
	contentArea.SetMarginTop(20)
	contentArea.SetMarginBottom(20)

	// Parameter info
	infoLabel := gtk.NewLabel(param.Description)
	infoLabel.SetWrap(true)
	infoLabel.SetXAlign(0)
	contentArea.Append(infoLabel)

	// Current value
	currentLabel := gtk.NewLabel(fmt.Sprintf("Current Value: %.1f %s", currentValue, param.Unit))
	currentLabel.SetXAlign(0)
	currentLabel.AddCSSClass("current-value")
	contentArea.Append(currentLabel)

	// Valid range
	rangeLabel := gtk.NewLabel(fmt.Sprintf("Valid Range: %.1f - %.1f %s", param.MinValue, param.MaxValue, param.Unit))
	rangeLabel.SetXAlign(0)
	contentArea.Append(rangeLabel)

	// Warning
	warningLabel := gtk.NewLabel("⚠️  Modifying ECU parameters can damage your engine!")
	warningLabel.AddCSSClass("warning-text")
	warningLabel.SetXAlign(0)
	contentArea.Append(warningLabel)

	// Entry box
	entryBox := gtk.NewBox(gtk.OrientationHorizontal, 10)
	entryLabel := gtk.NewLabel("New Value:")
	entryBox.Append(entryLabel)

	entry := gtk.NewEntry()
	entry.SetText(fmt.Sprintf("%.1f", currentValue))
	entry.SetHExpand(true)
	entryBox.Append(entry)

	unitLabel := gtk.NewLabel(param.Unit)
	entryBox.Append(unitLabel)
	contentArea.Append(entryBox)

	// Buttons
	dialog.AddButton("Cancel", int(gtk.ResponseCancel))
	dialog.AddButton("Save", int(gtk.ResponseAccept))

	dialog.ConnectResponse(func(responseID int) {
		if responseID == int(gtk.ResponseAccept) {
			var newValue float64
			if _, err := fmt.Sscanf(entry.Text(), "%f", &newValue); err != nil {
				mw.showErrorDialog(fmt.Sprintf("Invalid value: %v", err))
				dialog.Destroy()
				return
			}

			// Validate range
			if newValue < param.MinValue || newValue > param.MaxValue {
				mw.showErrorDialog(fmt.Sprintf("Value out of range! Must be between %.1f and %.1f", param.MinValue, param.MaxValue))
				dialog.Destroy()
				return
			}

			// Confirm and save
			mw.confirmAndSaveConfigParam(param, newValue, valueLabel, dialog)
		} else {
			dialog.Destroy()
		}
	})

	dialog.Show()
}

// confirmAndSaveConfigParam shows confirmation and saves config parameter
func (mw *MainWindow) confirmAndSaveConfigParam(param models.ConfigParam, newValue float64, valueLabel *gtk.Label, editDialog *gtk.Dialog) {
	confirmDialog := gtk.NewMessageDialog(
		&mw.window.Window,
		gtk.DialogModal,
		gtk.MessageWarning,
		gtk.ButtonsNone,
	)

	confirmDialog.SetMarkup(fmt.Sprintf("<b>Confirm ECU Modification</b>\n\nThis will modify the ECU binary file.\nA backup will be created automatically.\n\nParameter: %s\nNew Value: %.1f %s\n\nProceed with caution!",
		param.Name, newValue, param.Unit))

	confirmDialog.AddButton("Cancel", int(gtk.ResponseCancel))
	confirmDialog.AddButton("Save Changes", int(gtk.ResponseAccept))

	confirmDialog.ConnectResponse(func(responseID int) {
		if responseID == int(gtk.ResponseAccept) {
			mw.saveConfigParam(param, newValue, valueLabel)
			editDialog.Destroy()
		}
		confirmDialog.Destroy()
	})

	confirmDialog.Show()
}

// saveConfigParam saves a config parameter to the ECU file
func (mw *MainWindow) saveConfigParam(param models.ConfigParam, newValue float64, valueLabel *gtk.Label) {
	// Create backup
	_, err := editor.CreateBackup(mw.currentFile)
	if err != nil {
		mw.showErrorDialog(fmt.Sprintf("Failed to create backup: %v", err))
		return
	}

	// Write new value
	err = editor.WriteConfigParam(mw.currentFile, param, newValue)
	if err != nil {
		mw.showErrorDialog(fmt.Sprintf("Failed to save parameter: %v", err))
		return
	}

	// Update UI - read the actual value back from file to confirm
	actualValue, err := reader.ReadConfigParam(mw.currentFile, param)
	if err == nil {
		valueLabel.SetText(fmt.Sprintf("%.1f %s", actualValue, param.Unit))
	} else {
		valueLabel.SetText(fmt.Sprintf("%.1f %s", newValue, param.Unit))
	}

	mw.statusBar.SetText(fmt.Sprintf("%s updated to %.1f %s", param.Name, newValue, param.Unit))

	mw.showInfoDialog("Parameter saved successfully! Backup created.")
}
