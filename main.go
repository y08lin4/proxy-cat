package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/y08lin4/proxy-cat/internal/app"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	application := app.New()

	err := wails.Run(&options.App{
		Title:  "Proxy-Cat",
		Width:  1160,
		Height: 760,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:     application.Startup,
		OnDomReady:    application.DomReady,
		OnBeforeClose: application.BeforeClose,
		OnShutdown:    application.Shutdown,
		Bind: []interface{}{
			application,
		},
	})
	if err != nil {
		println("Proxy-Cat failed to start:", err.Error())
	}
}
