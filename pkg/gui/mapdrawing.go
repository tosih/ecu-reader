package gui

import (
	"fmt"
	"math"

	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// isDarkMode checks if the current theme is dark
func (mw *MainWindow) isDarkMode() bool {
	settings := gtk.SettingsGetDefault()
	return settings.ObjectProperty("gtk-application-prefer-dark-theme").(bool)
}

// getThemeColors returns text and background colors for the current theme
func (mw *MainWindow) getThemeColors() (textR, textG, textB, bgR, bgG, bgB float64) {
	isDark := mw.isDarkMode()
	if isDark {
		// Dark mode: light text on dark background
		return 0.9, 0.9, 0.9, 0.2, 0.2, 0.2
	}
	// Light mode: dark text on light background
	return 0.1, 0.1, 0.1, 0.95, 0.95, 0.95
}

// drawMapFunc is the drawing callback for the map visualization
func (mw *MainWindow) drawMapFunc(area *gtk.DrawingArea, cr *cairo.Context, width, height int) {
	if mw.currentMap == nil {
		mw.drawEmptyState(cr, width, height)
		return
	}

	// Get theme colors
	textR, textG, textB, bgR, bgG, bgB := mw.getThemeColors()

	// Fill background
	cr.SetSourceRGB(bgR, bgG, bgB)
	cr.Paint()

	// Calculate cell dimensions
	rows := mw.currentMap.Config.Rows
	cols := mw.currentMap.Config.Cols

	marginLeft := 80.0
	marginRight := 100.0
	marginTop := 60.0
	marginBottom := 80.0

	availableWidth := float64(width) - marginLeft - marginRight
	availableHeight := float64(height) - marginTop - marginBottom

	cellWidth := availableWidth / float64(cols)
	cellHeight := availableHeight / float64(rows)

	// Draw title
	cr.SetSourceRGB(textR, textG, textB)
	cr.SelectFontFace("Sans", cairo.FontSlantNormal, cairo.FontWeightBold)
	cr.SetFontSize(16)
	cr.MoveTo(marginLeft, 30)
	cr.ShowText(mw.currentMap.Config.Name)

	// Draw unit
	cr.SetFontSize(12)
	cr.MoveTo(marginLeft, 48)
	cr.ShowText(fmt.Sprintf("Unit: %s", mw.currentMap.Config.Unit))

	// Find min/max for color scaling
	minVal, maxVal := mw.findMinMax(mw.currentMap.Data)

	// Draw cells
	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			x := marginLeft + float64(col)*cellWidth
			y := marginTop + float64(row)*cellHeight

			value := mw.currentMap.Data[row][col]

			// Determine color based on value (heatmap)
			r, g, b := mw.valueToColor(value, minVal, maxVal)

			// Fill cell
			cr.Rectangle(x, y, cellWidth, cellHeight)
			cr.SetSourceRGB(r, g, b)
			cr.Fill()

			// Draw cell border (darker in light mode, lighter in dark mode)
			cr.Rectangle(x, y, cellWidth, cellHeight)
			if mw.isDarkMode() {
				cr.SetSourceRGB(0.5, 0.5, 0.5)
			} else {
				cr.SetSourceRGB(0.3, 0.3, 0.3)
			}
			cr.SetLineWidth(1)
			cr.Stroke()

			// Draw value text
			cr.SelectFontFace("Sans", cairo.FontSlantNormal, cairo.FontWeightNormal)
			cr.SetFontSize(10)

			text := fmt.Sprintf("%.2f", value)
			extents := cr.TextExtents(text)
			textX := x + (cellWidth-extents.Width)/2
			textY := y + (cellHeight+extents.Height)/2

			// White text for dark backgrounds, black for light
			luminance := 0.299*r + 0.587*g + 0.114*b
			if luminance < 0.5 {
				cr.SetSourceRGB(1, 1, 1)
			} else {
				cr.SetSourceRGB(0, 0, 0)
			}

			cr.MoveTo(textX, textY)
			cr.ShowText(text)
		}
	}

	// Draw RPM axis (horizontal)
	cr.SetSourceRGB(textR, textG, textB)
	cr.SelectFontFace("Sans", cairo.FontSlantNormal, cairo.FontWeightBold)
	cr.SetFontSize(11)

	for col := 0; col <= cols; col++ {
		x := marginLeft + float64(col)*cellWidth
		rpm := int(float64(col) / float64(cols) * 8000)

		text := fmt.Sprintf("%d", rpm)
		extents := cr.TextExtents(text)
		cr.MoveTo(x-extents.Width/2, marginTop+availableHeight+20)
		cr.ShowText(text)

		// Draw tick mark
		cr.MoveTo(x, marginTop+availableHeight)
		cr.LineTo(x, marginTop+availableHeight+5)
		cr.Stroke()
	}

	// RPM label
	cr.SetFontSize(12)
	text := "RPM"
	extents := cr.TextExtents(text)
	cr.MoveTo(marginLeft+availableWidth/2-extents.Width/2, float64(height)-20)
	cr.ShowText(text)

	// Draw Load axis (vertical)
	for row := 0; row <= rows; row++ {
		y := marginTop + float64(row)*cellHeight
		load := int(100 - float64(row)/float64(rows)*100)

		text := fmt.Sprintf("%d%%", load)
		extents := cr.TextExtents(text)
		cr.MoveTo(marginLeft-extents.Width-10, y+extents.Height/2)
		cr.ShowText(text)

		// Draw tick mark
		cr.MoveTo(marginLeft-5, y)
		cr.LineTo(marginLeft, y)
		cr.Stroke()
	}

	// Load label (rotated)
	cr.Save()
	cr.Translate(20, marginTop+availableHeight/2)
	cr.Rotate(-math.Pi / 2)
	text = "Load"
	extents = cr.TextExtents(text)
	cr.MoveTo(-extents.Width/2, 0)
	cr.ShowText(text)
	cr.Restore()

	// Draw color legend
	mw.drawColorLegend(cr, float64(width)-marginRight+20, marginTop, 60, availableHeight, minVal, maxVal)

	// If in comparison mode, draw differences
	if mw.compareMap != nil {
		mw.drawComparisonOverlay(cr, marginLeft, marginTop, cellWidth, cellHeight, rows, cols)
	}
}

// drawEmptyState draws a message when no file is loaded
func (mw *MainWindow) drawEmptyState(cr *cairo.Context, width, height int) {
	// Get theme colors
	textR, textG, textB, bgR, bgG, bgB := mw.getThemeColors()

	cr.SetSourceRGB(bgR, bgG, bgB)
	cr.Paint()

	cr.SetSourceRGB(textR*0.7, textG*0.7, textB*0.7)
	cr.SelectFontFace("Sans", cairo.FontSlantNormal, cairo.FontWeightNormal)
	cr.SetFontSize(20)

	text := "No ECU file loaded"
	extents := cr.TextExtents(text)
	cr.MoveTo(float64(width)/2-extents.Width/2, float64(height)/2)
	cr.ShowText(text)

	cr.SetFontSize(14)
	text = "Click 'Open ECU File' to begin"
	extents = cr.TextExtents(text)
	cr.MoveTo(float64(width)/2-extents.Width/2, float64(height)/2+30)
	cr.ShowText(text)
}

// valueToColor converts a value to RGB color for heatmap visualization
func (mw *MainWindow) valueToColor(value, minVal, maxVal float64) (float64, float64, float64) {
	// Normalize value to 0-1 range
	normalized := (value - minVal) / (maxVal - minVal)
	if math.IsNaN(normalized) {
		normalized = 0.5
	}

	// Use a blue -> cyan -> green -> yellow -> red gradient
	if normalized < 0.25 {
		// Blue to Cyan
		t := normalized / 0.25
		return 0, t, 1
	} else if normalized < 0.5 {
		// Cyan to Green
		t := (normalized - 0.25) / 0.25
		return 0, 1, 1 - t
	} else if normalized < 0.75 {
		// Green to Yellow
		t := (normalized - 0.5) / 0.25
		return t, 1, 0
	} else {
		// Yellow to Red
		t := (normalized - 0.75) / 0.25
		return 1, 1 - t, 0
	}
}

// findMinMax finds the minimum and maximum values in the map data
func (mw *MainWindow) findMinMax(data [][]float64) (float64, float64) {
	if len(data) == 0 || len(data[0]) == 0 {
		return 0, 1
	}

	minVal := data[0][0]
	maxVal := data[0][0]

	for _, row := range data {
		for _, val := range row {
			if val < minVal {
				minVal = val
			}
			if val > maxVal {
				maxVal = val
			}
		}
	}

	return minVal, maxVal
}

// drawColorLegend draws a color legend on the right side
func (mw *MainWindow) drawColorLegend(cr *cairo.Context, x, y, width, height, minVal, maxVal float64) {
	textR, textG, textB, _, _, _ := mw.getThemeColors()

	// Draw gradient bar
	numSteps := 100
	stepHeight := height / float64(numSteps)

	for i := 0; i < numSteps; i++ {
		value := minVal + (maxVal-minVal)*float64(numSteps-i)/float64(numSteps)
		r, g, b := mw.valueToColor(value, minVal, maxVal)

		cr.Rectangle(x, y+float64(i)*stepHeight, width, stepHeight)
		cr.SetSourceRGB(r, g, b)
		cr.Fill()
	}

	// Draw border
	cr.Rectangle(x, y, width, height)
	cr.SetSourceRGB(textR, textG, textB)
	cr.SetLineWidth(1)
	cr.Stroke()

	// Draw scale labels
	cr.SetSourceRGB(textR, textG, textB)
	cr.SelectFontFace("Sans", cairo.FontSlantNormal, cairo.FontWeightNormal)
	cr.SetFontSize(10)

	for i := 0; i <= 4; i++ {
		labelY := y + float64(i)*height/4
		value := maxVal - (maxVal-minVal)*float64(i)/4

		text := fmt.Sprintf("%.1f", value)
		extents := cr.TextExtents(text)
		cr.MoveTo(x+width+5, labelY+extents.Height/2)
		cr.ShowText(text)

		// Tick mark
		cr.MoveTo(x+width, labelY)
		cr.LineTo(x+width+4, labelY)
		cr.Stroke()
	}
}

// drawComparisonOverlay draws comparison indicators when comparing two files
func (mw *MainWindow) drawComparisonOverlay(cr *cairo.Context, marginLeft, marginTop, cellWidth, cellHeight float64, rows, cols int) {
	if mw.compareMap == nil {
		return
	}

	for row := 0; row < rows; row++ {
		for col := 0; col < cols; col++ {
			originalValue := mw.currentMap.Data[row][col]
			compareValue := mw.compareMap.Data[row][col]

			if math.Abs(originalValue-compareValue) > 0.01 {
				x := marginLeft + float64(col)*cellWidth
				y := marginTop + float64(row)*cellHeight

				// Draw a small indicator in the corner
				diff := compareValue - originalValue
				if diff > 0 {
					// Increased - green triangle
					cr.SetSourceRGBA(0, 1, 0, 0.7)
				} else {
					// Decreased - red triangle
					cr.SetSourceRGBA(1, 0, 0, 0.7)
				}

				// Draw small triangle in top-right corner
				cr.MoveTo(x+cellWidth-8, y+2)
				cr.LineTo(x+cellWidth-2, y+2)
				cr.LineTo(x+cellWidth-2, y+8)
				cr.ClosePath()
				cr.Fill()
			}
		}
	}
}

// getCellAtPosition returns the row and column for a given mouse position
func (mw *MainWindow) getCellAtPosition(x, y float64, width, height int) (row, col int, valid bool) {
	if mw.currentMap == nil {
		return 0, 0, false
	}

	rows := mw.currentMap.Config.Rows
	cols := mw.currentMap.Config.Cols

	marginLeft := 80.0
	marginRight := 100.0
	marginTop := 60.0
	marginBottom := 80.0

	availableWidth := float64(width) - marginLeft - marginRight
	availableHeight := float64(height) - marginTop - marginBottom

	cellWidth := availableWidth / float64(cols)
	cellHeight := availableHeight / float64(rows)

	// Check if click is within map area
	if x < marginLeft || x > marginLeft+availableWidth || y < marginTop || y > marginTop+availableHeight {
		return 0, 0, false
	}

	col = int((x - marginLeft) / cellWidth)
	row = int((y - marginTop) / cellHeight)

	if row >= 0 && row < rows && col >= 0 && col < cols {
		return row, col, true
	}

	return 0, 0, false
}
