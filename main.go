package main

import (
	"context"
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "vfoxG",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 254, G: 251, B: 255, A: 255},
		StartHidden:      true,
		OnStartup:        app.startup,
		OnDomReady: func(ctx context.Context) {
			runtime.WindowShow(ctx)
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "com.huajifruit.vfoxg",
			OnSecondInstanceLaunch: func(_ options.SecondInstanceData) {
				if app.ctx == nil {
					return
				}
				runtime.WindowShow(app.ctx)
				runtime.WindowUnminimise(app.ctx)
			},
		},
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Println("Error:", err.Error())
	}
}
