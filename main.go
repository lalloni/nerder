package main

import (
	"context"
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
	"nerder/internal/winstate"
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

	statePath, err := winstate.Path()
	if err != nil {
		log.Fatal(err)
	}
	state := winstate.Load(statePath)
	startState := options.Normal
	if state.Maximised {
		startState = options.Maximised
	}

	err = wails.Run(&options.App{
		Title:            "nerder",
		Width:            state.Width,
		Height:           state.Height,
		WindowStartState: startState,
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: app.NewFontHandler(a),
		},
		BackgroundColour: &options.RGBA{R: 250, G: 250, B: 249, A: 1},
		OnStartup: func(ctx context.Context) {
			a.Startup(ctx)
			if state.HasPos && !state.Maximised {
				wailsruntime.WindowSetPosition(ctx, state.X, state.Y)
			}
		},
		OnBeforeClose: func(ctx context.Context) bool {
			w, h := wailsruntime.WindowGetSize(ctx)
			x, y := wailsruntime.WindowGetPosition(ctx)
			max := wailsruntime.WindowIsMaximised(ctx)
			if err := winstate.Save(statePath, winstate.Merge(state, w, h, x, y, max)); err != nil {
				log.Printf("saving window state: %v", err)
			}
			return false
		},
		Bind: []interface{}{a},
	})
	if err != nil {
		log.Fatal(err)
	}
}
