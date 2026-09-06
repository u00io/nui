# nui documentation

Native GUI library for Go. Windows, keyboard, mouse, 2D canvas. No external Go dependencies (cgo + OS native APIs).

## Contents

- [getting-started.md](getting-started.md) — build, run, minimal app
- [window.md](window.md) — `Window` interface: creation, properties, events
- [multi-window.md](multi-window.md) — multiple windows, modal dialogs, goroutine rules
- [canvas.md](canvas.md) — 2D drawing API (`nuicanvas`)
- [keyboard.md](keyboard.md) — key codes (`nuikey`)
- [mouse.md](mouse.md) — mouse buttons/cursors (`nuimouse`)

## Packages

| Package | Import path | Purpose |
|---|---|---|
| `nui` | `github.com/u00io/nui/nui` | windows, event loop, app entry points |
| `nuicanvas` | `github.com/u00io/nui/nuicanvas` | draw into an `image.RGBA` (lines, rects, circles, text) |
| `nuikey` | `github.com/u00io/nui/nuikey` | `Key` constants, `KeyModifiers` |
| `nuimouse` | `github.com/u00io/nui/nuimouse` | `MouseButton`, `MouseCursor` constants |

## Supported OS

Linux (X11), Windows, macOS.
