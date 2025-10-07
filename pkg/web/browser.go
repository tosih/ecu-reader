package web

import (
	"os/exec"
	"runtime"
)

// openBrowser tries to open the default browser with the given URL
func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	}

	if err != nil {
		// Silently fail if we can't open the browser
		// User can manually navigate to the URL
	}
}
