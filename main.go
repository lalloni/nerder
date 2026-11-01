package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"nerder/internal/app"
	"nerder/internal/catalog"
	"nerder/internal/fetch"
	"nerder/internal/installer"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	cat, err := catalog.Load()
	if err != nil {
		log.Fatal(err)
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		log.Fatal(err)
	}
	a := app.New(cat, fetch.New(filepath.Join(cacheDir, "nerder"), catalog.ReleaseTag), installer.New(cat))
	a.OpenURL = wailsruntime.BrowserOpenURL

	err = wails.Run(&options.App{
		Title:  "nerder",
		Width:  1100,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: app.NewFontHandler(a),
		},
		BackgroundColour: &options.RGBA{R: 250, G: 250, B: 249, A: 1},
		OnStartup:        a.Startup,
		Bind:             []interface{}{a},
	})
	if err != nil {
		log.Fatal(err)
	}
}
