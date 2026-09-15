# nui - Native UI Library

Native gateway between OS UI & Golang with minimum dependencies.

- Windows management
- Keyboard input
- Mouse input

Documentation: [doc/README.md](doc/README.md)

# Operating Systems
- Linux
- Windows
- MacOS

# Linux build
- go build -o bin/nui ./main.go

No C compiler or X11 headers needed to build (the Linux backend talks to
Xlib at runtime via [purego](https://github.com/ebitengine/purego), not
cgo), so this also cross-compiles from macOS/Windows with a plain
`GOOS=linux go build`. `libX11.so` still has to be present on whatever
machine actually runs the binary - true of virtually any Linux desktop
(GNOME/KDE, even under Wayland via XWayland).

# Windows build
- go build -o bin/nui.exe -ldflags="-H=windowsgui"
