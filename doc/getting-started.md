# Getting started

## Install

```
go get github.com/u00io/nui
```

## Build requirements

### Linux
No packages needed to build - the Linux backend talks to Xlib at runtime
via [purego](https://github.com/ebitengine/purego), not cgo, so
`GOOS=linux go build` also cross-compiles from macOS/Windows. `libX11.so`
must be present on whatever machine *runs* the binary (true of virtually
any Linux desktop, GNOME/KDE included, even under Wayland via XWayland).

### Windows
```
go build -ldflags="-H=windowsgui" .
```

### macOS
No extra packages needed (uses Cocoa via cgo).

## Minimal app

```go
package main

import "github.com/u00io/nui/nui"

func main() {
	nui.CreateDefaultWindow().Exec()
}
```

`Exec()` shows the window and blocks the calling goroutine, running the event loop until the window is closed.

## Window with drawing and input

```go
package main

import (
	"image"
	"image/color"

	"github.com/u00io/nui/nui"
	"github.com/u00io/nui/nuicanvas"
	"github.com/u00io/nui/nuikey"
)

func main() {
	win := nui.CreateWindow("My App", 100, 100, 800, 600, true, false)

	win.OnPaint(func(rgba *image.RGBA) {
		cnv := nuicanvas.NewCanvas(rgba)
		cnv.SetColor(color.RGBA{255, 255, 255, 255})
		cnv.DrawFixedString(10, 10, "Hello, nui!", 2)
	})

	win.OnKeyDown(func(key nuikey.Key, mods nuikey.KeyModifiers) bool {
		if key == nuikey.KeyEsc {
			win.Close()
		}
		return true
	})

	win.Exec()
}
```

- `OnPaint` is called whenever the window needs to redraw. Draw only inside it.
- Call `win.Update()` after changing any state that affects the picture, to request a repaint.
- Callback return values of `true` mean "event handled" (see [window.md](window.md)).

## Run examples in this repo

```
go run ./main.go
```

`main.go` runs `examples/example00_demo`. Edit it to point at another example package under `examples/`.
