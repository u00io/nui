package nui

// Darwin: callbacks from cocoa_darwin.go's ObjC classes plus Mac-specific
// input / size helpers. No cgo - callers are Go closures in the same binary.

import (
	"image"
	"image/color"
	"time"
	"unicode"
	"unsafe"

	"github.com/u00io/nui/nuikey"
	"github.com/u00io/nui/nuimouse"
)

func go_on_paint(hwnd windowId, ptr unsafe.Pointer, width int, height int) {
	// width/height come from drawRect's drawable rect in cocoa_darwin.go.
	img := &image.RGBA{
		Pix:    unsafe.Slice((*uint8)(ptr), width*height*4),
		Stride: width * 4,
		Rect:   image.Rect(0, 0, width, height),
	}

	if win, ok := hwnds[hwnd]; ok {
		win.windowPaint(img)
	}
}

func go_on_resize(hwnd windowId, width int, height int) {
	// width/height: NSWindow.contentLayoutRect (client area)
	if win, ok := hwnds[hwnd]; ok {
		win.windowResized(width, height)
	}
}

func go_on_close_request(hwnd windowId) bool {
	win, ok := hwnds[hwnd]
	if !ok || win.onCloseRequest == nil {
		return true
	}
	return win.onCloseRequest()
}

func go_on_window_will_close(hwnd windowId) {
	delete(hwnds, hwnd)
	if mainWindowIDSet && hwnd == mainWindowID {
		quitApp()
	}
}

func go_on_key_down(hwnd windowId, code int) {
	key := nuikey.Key(ConvertMacOSKeyToNuiKey(code))
	if win, ok := hwnds[hwnd]; ok {
		win.windowKeyDown(key)
	}
}

func go_on_key_up(hwnd windowId, code int) {
	key := nuikey.Key(ConvertMacOSKeyToNuiKey(code))
	if win, ok := hwnds[hwnd]; ok {
		win.windowKeyUp(key)
	}
}

func go_on_modifier_change(hwnd windowId, shift, ctrl, alt, cmd, caps, num, fnKey bool) {
	if win, ok := hwnds[hwnd]; ok {
		win.windowKeyModifiersChanged(shift, ctrl, alt, cmd, caps, num, fnKey)
	}
}

func go_on_char(hwnd windowId, codepoint int) {
	if win, ok := hwnds[hwnd]; ok {
		win.windowChar(rune(codepoint))
	}
}

func convertMacMouseButtons(button int) nuimouse.MouseButton {
	switch button {
	case 0:
		return nuimouse.MouseButtonLeft
	case 1:
		return nuimouse.MouseButtonRight
	case 2:
		return nuimouse.MouseButtonMiddle
	}
	return nuimouse.MouseButtonLeft
}

func go_on_window_move(hwnd windowId, x int, y int) {
	// x,Y: frame left and top-down Y from cocoa_darwin.go (matches Move()).
	if win, ok := hwnds[hwnd]; ok {
		win.windowMoved(x, y)
	}
}

func go_on_declare_draw_time(hwnd windowId, dt int) {
	if win, ok := hwnds[hwnd]; ok {
		win.windowDeclareDrawTime(dt)
	}
}

func go_on_mouse_down(hwnd windowId, button, x, y int) {
	if win, ok := hwnds[hwnd]; ok {
		if button >= 0 && button <= 2 {
			win.windowMouseButtonDown(convertMacMouseButtons(button), x, y)
		}
	}
}

func go_on_mouse_up(hwnd windowId, button, x, y int) {
	if win, ok := hwnds[hwnd]; ok {
		if button >= 0 && button <= 2 {
			win.windowMouseButtonUp(convertMacMouseButtons(button), x, y)
		}
	}
}

func go_on_mouse_move(hwnd windowId, x, y int) {
	if win, ok := hwnds[hwnd]; ok {
		win.windowMouseMove(x, y)
		win.macSetMouseCursor(win.currentCursor)
	}
}

func go_on_mouse_scroll(hwnd windowId, deltaX float64, deltaY float64) {
	if win, ok := hwnds[hwnd]; ok {
		win.windowMouseWheel(deltaX, deltaY)
	}
}

func go_on_mouse_enter(hwnd windowId) {
	if win, ok := hwnds[hwnd]; ok {
		win.windowMouseEnter()
	}
}

func go_on_mouse_leave(hwnd windowId) {
	if win, ok := hwnds[hwnd]; ok {
		win.windowMouseLeave()
	}
}

func go_on_mouse_double_click(hwnd windowId, button, x, y int) {
	if win, ok := hwnds[hwnd]; ok {
		if button >= 0 && button <= 2 {
			win.windowMouseButtonDblClick(convertMacMouseButtons(button), x, y)
		}
	}
}

func go_on_timer(hwnd windowId) {
	if win, ok := hwnds[hwnd]; ok {
		dtNow := time.Now()
		if dtNow.Sub(win.platform.lastTimerTick) < time.Millisecond*50 {
			return
		}
		win.platform.lastTimerTick = dtNow
		if win.onTimer != nil {
			win.onTimer()
		}
	}
}

// Main display frame in points (see cocoa_darwin.go getScreenWidth/Height).
func GetScreenSize() (width, height int) {
	return getScreenWidth(), getScreenHeight()
}

const maxCanvasWidth = 10000
const maxCanvasHeight = 5000

var canvasBufferBackground = make([]byte, maxCanvasWidth*maxCanvasHeight*4)
var canvasBufferBackgroundColor color.Color

// Filling the 200MB background buffer is expensive; every createWindow() call used to redo it
// with the same default color and stall the main thread. Skip it when the color didn't change.
func initCanvasBufferBackground(col color.Color) {
	if canvasBufferBackgroundColor == col {
		return
	}
	canvasBufferBackgroundColor = col

	r, g, b, a := col.RGBA()
	rb, gb, bb, ab := byte(b), byte(g), byte(r), byte(a)
	for i := 0; i < len(canvasBufferBackground); i += 4 {
		canvasBufferBackground[i+0] = rb
		canvasBufferBackground[i+1] = gb
		canvasBufferBackground[i+2] = bb
		canvasBufferBackground[i+3] = ab
	}
}

var macToPCScanCode = map[int]nuikey.Key{
	0x00: nuikey.KeyA,
	0x01: nuikey.KeyS,
	0x02: nuikey.KeyD,
	0x03: nuikey.KeyF,
	0x04: nuikey.KeyH,
	0x05: nuikey.KeyG,
	0x06: nuikey.KeyZ,
	0x07: nuikey.KeyX,
	0x08: nuikey.KeyC,
	0x09: nuikey.KeyV,
	0x0B: nuikey.KeyB,
	0x0C: nuikey.KeyQ,
	0x0D: nuikey.KeyW,
	0x0E: nuikey.KeyE,
	0x0F: nuikey.KeyR,
	0x10: nuikey.KeyY,
	0x11: nuikey.KeyT,
	0x12: nuikey.Key1,
	0x13: nuikey.Key2,
	0x14: nuikey.Key3,
	0x15: nuikey.Key4,
	0x16: nuikey.Key6,
	0x17: nuikey.Key5,
	0x18: nuikey.KeyEqual,
	0x19: nuikey.Key9,
	0x1A: nuikey.Key7,
	0x1B: nuikey.KeyMinus,
	0x1C: nuikey.Key8,
	0x1D: nuikey.Key0,
	0x1E: nuikey.KeyRightBracket,
	0x1F: nuikey.KeyO,
	0x20: nuikey.KeyU,
	0x21: nuikey.KeyLeftBracket,
	0x22: nuikey.KeyI,
	0x23: nuikey.KeyP,
	0x25: nuikey.KeyL,
	0x26: nuikey.KeyJ,
	0x27: nuikey.KeyApostrophe,
	0x28: nuikey.KeyK,
	0x29: nuikey.KeySemicolon,
	0x2A: nuikey.KeyBackslash,
	0x2B: nuikey.KeyComma,
	0x2C: nuikey.KeySlash,
	0x2D: nuikey.KeyN,
	0x2E: nuikey.KeyM,
	0x2F: nuikey.KeyDot,
	0x32: nuikey.KeyGrave,
	0x41: nuikey.KeyNumpadDot,
	0x43: nuikey.KeyNumpadAsterisk,
	0x45: nuikey.KeyNumpadPlus,
	//0x47: KeyNumpadClear,
	0x4B: nuikey.KeyNumpadSlash,
	0x4C: nuikey.KeyEnter,
	0x4E: nuikey.KeyNumpadMinus,
	//0x51: KeyNumpadEquals,
	0x52: nuikey.KeyNumpad0,
	0x53: nuikey.KeyNumpad1,
	0x54: nuikey.KeyNumpad2,
	0x55: nuikey.KeyNumpad3,
	0x56: nuikey.KeyNumpad4,
	0x57: nuikey.KeyNumpad5,
	0x58: nuikey.KeyNumpad6,
	0x59: nuikey.KeyNumpad7,
	0x5B: nuikey.KeyNumpad8,
	0x5C: nuikey.KeyNumpad9,
	0x24: nuikey.KeyEnter,
	0x30: nuikey.KeyTab,
	0x31: nuikey.KeySpace,
	0x33: nuikey.KeyBackspace,
	0x35: nuikey.KeyEsc,
	0x37: nuikey.KeyCommand,
	0x38: nuikey.KeyShift,
	0x39: nuikey.KeyCapsLock,
	0x3B: nuikey.KeyCtrl,
	0x3C: nuikey.KeyShift,
	0x3E: nuikey.KeyCtrl,
	0x3F: nuikey.KeyFunction,
	0x40: nuikey.KeyF17,
	0x4F: nuikey.KeyF18,
	0x50: nuikey.KeyF19,
	0x5A: nuikey.KeyF20,
	0x60: nuikey.KeyF5,
	0x61: nuikey.KeyF6,
	0x62: nuikey.KeyF7,
	0x63: nuikey.KeyF3,
	0x64: nuikey.KeyF8,
	0x65: nuikey.KeyF9,
	0x67: nuikey.KeyF11,
	0x69: nuikey.KeyF13,
	0x6A: nuikey.KeyF16,
	0x6B: nuikey.KeyF14,
	0x6D: nuikey.KeyF10,
	0x6F: nuikey.KeyF12,
	0x71: nuikey.KeyF15,
	0x73: nuikey.KeyHome,
	0x74: nuikey.KeyPageUp,
	0x75: nuikey.KeyDelete,
	0x76: nuikey.KeyF4,
	0x77: nuikey.KeyEnd,
	0x78: nuikey.KeyF2,
	0x79: nuikey.KeyPageDown,
	0x7A: nuikey.KeyF1,
	0x7B: nuikey.KeyArrowLeft,
	0x7C: nuikey.KeyArrowRight,
	0x7D: nuikey.KeyArrowDown,
	0x7E: nuikey.KeyArrowUp,
}

func ConvertMacOSKeyToNuiKey(macosKey int) nuikey.Key {
	if key, ok := macToPCScanCode[macosKey]; ok {
		return key
	}
	return nuikey.Key(0)
}

func (c *nativeWindow) startTimer(intervalMs float64) {
	startTimer(c.hwnd, intervalMs)
}

func (c *nativeWindow) stopTimer() {
	stopTimer(c.hwnd)
}

func (c *nativeWindow) windowMouseMove(x, y int) {
	if c.onMouseMove != nil {
		// NSView coords: origin bottom-left; nui uses top-left Y.
		_, areaH := c.requestClientAreaSize()
		y = areaH - y
		c.onMouseMove(x, y)
	}
	c.Update()
}

func (c *nativeWindow) windowResized(width, height int) {
	// Client-area width/height from window.m (contentLayoutRect).
	c.windowWidth = width
	c.windowHeight = height
	if c.onResize != nil {
		c.onResize(width, height)
	}
}

func (c *nativeWindow) windowMouseWheel(deltaX, deltaY float64) {
	deltaXInt := 0
	if deltaX > 0.2 {
		deltaXInt = 1
	}
	if deltaX < -0.2 {
		deltaXInt = -1
	}

	deltaYInt := 0
	if deltaY > 0.2 {
		deltaYInt = 1
	}
	if deltaY < -0.2 {
		deltaYInt = -1
	}

	if c.onMouseWheel != nil {
		c.onMouseWheel(deltaXInt, deltaYInt)
	}
}

func (c *nativeWindow) windowMouseEnter() {
	if c.onMouseEnter != nil {
		c.onMouseEnter()
	}
	c.macSetMouseCursor(c.currentCursor)
}

func (c *nativeWindow) windowMouseLeave() {
	if c.onMouseLeave != nil {
		c.onMouseLeave()
	}
	c.macSetMouseCursor(nuimouse.MouseCursorArrow)
}

// key modifiers
func (c *nativeWindow) windowKeyModifiersChanged(shift bool, ctrl bool, alt bool, cmd bool, caps bool, num bool, _ bool) {
	// Key shift
	if c.platform.keyModifiers.Shift && !shift {
		c.windowKeyUp(nuikey.KeyShift)
	}
	if !c.platform.keyModifiers.Shift && shift {
		c.windowKeyDown(nuikey.KeyShift)
	}
	c.platform.keyModifiers.Shift = shift

	// Key ctrl
	if c.platform.keyModifiers.Ctrl && !ctrl {
		c.windowKeyUp(nuikey.KeyCtrl)
	}
	if !c.platform.keyModifiers.Ctrl && ctrl {
		c.windowKeyDown(nuikey.KeyCtrl)
	}
	c.platform.keyModifiers.Ctrl = ctrl

	// Key alt
	if c.platform.keyModifiers.Alt && !alt {
		c.windowKeyUp(nuikey.KeyAlt)
	}
	if !c.platform.keyModifiers.Alt && alt {
		c.windowKeyDown(nuikey.KeyAlt)
	}
	c.platform.keyModifiers.Alt = alt

	// Key cmd
	if c.platform.keyModifiers.Cmd && !cmd {
		c.windowKeyUp(nuikey.KeyCommand)
	}
	if !c.platform.keyModifiers.Cmd && cmd {
		c.windowKeyDown(nuikey.KeyCommand)
	}
	c.platform.keyModifiers.Cmd = cmd

	if caps != c.platform.lastCapsLockState {
		if caps {
			c.windowKeyDown(nuikey.KeyCapsLock)
		} else {
			c.windowKeyDown(nuikey.KeyCapsLock)
		}
		c.platform.lastCapsLockState = caps
	}

	if num != c.platform.lastNumLockState {
		if num {
			c.windowKeyDown(nuikey.KeyNumLock)
		} else {
			c.windowKeyDown(nuikey.KeyNumLock)
		}
		c.platform.lastNumLockState = num
	}
}

func (c *nativeWindow) windowKeyDown(keyCode nuikey.Key) {
	if c.onKeyDown != nil {
		c.onKeyDown(keyCode, c.platform.keyModifiers)
	}
}

func (c *nativeWindow) windowKeyUp(keyCode nuikey.Key) {
	if c.onKeyUp != nil {
		keyModifiers := c.platform.keyModifiers
		if keyCode == nuikey.KeyShift {
			keyModifiers.Shift = false
		}
		if keyCode == nuikey.KeyCtrl {
			keyModifiers.Ctrl = false
		}
		if keyCode == nuikey.KeyAlt {
			keyModifiers.Alt = false
		}
		if keyCode == nuikey.KeyCommand {
			keyModifiers.Cmd = false
		}
		c.onKeyUp(keyCode, keyModifiers)
	}
}

func (c *nativeWindow) windowDeclareDrawTime(dt int) {
	c.drawTimes[c.drawTimesIndex] = int64(dt)
	c.drawTimesIndex++
	if c.drawTimesIndex >= len(c.drawTimes) {
		c.drawTimesIndex = 0
	}
}

func (c *nativeWindow) windowPaint(rgba *image.RGBA) {

	imgDataSize := rgba.Rect.Dx() * rgba.Rect.Dy() * 4
	copy(rgba.Pix[:imgDataSize], canvasBufferBackground)

	if c.onPaint != nil {
		c.onPaint(rgba)
	}
}

func (c *nativeWindow) windowChar(char rune) {
	if !unicode.IsPrint(char) {
		return
	}

	if c.onChar != nil {
		c.onChar(char)
	}
}

func (c *nativeWindow) windowMouseButtonDown(button nuimouse.MouseButton, x, y int) {
	if c.onMouseButtonDown != nil {
		// Flip Y: see windowMouseMove.
		_, areaH := c.requestClientAreaSize()
		y = areaH - y
		c.onMouseButtonDown(button, x, y)
	}
	c.macSetMouseCursor(c.currentCursor)
}

func (c *nativeWindow) windowMouseButtonUp(button nuimouse.MouseButton, x, y int) {
	if c.onMouseButtonUp != nil {
		// Flip Y: see windowMouseMove.
		_, areaH := c.requestClientAreaSize()
		y = areaH - y
		c.onMouseButtonUp(button, x, y)
	}
	c.macSetMouseCursor(c.currentCursor)
}

func (c *nativeWindow) windowMouseButtonDblClick(button nuimouse.MouseButton, x, y int) {
	if c.onMouseButtonDblClick != nil {
		// Flip Y: see windowMouseMove.
		_, areaH := c.requestClientAreaSize()
		y = areaH - y
		c.onMouseButtonDblClick(button, x, y)
	}
	c.macSetMouseCursor(c.currentCursor)
}

func (c *nativeWindow) windowMoved(x, y int) {
	c.windowPosX = x
	c.windowPosY = y
	if c.onMove != nil {
		c.onMove(x, y)
	}
}

// Frame position: X is Cocoa left edge; Y is top-down distance to window top (cocoa_darwin.go helpers).
func (c *nativeWindow) requestWindowPosition() (int, int) {
	return getWindowPositionX(c.hwnd), getWindowPositionY(c.hwnd)
}

// Client-area size via contentLayoutRect (same getters as getWindowWidth/Height in cocoa_darwin.go).
func (c *nativeWindow) requestWindowSize() (int, int) {
	return getWindowWidth(c.hwnd), getWindowHeight(c.hwnd)
}

// On Darwin, identical dimensions to requestWindowSize (thin bridge to ObjC symmetry).
func (c *nativeWindow) requestClientAreaSize() (int, int) {
	return getClientAreaWidth(c.hwnd), getClientAreaHeight(c.hwnd)
}
