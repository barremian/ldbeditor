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

	var showSettingsWindow func()
	var mainWindow application.Window
	var settingsWindow application.Window
	var settingsWindowReady bool
	var settingsWindowShowPending bool
	var unregisterSettingsWindowLoaded func()
	isWindows := runtime.GOOS == "windows"

	greetService := &GreetService{}
	levelDBService := &LevelDBService{}
	windowService := &WindowService{}

	// Create a new Wails application by providing the necessary options.
	// Variables 'Name' and 'Description' are for application metadata.
	// 'Assets' configures the asset server with the 'FS' variable pointing to the frontend files.
	// 'Bind' is a list of Go struct instances. The frontend has access to the methods of these instances.
	// 'Mac' options tailor the application when running an macOS.
	app := application.New(application.Options{
		Name:        "LevelDB Editor",
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
		KeyBindings: map[string]func(window application.Window){
			settingsShortcut: func(window application.Window) {
				if showSettingsWindow != nil {
					showSettingsWindow()
				}
			},
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
		showAndFocusSettingsWindow := func() {
			if settingsWindow == nil {
				return
			}
			if settingsWindow.IsMinimised() {
				settingsWindow.UnMinimise()
			}
			settingsWindow.Show()
			settingsWindow.Focus()
		}

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
			applyWindowsMenuVisibilityPolicy(settingsWindow)

			settingsWindowReady = false
			settingsWindowShowPending = false
			if unregisterSettingsWindowLoaded != nil {
				unregisterSettingsWindowLoaded()
				unregisterSettingsWindowLoaded = nil
			}

			var settingsReadyEvent events.WindowEventType
			switch runtime.GOOS {
			case "darwin":
				settingsReadyEvent = events.Mac.WebViewDidFinishNavigation
			case "windows":
				settingsReadyEvent = events.Windows.WebViewNavigationCompleted
			default:
				settingsReadyEvent = events.Linux.WindowLoadFinished
			}
			unregisterSettingsWindowLoaded = settingsWindow.OnWindowEvent(settingsReadyEvent, func(_ *application.WindowEvent) {
				settingsWindowReady = true
				if unregisterSettingsWindowLoaded != nil {
					unregisterSettingsWindowLoaded()
					unregisterSettingsWindowLoaded = nil
				}
				if settingsWindowShowPending {
					settingsWindowShowPending = false
					showAndFocusSettingsWindow()
				}
			})
		}
		if settingsWindowReady {
			showAndFocusSettingsWindow()
			return
		}

		// Keep the window hidden until the first render completes
		// to avoid flashing a blank white webview.
		settingsWindowShowPending = true
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
		preferencesMenu.Add("Settings").
			SetAccelerator(settingsShortcut).
			OnClick(func(_ *application.Context) {
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
	mainWindow = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:               "main",
		Title:              "LevelDB Editor",
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
	})
	applyWindowsMenuVisibilityPolicy(mainWindow)

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
	err := app.Run()

	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}
