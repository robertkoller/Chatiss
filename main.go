package main

import (
	"embed"

	"github.com/robertkoller/Chatiss/app"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	a := app.NewApp()
	err := wails.Run(&options.App{
		Title:  "Chatiss",
		Width:  1100,
		Height: 720,
		MinWidth: 800,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 18, G: 18, B: 18, A: 1},
		OnStartup:        a.Startup,
		Bind:             []interface{}{a},
	})
	if err != nil {
		println("Error:", err.Error())
	}
}
