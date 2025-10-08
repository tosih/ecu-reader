package main

import (
	"os"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/tosih/motronic-m21-tool/pkg/gui"
)

func main() {
	app := gtk.NewApplication("com.github.tosih.motronic-m21-tool", gio.ApplicationFlagsNone)
	app.ConnectActivate(func() {
		gui.NewMainWindow(app)
	})

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}
