package main

import (
	"GestorCuentas/backend"
	"embed"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

func main() {
	// Wayland no soporta gtk_window_set_icon() en la decoración de ventana.
	// En Wayland (KDE Plasma) el icono del título depende del .desktop, no de GTK.
	// Forzamos X11 backend en Wayland para que el icono funcione en título + barra + lanzador.
	if os.Getenv("XDG_SESSION_TYPE") == "wayland" && os.Getenv("GDK_BACKEND") == "" {
		os.Setenv("GDK_BACKEND", "x11")
	}

	// Create an instance of the app structure
	app := NewApp()

	// Blindaje: Asegurar que la clave se borre incluso si hay un pánico (crash)
	defer func() {
		if r := recover(); r != nil {
			backend.Lock()
			panic(r) // Relanzar después de limpiar
		}
	}()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "GestorCuentas",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		// Coincide con el fondo oscuro de la nueva paleta (#1a1d23)
		BackgroundColour: &options.RGBA{R: 26, G: 29, B: 35, A: 1},
		Linux: &linux.Options{
			Icon: appIcon,
			ProgramName: "GestorCuentas",
		},
		OnStartup: app.startup,
		OnShutdown: app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
