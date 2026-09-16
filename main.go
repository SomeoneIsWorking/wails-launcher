package main

import (
	"context"
	"embed"

	"wails-launcher/pkg/launcher"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:build/bin
var assets embed.FS

// App is the GUI shell around the launcher. It exists so the generated frontend
// bindings stay on main.App, and so everything window-shaped (events, the file
// dialog) lives here rather than in the launcher itself.
type App struct {
	*launcher.App
	ctx context.Context
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.Startup(ctx)
}

// emit forwards a service event to the window.
func (a *App) emit(event string, serviceId string, data interface{}) {
	runtime.EventsEmit(a.ctx, "serviceEvent", map[string]interface{}{
		"type":      event,
		"serviceId": serviceId,
		"data":      data,
	})
}

// pick opens the OS file dialog.
func (a *App) pick(title string, filterName string, pattern string) (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   title,
		Filters: []runtime.FileFilter{{DisplayName: filterName, Pattern: pattern}},
	})
}

func (a *App) shutdown(ctx context.Context) {
	a.Shutdown(ctx)
}

func main() {
	app := &App{}
	app.App = launcher.NewAppWith(app.emit, app.pick)

	err := wails.Run(&options.App{
		Title:  "wails-launcher",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
		HideWindowOnClose: true,
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
