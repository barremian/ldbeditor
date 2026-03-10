package main

import (
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// WindowService provides desktop window-level actions for the frontend.
type WindowService struct {
	app *application.App
}

func (s *WindowService) SetApp(app *application.App) {
	s.app = app
}

// ToggleCurrentWindowMenuBar toggles the active window menu bar on Windows.
func (s *WindowService) ToggleCurrentWindowMenuBar() {
	if runtime.GOOS != "windows" || s.app == nil {
		return
	}

	window := s.app.Window.Current()
	if window == nil {
		return
	}
	window.ToggleMenuBar()
}
