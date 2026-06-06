package main

import (
	"context"
	"embed"
	"log"

	vfoxapp "vfoxG/internal/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := vfoxapp.NewApp()

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
		OnStartup: func(ctx context.Context) {
			vfoxapp.Startup(app, ctx)
		},
		OnDomReady: func(ctx context.Context) {
			runtime.WindowShow(ctx)
		},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "com.huajifruit.vfoxg",
			OnSecondInstanceLaunch: func(_ options.SecondInstanceData) {
				vfoxapp.ShowMainWindow(app)
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
