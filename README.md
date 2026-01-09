# nui - Native UI Library

Native gateway between OS UI & Golang with minimum dependencies.

- Windows management
- Keyboard input
- Mouse input

# Operating Systems
- Linux
- Windows
- MacOS

# Linux build
- export CGO_ENABLED=1
- sudo apt install gcc
- sudo apt install -y libx11-dev
- go build -o bin/nui ./main.go

# Windows build
- go build -o bin/nui.exe -ldflags="-H=windowsgui"
