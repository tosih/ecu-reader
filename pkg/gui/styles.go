package gui

import (
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// applyCSSStyles applies custom CSS for dark mode compatibility
func (mw *MainWindow) applyCSSStyles() {
	cssProvider := gtk.NewCSSProvider()

	css := `
/* Main window styling */
window {
	background-color: @theme_bg_color;
}

/* Sidebar styling */
.sidebar {
	background-color: @theme_bg_color;
	border-right: 1px solid @borders;
	padding: 10px;
}

.sidebar-header {
	font-weight: bold;
	font-size: 14pt;
	margin-bottom: 10px;
	color: @theme_fg_color;
}

/* Map list items */
listboxrow {
	border-radius: 6px;
	margin: 2px 0;
}

listboxrow:selected {
	background-color: @theme_selected_bg_color;
}

.map-name {
	font-weight: bold;
	font-size: 11pt;
	color: @theme_fg_color;
}

.map-detail {
	font-size: 9pt;
	color: alpha(@theme_fg_color, 0.7);
}

/* Status bar */
.statusbar {
	padding: 8px 12px;
	background-color: @theme_bg_color;
	border-top: 1px solid @borders;
	font-size: 10pt;
	color: @theme_fg_color;
}

/* Config parameters view */
.config-header {
	font-weight: bold;
	font-size: 14pt;
	margin-bottom: 10px;
	color: @theme_fg_color;
}

.param-name {
	font-weight: bold;
	font-size: 11pt;
	color: @theme_fg_color;
}

.param-description {
	font-size: 9pt;
	color: alpha(@theme_fg_color, 0.7);
}

.param-value {
	font-family: monospace;
	font-size: 11pt;
	font-weight: bold;
	color: @accent_color;
}

.current-value {
	font-family: monospace;
	font-weight: bold;
	color: @accent_color;
}

/* Warning labels */
.warning-text {
	color: #ff6b6b;
	font-weight: bold;
}

/* Scanner view */
.scanner-header {
	font-weight: bold;
	font-size: 14pt;
	margin-bottom: 10px;
	color: @theme_fg_color;
}

/* Notebook tabs */
notebook > header {
	background-color: @theme_bg_color;
}

notebook > header > tabs > tab {
	color: @theme_fg_color;
	padding: 8px 16px;
}

notebook > header > tabs > tab:checked {
	background-color: @theme_selected_bg_color;
	font-weight: bold;
}

/* Drawing area - ensure it has proper background */
drawingarea {
	background-color: @theme_base_color;
}

/* Buttons */
button {
	color: @theme_fg_color;
}

button:hover {
	background-color: alpha(@theme_fg_color, 0.1);
}

/* Entry fields */
entry {
	color: @theme_fg_color;
	background-color: @theme_base_color;
	caret-color: @theme_fg_color;
}

/* Labels in dialogs */
label {
	color: @theme_fg_color;
}

/* Dialog styling */
dialog {
	background-color: @theme_bg_color;
}

dialog > box {
	color: @theme_fg_color;
}

/* Message dialogs */
messagedialog {
	background-color: @theme_bg_color;
}

messagedialog label {
	color: @theme_fg_color;
}

/* Scrolled windows */
scrolledwindow {
	background-color: @theme_base_color;
}

/* Ensure proper text color in various contexts */
box label,
listboxrow label {
	color: @theme_fg_color;
}
`

	cssProvider.LoadFromData(css)

	// Get display from GDK
	display := gdk.DisplayGetDefault()

	gtk.StyleContextAddProviderForDisplay(
		display,
		cssProvider,
		gtk.STYLE_PROVIDER_PRIORITY_APPLICATION,
	)
}
