package main

import (
	backendapp "backend-go/app"
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	assetserver "github.com/wailsapp/wails/v2/pkg/options/assetserver"
	macoptions "github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app, err := backendapp.New()
	if err != nil {
		log.Fatalf("failed to initialize desktop backend: %v", err)
	}

	handler, err := newDesktopHandler(app.Handler, assets)
	if err != nil {
		_ = app.Close()
		log.Fatalf("failed to initialize desktop asset handler: %v", err)
	}

	err = wails.Run(&options.App{
		Title:     "SKM",
		Width:     1440,
		Height:    920,
		MinWidth:  1100,
		MinHeight: 720,
		Mac: &macoptions.Options{
			TitleBar: macoptions.TitleBarHiddenInset(),
		},
		AssetServer: &assetserver.Options{
			Assets:  assets,
			Handler: handler,
		},
		OnShutdown: func(context.Context) {
			_ = app.Close()
		},
	})
	if err != nil {
		log.Fatalf("failed to run desktop app: %v", err)
	}
}

func newDesktopHandler(api http.Handler, assets embed.FS) (http.Handler, error) {
	distFS, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		return nil, err
	}

	spa := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isBackendRoute(r.URL.Path) {
			api.ServeHTTP(w, r)
			return
		}

		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		clone := r.Clone(r.Context())
		urlCopy := *clone.URL
		clone.URL = &urlCopy
		clone.URL.Path = "/index.html"
		spa.ServeHTTP(w, clone)
	}), nil
}

func isBackendRoute(path string) bool {
	return path == "/healthz" || path == "/version" || path == "/api" || strings.HasPrefix(path, "/api/")
}
