package main

import (
	"embed"
	_ "embed"
	"log"
	"runtime"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	// Register a custom event whose associated data type is string.
	// This is not required, but the binding generator will pick up registered events
	// and provide a strongly typed JS/TS API for them.
	application.RegisterEvent[string]("time")
}

// main function serves as the application's entry point. It initializes the application, creates a window,
// and starts a goroutine that emits a time-based event every second. It subsequently runs the application and
// logs any error that might occur.
func main() {
	const settingsWindowName = "settings"
	const settingsShortcut = "CmdOrCtrl+,"
	const appName = "LevelDB Editor"

	var showSettingsWindow func()
	var persistMainWindowState func()
	var mainWindow application.Window
	var settingsWindow application.Window
	isWindows := runtime.GOOS == "windows"
	keyBindings := map[string]func(window application.Window){}
	var stateStore *windowStateStore
	windowState := persistedWindowState{
		Version: 1,
		Mode:    windowModeNormal,
	}
	lastNormalBounds := windowBounds{}

	if !isWindows {
		keyBindings[settingsShortcut] = func(window application.Window) {
			if showSettingsWindow != nil {
				showSettingsWindow()
			}
		}
	}

	greetService := &GreetService{}
	levelDBService := &LevelDBService{}
	windowService := &WindowService{}

	var err error
	stateStore, err = newWindowStateStore(appName)
	if err != nil {
		log.Printf("window state persistence disabled: %v", err)
		stateStore = nil
	} else {
		loadedState, loadErr := stateStore.Load()
		if loadErr != nil {
			log.Printf("failed to load window state; using defaults: %v", loadErr)
		} else {
			windowState = loadedState
			if bounds, ok := loadedState.startupBounds(); ok {
				lastNormalBounds = bounds
			}
		}
	}

	// Create a new Wails application by providing the necessary options.
	// Variables 'Name' and 'Description' are for application metadata.
	// 'Assets' configures the asset server with the 'FS' variable pointing to the frontend files.
	// 'Bind' is a list of Go struct instances. The frontend has access to the methods of these instances.
	// 'Mac' options tailor the application when running an macOS.
	app := application.New(application.Options{
		Name:        appName,
		Description: "View and edit LevelDB databases",
		Services: []application.Service{
			application.NewService(greetService),
			application.NewService(levelDBService),
			application.NewService(windowService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		KeyBindings: keyBindings,
		OnShutdown: func() {
			if persistMainWindowState != nil {
				persistMainWindowState()
			}
		},
	})
	windowService.SetApp(app)

	applyWindowsMenuVisibilityPolicy := func(window application.Window) {
		if !isWindows || window == nil {
			return
		}
		window.HideMenuBar()
	}

	showSettingsWindow = func() {
		if settingsWindow == nil {
			var exists bool
			settingsWindow, exists = app.Window.GetByName(settingsWindowName)
			if !exists {
				settingsWindow = nil
			}
		}

		if settingsWindow == nil {
			settingsWindow = app.Window.NewWithOptions(application.WebviewWindowOptions{
				Name:                settingsWindowName,
				Title:               "Preferences",
				Width:               860,
				Height:              620,
				MinWidth:            860,
				MinHeight:           620,
				MaxWidth:            860,
				MaxHeight:           620,
				DisableResize:       true,
				MinimiseButtonState: application.ButtonDisabled,
				MaximiseButtonState: application.ButtonDisabled,
				Hidden:              true,
				BackgroundColour:    application.NewRGB(242, 242, 247),
				URL:                 "/settings",
				UseApplicationMenu:  true,
				Mac: application.MacWindow{
					Backdrop:           application.MacBackdropNormal,
					TitleBar:           application.MacTitleBarDefault,
					CollectionBehavior: application.MacWindowCollectionBehaviorFullScreenNone,
				},
			})
			settingsWindow.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
				settingsWindow = nil
			})
			applyWindowsMenuVisibilityPolicy(settingsWindow)
		}
		if settingsWindow.IsMinimised() {
			settingsWindow.UnMinimise()
		}
		settingsWindow.Show()
		settingsWindow.Focus()
	}

	menu := app.Menu.New()

	if !isWindows {
		appMenu := menu.AddSubmenu("LevelDB Editor")
		appMenu.AddRole(application.About)
		appMenu.AddSeparator()
		if runtime.GOOS == "darwin" {
			appMenu.Add("Settings...").
				SetAccelerator(settingsShortcut).
				OnClick(func(_ *application.Context) {
					showSettingsWindow()
				})
			appMenu.AddSeparator()
			appMenu.AddRole(application.ServicesMenu)
			appMenu.AddSeparator()
			appMenu.AddRole(application.Hide)
			appMenu.AddRole(application.HideOthers)
			appMenu.AddRole(application.UnHide)
			appMenu.AddSeparator()
		}
		appMenu.AddRole(application.Quit)
	}

	fileMenu := menu.AddSubmenu("File")
	if isWindows {
		preferencesMenu := fileMenu.AddSubmenu("Preferences")
		preferencesMenu.Add("Settings").OnClick(func(_ *application.Context) {
			showSettingsWindow()
		})
		fileMenu.AddSeparator()
	}
	fileMenu.AddRole(application.CloseWindow)
	menu.AddRole(application.EditMenu)
	menu.AddRole(application.ViewMenu)
	menu.AddRole(application.WindowMenu)
	menu.AddRole(application.HelpMenu)
	app.Menu.Set(menu)

	// Create a new window with the necessary options.
	// 'Title' is the title of the window.
	// 'Mac' options tailor the window when running on macOS.
	// 'BackgroundColour' is the background colour of the window.
	// 'URL' is the URL that will be loaded into the webview.
	mainWindowOptions := application.WebviewWindowOptions{
		Name:               "main",
		Title:              appName,
		Width:              1400,
		Height:             900,
		UseApplicationMenu: true,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	}
	if bounds, ok := windowState.startupBounds(); ok {
		mainWindowOptions.Width = bounds.Width
		mainWindowOptions.Height = bounds.Height
		mainWindowOptions.InitialPosition = application.WindowXY
		mainWindowOptions.X = bounds.X
		mainWindowOptions.Y = bounds.Y
	}
	switch windowState.modeOrDefault() {
	case windowModeMaximised:
		mainWindowOptions.StartState = application.WindowStateMaximised
	case windowModeFullscreen:
		mainWindowOptions.StartState = application.WindowStateFullscreen
	}

	mainWindow = app.Window.NewWithOptions(mainWindowOptions)
	applyWindowsMenuVisibilityPolicy(mainWindow)
	persistMainWindowState = func() {
		if stateStore == nil || mainWindow == nil {
			return
		}

		updatedState := windowState
		switch {
		case mainWindow.IsFullscreen():
			updatedState.Mode = windowModeFullscreen
		case mainWindow.IsMaximised():
			updatedState.Mode = windowModeMaximised
		default:
			updatedState.Mode = windowModeNormal
		}

		currentBounds := windowBoundsFromRect(mainWindow.Bounds())
		if updatedState.Mode == windowModeNormal {
			if currentBounds.valid() {
				updatedState.Bounds = currentBounds
				updatedState.NormalBounds = currentBounds
				lastNormalBounds = currentBounds
			}
		} else {
			if currentBounds.valid() {
				updatedState.Bounds = currentBounds
			}
			if lastNormalBounds.valid() {
				updatedState.NormalBounds = lastNormalBounds
			}
		}

		if err := stateStore.Save(updatedState); err != nil {
			log.Printf("failed to save window state: %v", err)
			return
		}
		windowState = updatedState
	}

	mainWindow.OnWindowEvent(events.Common.WindowDidMove, func(_ *application.WindowEvent) {
		if mainWindow.IsMinimised() || mainWindow.IsMaximised() || mainWindow.IsFullscreen() {
			return
		}
		bounds := windowBoundsFromRect(mainWindow.Bounds())
		if bounds.valid() {
			lastNormalBounds = bounds
		}
		persistMainWindowState()
	})
	mainWindow.OnWindowEvent(events.Common.WindowDidResize, func(_ *application.WindowEvent) {
		if mainWindow.IsMinimised() || mainWindow.IsMaximised() || mainWindow.IsFullscreen() {
			return
		}
		bounds := windowBoundsFromRect(mainWindow.Bounds())
		if bounds.valid() {
			lastNormalBounds = bounds
		}
		persistMainWindowState()
	})
	mainWindow.OnWindowEvent(events.Common.WindowClosing, func(_ *application.WindowEvent) {
		persistMainWindowState()
	})

	// Create a goroutine that emits an event containing the current time every second.
	// The frontend can listen to this event and update the UI accordingly.
	go func() {
		for {
			now := time.Now().Format(time.RFC1123)
			app.Event.Emit("time", now)
			time.Sleep(time.Second)
		}
	}()

	// Run the application. This blocks until the application has been exited.
	err = app.Run()

	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}
