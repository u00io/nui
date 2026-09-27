package nui

import "image"

// PopupWindow is a borderless window for tooltips and similar popups that
// may extend beyond the owner window. It never takes activation or keyboard
// focus, has no taskbar button, stays above its owner and lets the mouse
// pass through to the window below it.
type PopupWindow interface {
	OnPaint(func(rgba *image.RGBA))

	// ShowAt moves the popup to (x, y) in screen coordinates, resizes it and
	// shows it without activating it
	ShowAt(x, y, width, height int)
	Hide()
	Update()
	Close()
}

// CreatePopupWindow creates a hidden popup owned by owner. Returns nil when
// the platform doesn't support popups yet - callers should fall back to
// drawing inside the owner window.
func CreatePopupWindow(owner Window) PopupWindow {
	return createPopupWindow(owner)
}
