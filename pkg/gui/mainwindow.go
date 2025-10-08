package gui

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/tosih/motronic-m21-tool/pkg/models"
	"github.com/tosih/motronic-m21-tool/pkg/reader"
)

// MainWindow represents the main application window
type MainWindow struct {
	app            *gtk.Application
	window         *gtk.ApplicationWindow
	currentFile    string
	currentMap     *models.ECUMap
	selectedMapIdx int

	// UI Components
	headerBar      *gtk.HeaderBar
	mainBox        *gtk.Box
	sidebar        *gtk.Box
	contentArea    *gtk.Box
	mapListView    *gtk.ListBox
	mapDrawArea    *gtk.DrawingArea
	statusBar      *gtk.Label
	configTreeView *gtk.TreeView
	notebookTabs   *gtk.Notebook

	// Comparison mode
	compareFile string
	compareMap  *models.ECUMap
}

// NewMainWindow creates and displays the main application window
func NewMainWindow(app *gtk.Application) *MainWindow {
	mw := &MainWindow{
		app:            app,
		selectedMapIdx: 0,
	}

	mw.buildUI()
	mw.setupActions()
	mw.window.Show()

	return mw
}

// buildUI constructs the entire window layout
func (mw *MainWindow) buildUI() {
	// Create main window
	mw.window = gtk.NewApplicationWindow(mw.app)
	mw.window.SetTitle("Motronic M2.1 ECU Tool")
	mw.window.SetDefaultSize(1200, 800)

	// Create header bar with menu
	mw.headerBar = gtk.NewHeaderBar()
	mw.window.SetTitlebar(mw.headerBar)

	// Add menu button
	menuButton := mw.createMenuButton()
	mw.headerBar.PackStart(menuButton)

	// Add open file button
	openButton := gtk.NewButtonWithLabel("Open ECU File")
	openButton.ConnectClicked(func() {
		mw.openFileDialog()
	})
	mw.headerBar.PackStart(openButton)

	// Add compare button
	compareButton := gtk.NewButtonWithLabel("Compare Files")
	compareButton.ConnectClicked(func() {
		mw.openCompareDialog()
	})
	mw.headerBar.PackStart(compareButton)

	// Main content box (horizontal split)
	mw.mainBox = gtk.NewBox(gtk.OrientationHorizontal, 0)
	mw.window.SetChild(mw.mainBox)

	// Left sidebar for map selection
	mw.buildSidebar()

	// Right content area with notebook tabs
	mw.buildContentArea()

	// Status bar at bottom
	mw.statusBar = gtk.NewLabel("Ready. Open an ECU file to begin.")
	mw.statusBar.SetXAlign(0)
	mw.statusBar.AddCSSClass("statusbar")

	// Overall vertical layout
	vbox := gtk.NewBox(gtk.OrientationVertical, 0)
	vbox.Append(mw.mainBox)
	vbox.Append(mw.statusBar)
	mw.window.SetChild(vbox)
}

// buildSidebar creates the left sidebar with map list
func (mw *MainWindow) buildSidebar() {
	mw.sidebar = gtk.NewBox(gtk.OrientationVertical, 5)
	mw.sidebar.SetSizeRequest(250, -1)
	mw.sidebar.AddCSSClass("sidebar")

	// Sidebar header
	sidebarLabel := gtk.NewLabel("ECU Maps")
	sidebarLabel.AddCSSClass("sidebar-header")
	sidebarLabel.SetXAlign(0)
	mw.sidebar.Append(sidebarLabel)

	// Scrolled window for map list
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetVExpand(true)
	scrolled.SetPolicy(gtk.PolicyNever, gtk.PolicyAutomatic)

	// List box for maps
	mw.mapListView = gtk.NewListBox()
	mw.mapListView.SetSelectionMode(gtk.SelectionSingle)
	mw.mapListView.ConnectRowSelected(mw.onMapSelected)
	scrolled.SetChild(mw.mapListView)

	// Populate map list
	mw.populateMapList()

	mw.sidebar.Append(scrolled)

	// Add separator
	separator := gtk.NewSeparator(gtk.OrientationVertical)

	// Add sidebar and separator to main box
	mw.mainBox.Append(mw.sidebar)
	mw.mainBox.Append(separator)
}

// buildContentArea creates the main content area with tabs
func (mw *MainWindow) buildContentArea() {
	mw.contentArea = gtk.NewBox(gtk.OrientationVertical, 0)
	mw.contentArea.SetHExpand(true)

	// Notebook with tabs
	mw.notebookTabs = gtk.NewNotebook()

	// Tab 1: Map Visualization
	mw.mapDrawArea = gtk.NewDrawingArea()
	mw.mapDrawArea.SetDrawFunc(mw.drawMapFunc)
	mw.mapDrawArea.SetSizeRequest(800, 600)

	// Add mouse click handler for cell editing
	clickGesture := gtk.NewGestureClick()
	clickGesture.SetButton(1) // Left click
	clickGesture.ConnectPressed(mw.onMapClicked)
	mw.mapDrawArea.AddController(clickGesture)

	mapScrolled := gtk.NewScrolledWindow()
	mapScrolled.SetChild(mw.mapDrawArea)
	mw.notebookTabs.AppendPage(mapScrolled, gtk.NewLabel("Map View"))

	// Tab 2: Configuration Parameters
	configBox := mw.buildConfigView()
	mw.notebookTabs.AppendPage(configBox, gtk.NewLabel("Config Parameters"))

	// Tab 3: Scanner
	scannerBox := mw.buildScannerView()
	mw.notebookTabs.AppendPage(scannerBox, gtk.NewLabel("Scanner"))

	mw.contentArea.Append(mw.notebookTabs)
	mw.mainBox.Append(mw.contentArea)
}

// populateMapList fills the sidebar with available maps
func (mw *MainWindow) populateMapList() {
	for i, mapConfig := range models.MapConfigs {
		row := gtk.NewListBoxRow()

		box := gtk.NewBox(gtk.OrientationVertical, 2)
		box.SetMarginStart(10)
		box.SetMarginEnd(10)
		box.SetMarginTop(5)
		box.SetMarginBottom(5)

		nameLabel := gtk.NewLabel(mapConfig.Name)
		nameLabel.SetXAlign(0)
		nameLabel.AddCSSClass("map-name")

		detailLabel := gtk.NewLabel(fmt.Sprintf("%dx%d - %s", mapConfig.Rows, mapConfig.Cols, mapConfig.Unit))
		detailLabel.SetXAlign(0)
		detailLabel.AddCSSClass("map-detail")

		box.Append(nameLabel)
		box.Append(detailLabel)

		row.SetChild(box)
		row.SetName(fmt.Sprintf("%d", i))
		mw.mapListView.Append(row)
	}
}

// createMenuButton creates the application menu
func (mw *MainWindow) createMenuButton() *gtk.MenuButton {
	menuButton := gtk.NewMenuButton()
	menuButton.SetIconName("open-menu-symbolic")

	menu := gio.NewMenu()

	// File menu section
	fileSection := gio.NewMenu()
	fileSection.Append("Open File...", "app.open")
	fileSection.Append("Export to CSV...", "app.export")
	fileSection.Append("Quit", "app.quit")
	menu.AppendSection("", fileSection)

	// Tools menu section
	toolsSection := gio.NewMenu()
	toolsSection.Append("Scanner", "app.scanner")
	toolsSection.Append("Compare Files", "app.compare")
	menu.AppendSection("", toolsSection)

	// Help menu section
	helpSection := gio.NewMenu()
	helpSection.Append("About", "app.about")
	menu.AppendSection("", helpSection)

	menuButton.SetMenuModel(menu)
	return menuButton
}

// setupActions configures application actions
func (mw *MainWindow) setupActions() {
	// Open action
	openAction := gio.NewSimpleAction("open", nil)
	openAction.ConnectActivate(func(param *gio.Variant) {
		mw.openFileDialog()
	})
	mw.app.AddAction(openAction)

	// Export action
	exportAction := gio.NewSimpleAction("export", nil)
	exportAction.ConnectActivate(func(param *gio.Variant) {
		mw.exportDialog()
	})
	mw.app.AddAction(exportAction)

	// Compare action
	compareAction := gio.NewSimpleAction("compare", nil)
	compareAction.ConnectActivate(func(param *gio.Variant) {
		mw.openCompareDialog()
	})
	mw.app.AddAction(compareAction)

	// Scanner action
	scannerAction := gio.NewSimpleAction("scanner", nil)
	scannerAction.ConnectActivate(func(param *gio.Variant) {
		mw.notebookTabs.SetCurrentPage(2) // Switch to scanner tab
	})
	mw.app.AddAction(scannerAction)

	// About action
	aboutAction := gio.NewSimpleAction("about", nil)
	aboutAction.ConnectActivate(func(param *gio.Variant) {
		mw.showAboutDialog()
	})
	mw.app.AddAction(aboutAction)

	// Quit action
	quitAction := gio.NewSimpleAction("quit", nil)
	quitAction.ConnectActivate(func(param *gio.Variant) {
		mw.window.Close()
	})
	mw.app.AddAction(quitAction)
}

// openFileDialog shows file chooser for opening ECU files
func (mw *MainWindow) openFileDialog() {
	dialog := gtk.NewFileChooserDialog(
		"Open ECU Binary File",
		&mw.window.Window,
		gtk.FileChooserActionOpen,
	)

	dialog.AddButton("Cancel", int(gtk.ResponseCancel))
	dialog.AddButton("Open", int(gtk.ResponseAccept))
	dialog.SetModal(true)

	// Add file filter for .bin files
	filter := gtk.NewFileFilter()
	filter.SetName("ECU Binary Files (*.bin)")
	filter.AddPattern("*.bin")
	filter.AddPattern("*.BIN")
	dialog.AddFilter(filter)

	allFilter := gtk.NewFileFilter()
	allFilter.SetName("All Files")
	allFilter.AddPattern("*")
	dialog.AddFilter(allFilter)

	// Set default folder if it exists
	if _, err := os.Stat("bins"); err == nil {
		binFile := gio.NewFileForPath("bins")
		dialog.SetCurrentFolder(binFile)
	}

	dialog.ConnectResponse(func(responseID int) {
		if responseID == int(gtk.ResponseAccept) {
			file := dialog.File()
			if file != nil {
				path := file.Path()
				mw.loadECUFile(path)
			}
		}
		dialog.Destroy()
	})

	dialog.Show()
}

// loadECUFile loads an ECU binary file
func (mw *MainWindow) loadECUFile(filename string) {
	mw.currentFile = filename

	// Update window title
	mw.window.SetTitle(fmt.Sprintf("Motronic M2.1 ECU Tool - %s", filepath.Base(filename)))

	// Load the currently selected map
	mw.loadCurrentMap()

	// Update status
	mw.statusBar.SetText(fmt.Sprintf("Loaded: %s", filename))
}

// loadCurrentMap loads the currently selected map from the file
func (mw *MainWindow) loadCurrentMap() {
	if mw.currentFile == "" {
		return
	}

	if mw.selectedMapIdx >= len(models.MapConfigs) {
		return
	}

	mapConfig := models.MapConfigs[mw.selectedMapIdx]

	// Read the map
	ecuMap, err := reader.ReadMap(mw.currentFile, mapConfig)
	if err != nil {
		mw.showErrorDialog(fmt.Sprintf("Error reading map: %v", err))
		return
	}

	mw.currentMap = ecuMap

	// If in comparison mode, load comparison map too
	if mw.compareFile != "" {
		compareMap, err := reader.ReadMap(mw.compareFile, mapConfig)
		if err != nil {
			mw.showErrorDialog(fmt.Sprintf("Error reading comparison map: %v", err))
			return
		}
		mw.compareMap = compareMap
	}

	// Redraw
	mw.mapDrawArea.QueueDraw()
}

// onMapSelected handles map selection from sidebar
func (mw *MainWindow) onMapSelected(row *gtk.ListBoxRow) {
	if row == nil {
		return
	}

	name := row.Name()
	var idx int
	fmt.Sscanf(name, "%d", &idx)

	mw.selectedMapIdx = idx
	mw.loadCurrentMap()
}

// showErrorDialog displays an error message
func (mw *MainWindow) showErrorDialog(message string) {
	dialog := gtk.NewMessageDialog(
		&mw.window.Window,
		gtk.DialogModal,
		gtk.MessageError,
		gtk.ButtonsOK,
		"%s",
		message,
	)
	dialog.ConnectResponse(func(response int) {
		dialog.Destroy()
	})
	dialog.Show()
}

// showAboutDialog displays the about dialog
func (mw *MainWindow) showAboutDialog() {
	about := gtk.NewAboutDialog()
	about.SetTransientFor(&mw.window.Window)
	about.SetProgramName("Motronic M2.1 ECU Tool")
	about.SetVersion("1.0.0")
	about.SetComments("Read, analyze, and edit Motronic M2.1 ECU binary files")
	about.SetWebsite("https://github.com/tosih/motronic-m21-tool")
	about.SetAuthors([]string{"Motronic M2.1 Tool Contributors"})
	about.SetLicense("MIT License")
	about.Show()
}
