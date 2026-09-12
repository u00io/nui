package nui

import (
	"image"
	"image/color"

	"github.com/u00io/nui/nuikey"
	"github.com/u00io/nui/nuimouse"
)

type Window interface {
	// Change window
	Show()                   // Shows the window, non-modal; returns immediately
	ShowModal(parent Window) // Shows the window as a modal dialog owned by parent; returns immediately
	Update()                 // Updates the window content
	Close()                  // Closes the window
	Exec()                   // Waits for the window to close

	SystemHandle() any

	// Keyboard events
	OnKeyDown(func(keyCode nuikey.Key, mods nuikey.KeyModifiers) bool)
	OnKeyUp(func(keyCode nuikey.Key, mods nuikey.KeyModifiers))
	OnChar(func(char rune))

	// Mouse events
	OnMouseEnter(func())
	OnMouseLeave(func())
	OnMouseMove(func(x, y int))
	OnMouseButtonDown(func(btn nuimouse.MouseButton, x int, y int))
	OnMouseButtonUp(func(btn nuimouse.MouseButton, x int, y int))
	OnMouseButtonDblClick(func(btn nuimouse.MouseButton, x int, y int))
	OnMouseWheel(func(deltaX int, deltaY int))

	// Window events
	OnCreated(func())
	OnPaint(func(rgba *image.RGBA))
	OnMove(func(x, y int))
	OnResize(func(width, height int))
	OnCloseRequest(func() bool)
	OnTimer(func())

	// Window appearance
	SetTitle(title string)
	SetAppIcon(icon *image.RGBA)
	SetBackgroundColor(color color.RGBA)
	SetMouseCursor(cursor nuimouse.MouseCursor)

	Move(width int, height int)
	MoveToCenterOfScreen()
	Resize(width int, height int)
	MinimizeWindow()
	MaximizeWindow()
	IsMaximized() bool

	SetAllowMinimize(allow bool)
	SetAllowMaximize(allow bool)

	// Get window information
	Size() (width, height int)
	Pos() (x, y int)
	PosX() int
	PosY() int
	Width() int
	Height() int
	KeyModifiers() nuikey.KeyModifiers
	DrawTimeUs() int64
}
