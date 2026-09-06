package nui

import (
	"image"
	"image/color"
	"sync"

	"github.com/u00io/nui/nuicanvas"
)

type application struct {
	windows map[windowId]*nativeWindow
}

var app *application

func newApp() *application {
	return &application{
		windows: make(map[windowId]*nativeWindow),
	}
}

func init() {
	app = newApp()
}

func makeDefaultIcon() *image.RGBA {
	size := 32
	padding := 1
	rectSize := size/2 - padding*4

	x1 := padding
	y1 := padding
	x2 := padding + rectSize + padding + padding + padding + padding
	y2 := padding + rectSize + padding + padding + padding + padding

	icon := image.NewRGBA(image.Rect(0, 0, size, size))
	nuicanvas := nuicanvas.NewCanvas(icon)
	nuicanvas.SetColor(color.RGBA{0, 128, 255, 255})

	nuicanvas.FillRect(x1, y1, rectSize, rectSize, 1)
	nuicanvas.FillRect(x2, y1, rectSize, rectSize, 1)
	nuicanvas.FillRect(x1, y2, rectSize, rectSize, 1)
	nuicanvas.FillRect(x2, y2, rectSize, rectSize, 1)

	return icon
}

func CreateWindow(title string, posX int, posY int, width int, height int, center bool, maximized bool) Window {
	w := createWindow(title, posX, posY, width, height, center, maximized)
	w.SetAppIcon(makeDefaultIcon())
	return w
}

func CreateDefaultWindow() Window {
	w := CreateWindow("App", 100, 100, 800, 600, true, false)
	return w
}

// Run shows and runs each window's event loop concurrently, one goroutine per
// window, and blocks until all of them have been closed. Additional windows
// created later (e.g. from a callback) can still be launched manually with
// `go win.Exec()` — Run is just a convenience for the common case.
func Run(windows ...Window) {
	var wg sync.WaitGroup
	for _, w := range windows {
		wg.Add(1)
		go func(w Window) {
			defer wg.Done()
			w.Exec()
		}(w)
	}
	wg.Wait()
}
