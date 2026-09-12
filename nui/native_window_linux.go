//go:build linux
// +build linux

package nui

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
	"unsafe"

	"github.com/u00io/nui/nuikey"
	"github.com/u00io/nui/nuimouse"
)

/*
#cgo LDFLAGS: -lX11
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/Xatom.h>
#include <stdlib.h>
#include <string.h>
#include "ximage_helper.h"
#include <locale.h>
*/
import "C"

func init() {
	C.setlocale(C.LC_ALL, C.CString(""))
	// Required before any Xlib call once multiple windows run their event loops on separate goroutines.
	C.XInitThreads()
}

type windowId C.Window

type nativeWindowPlatform struct {
	display *C.Display
	window  C.Window
	screen  C.int

	closed bool
	// Set by Close() from any goroutine; only the owning Exec goroutine
	// acts on it and actually tears down the Display, avoiding a use-after-free
	// if Close() is called from another window's goroutine.
	closeRequested int32

	lastMouseDownX      int
	lastMouseDownY      int
	lastMouseDownButton nuimouse.MouseButton
	lastMouseDownTime   time.Time

	dtLastUpdateCalled time.Time
	needUpdateInTimer  bool

	wmProtocols    C.Atom
	wmDeleteWindow C.Atom

	prevSetPosX int
	prevSetPosY int

	netWMState              C.Atom
	netWMStateMaximizedHorz C.Atom
	netWMStateMaximizedVert C.Atom

	// Per-window paint surface, sized to the window's current dimensions.
	canvasBuffer []byte
	bgColor      color.RGBA

	// setWindowDecorations() sets the minimize/maximize _MOTIF_WM_HINTS bits
	// together as one property, so SetAllowMinimize/SetAllowMaximize each
	// need the other's last-set value on hand to avoid clobbering it.
	allowMinimize bool
	allowMaximize bool

	// Show() maps the window and spawns pumpEvents() on its own goroutine
	// exactly once (pumpStarted guards that), so callers never have to know
	// this needs its own goroutine at all - Exec() just waits on
	// pumpDone. Xlib itself has no thread-affinity requirement like Win32's
	// (XInitThreads(), called at package init, is enough to let another
	// goroutine safely drive this Display later), so unlike the Windows
	// backend this doesn't need a dedicated locked OS thread from creation.
	pumpStarted bool
	pumpDone    chan struct{}

	// modalChildCount counts this window's currently-open ShowModal children.
	// While > 0, pumpEvents drops keyboard/mouse callbacks for this window:
	// kwin_x11 (and apparently other Linux WMs) accepts _NET_WM_STATE_MODAL
	// and withholds keyboard focus from the parent, but still happily
	// delivers pointer button/motion events straight to it, so nui has to
	// enforce the block itself. Accessed with atomics since ShowModal/doClose
	// can run on a different goroutine than this window's own pump loop.
	modalChildCount int32

	// modalParent is the window this dialog was shown modally over (set by
	// ShowModal), kept so doClose can decrement its modalChildCount and
	// un-block it again once this dialog closes.
	modalParent *nativeWindow
}

type rect struct {
	left, top, right, bottom int32
}

func loadPngFromBytes(bs []byte) (*image.RGBA, error) {
	img, err := png.Decode(bytes.NewReader(bs))
	if err != nil {
		return nil, err
	}

	rgba := image.NewRGBA(img.Bounds())
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}

	return rgba, nil
}

var hwnds map[windowId]*nativeWindow
var hwndsMu sync.Mutex

func init() {
	hwnds = make(map[windowId]*nativeWindow)
}

func GetNativeWindowByHandle(hwnd C.Window) *nativeWindow {
	hwndsMu.Lock()
	defer hwndsMu.Unlock()
	if w, ok := hwnds[windowId(hwnd)]; ok {
		return w
	}
	return nil
}

func getHDCSize(hdc uintptr) (width int32, height int32) {
	var r rect
	return r.right - r.left, r.bottom - r.top
}

// Sanity caps on a single window's paintable area, not a shared buffer size.
const maxCanvasWidth = 10000
const maxCanvasHeight = 5000

// ensureCanvasBuffer grows this window's own paint buffer to fit size bytes, if needed.
func (c *nativeWindow) ensureCanvasBuffer(size int) []byte {
	if cap(c.platform.canvasBuffer) < size {
		c.platform.canvasBuffer = make([]byte, size)
	} else {
		c.platform.canvasBuffer = c.platform.canvasBuffer[:size]
	}
	return c.platform.canvasBuffer
}

// fillCanvasBuffer paints buf with this window's solid background color.
func fillCanvasBuffer(buf []byte, col color.RGBA) {
	r, g, b, a := col.B, col.G, col.R, col.A
	for i := 0; i+3 < len(buf); i += 4 {
		buf[i+0] = r
		buf[i+1] = g
		buf[i+2] = b
		buf[i+3] = a
	}
}

///////////////////////////////////////////////////////////////////

func createWindow(title string, posX int, posY int, width int, height int, center bool, maximized bool) *nativeWindow {
	var c nativeWindow
	c.showMaximized = maximized
	c.platform.bgColor = color.RGBA{0, 50, 0, 255}
	c.platform.allowMinimize = true
	c.platform.allowMaximize = true
	c.platform.prevSetPosX = -1
	c.platform.prevSetPosY = -1

	c.platform.display = C.XOpenDisplay(nil)
	if c.platform.display == nil {
		panic("Unable to open X display")
	}
	//defer C.XCloseDisplay(c.display)

	c.platform.screen = C.XDefaultScreen(c.platform.display)

	attrs := C.XSetWindowAttributes{}
	attrs.background_pixmap = C.None

	mask := C.CWBackPixmap

	c.platform.window = C.XCreateWindow(
		c.platform.display,
		C.XRootWindow(c.platform.display, c.platform.screen),
		100, 100, // x, y
		C.uint(width), C.uint(height), // width, height
		1,                // border width
		C.CopyFromParent, // depth
		C.InputOutput,    // class
		nil,              // visual
		C.ulong(mask),    // valuemask
		&attrs,           // attributes pointer (не значение!)
	)

	C.XSelectInput(c.platform.display, c.platform.window, C.ExposureMask|C.PropertyChangeMask|C.StructureNotifyMask|C.KeyPressMask|C.KeyReleaseMask|C.EnterWindowMask|C.LeaveWindowMask|C.ButtonPressMask|C.ButtonReleaseMask|C.PointerMotionMask)

	var getAttr C.XWindowAttributes
	C.XGetWindowAttributes(c.platform.display, c.platform.window, &getAttr)
	c.windowWidth, c.windowHeight = int(getAttr.width), int(getAttr.height)

	// Store the window handle
	hwndsMu.Lock()
	hwnds[windowId(c.platform.window)] = &c
	hwndsMu.Unlock()

	// Set default icon
	icon := image.NewRGBA(image.Rect(0, 0, 32, 32))
	c.SetAppIcon(icon)

	c.SetTitle(title)

	c.initCloseProtocol()
	c.initWindowStateAtoms()

	return &c
}

func (c *nativeWindow) initCloseProtocol() {
	display := (*C.Display)(c.platform.display)
	window := C.Window(c.platform.window)

	nameProtocols := C.CString("WM_PROTOCOLS")
	nameDelete := C.CString("WM_DELETE_WINDOW")
	defer C.free(unsafe.Pointer(nameProtocols))
	defer C.free(unsafe.Pointer(nameDelete))

	c.platform.wmProtocols = C.XInternAtom(display, nameProtocols, C.False)
	c.platform.wmDeleteWindow = C.XInternAtom(display, nameDelete, C.False)

	C.XSetWMProtocols(display, window, &c.platform.wmDeleteWindow, 1)
}

func (c *nativeWindow) initWindowStateAtoms() {
	display := c.platform.display

	nameState := C.CString("_NET_WM_STATE")
	nameMaxH := C.CString("_NET_WM_STATE_MAXIMIZED_HORZ")
	nameMaxV := C.CString("_NET_WM_STATE_MAXIMIZED_VERT")
	defer C.free(unsafe.Pointer(nameState))
	defer C.free(unsafe.Pointer(nameMaxH))
	defer C.free(unsafe.Pointer(nameMaxV))

	c.platform.netWMState = C.XInternAtom(display, nameState, C.False)
	c.platform.netWMStateMaximizedHorz = C.XInternAtom(display, nameMaxH, C.False)
	c.platform.netWMStateMaximizedVert = C.XInternAtom(display, nameMaxV, C.False)
}

// Show maps the window and, the first time it's called, starts this
// window's own event pump on a new goroutine - callers never need to spawn
// one themselves (compare ShowModal, which has always hidden this the same
// way). Safe to call more than once; only the first call does anything.
func (c *nativeWindow) Show() {
	if c.platform.pumpStarted {
		return
	}
	c.platform.pumpStarted = true
	c.platform.pumpDone = make(chan struct{})

	C.XMapWindow(c.platform.display, c.platform.window)
	C.XFlush(c.platform.display)

	go func() {
		c.pumpEvents()
		close(c.platform.pumpDone)
	}()
}

func (c *nativeWindow) Hide() {
}

func (c *nativeWindow) Update() {
	if time.Since(c.platform.dtLastUpdateCalled) < 40*time.Millisecond {
		c.platform.needUpdateInTimer = true
		return
	}
	c.platform.dtLastUpdateCalled = time.Now()

	C.XClearArea(
		c.platform.display,
		c.platform.window,
		0, 0,
		0, 0,
		1, // last parameter is `exposures`: if True — generate Expose event
	)
	C.XFlush(c.platform.display)
	//C.XClearWindow(c.display, c.window)
}

func eventType(event C.XEvent) int {
	return int(*(*C.int)(unsafe.Pointer(&event)))
}

/*var posX C.uint
var posY C.uint
var width C.uint
var height C.uint*/

// Exec blocks the calling goroutine until this window closes. The actual
// X11 event pump runs on the goroutine Show() started (idempotent, so
// calling it here covers a bare Exec() call with no prior Show()).
func (c *nativeWindow) Exec() {
	c.Show()
	<-c.platform.pumpDone
}

// pumpEvents is the real XPending/XNextEvent loop, run on the goroutine
// Show() spawns for the life of the window.
func (c *nativeWindow) pumpEvents() {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	dtLastPaint := time.Now()

	for !c.platform.closed {
		if atomic.LoadInt32(&c.platform.closeRequested) != 0 {
			c.doClose()
			break
		}

		for !c.platform.closed && C.XPending(c.platform.display) > 0 {
			var event C.XEvent
			C.XNextEvent(c.platform.display, &event)

			if c.platform.closed {
				break
			}

			{
				_, _, _, _, ok := c.getFrameExtents()
				if ok {
					if c.platform.prevSetPosX >= 0 && c.platform.prevSetPosY >= 0 {
						c.Move(c.platform.prevSetPosX, c.platform.prevSetPosY)
						c.platform.prevSetPosX = -1
						c.platform.prevSetPosY = -1
					}
				}
			}

			// A modal dialog owned by this window is open: kwin_x11 (and
			// apparently other Linux WMs) withholds keyboard focus from us
			// but still delivers pointer/keyboard events straight to this
			// window, so drop them here ourselves instead of dispatching to
			// app callbacks. Once the dialog closes, doClose() drops
			// modalChildCount back to 0 and these events flow again.
			if c.inputBlocked() {
				switch eventType(event) {
				case C.KeyPress, C.KeyRelease, C.ButtonPress, C.ButtonRelease, C.MotionNotify, C.EnterNotify, C.LeaveNotify:
					continue
				}
			}

			switch eventType(event) {

			case C.Expose:
				{
					{
						dtBeginPaint := time.Now()
						dtLastPaint = time.Now()
						hdcWidth, hdcHeight := c.windowWidth, c.windowHeight
						if hdcWidth > maxCanvasWidth {
							hdcWidth = maxCanvasWidth
						}

						if hdcHeight > maxCanvasHeight {
							hdcHeight = maxCanvasHeight
						}

						canvasDataBufferSize := int(hdcWidth * hdcHeight * 4)
						buf := c.ensureCanvasBuffer(canvasDataBufferSize)
						fillCanvasBuffer(buf, c.platform.bgColor)

						img := &image.RGBA{
							Pix:    buf,
							Stride: int(hdcWidth) * 4,
							Rect:   image.Rect(0, 0, int(hdcWidth), int(hdcHeight)),
						}

						if c.onPaint != nil {
							c.onPaint(img)
						}

						c.drawImageRGBA(c.platform.display, c.platform.window, img)
						paintTime := time.Since(dtLastPaint)
						_ = paintTime
						//fmt.Println("PaintTime:", paintTime.Microseconds())

						c.drawTimes[c.drawTimesIndex] = time.Since(dtBeginPaint).Microseconds()
						c.drawTimesIndex++
						if c.drawTimesIndex >= len(c.drawTimes) {
							c.drawTimesIndex = 0
						}

					}

				}
			case C.MapNotify:
				mapEvent := (*C.XMapEvent)(unsafe.Pointer(&event))
				fmt.Printf("Window became visible. Window ID: %d\n", mapEvent.window)

			case C.UnmapNotify:
				unmapEvent := (*C.XUnmapEvent)(unsafe.Pointer(&event))
				fmt.Printf("Window was hidden. Window ID: %d\n", unmapEvent.window)

				// The WM just iconified us (titlebar button, window menu,
				// keyboard shortcut - ICCCM has the client unmap itself to go
				// Iconic regardless of which one triggered it), but
				// SetAllowMinimize(false) says this window shouldn't be
				// minimizable. Since kwin's decoration doesn't reliably honor
				// the _MOTIF_WM_HINTS decorations bits for button visibility
				// on every theme, undo the effect directly: re-map right
				// away instead of trying to prevent the click itself.
				if !c.platform.allowMinimize && !c.platform.closed && atomic.LoadInt32(&c.platform.closeRequested) == 0 {
					C.XMapWindow(c.platform.display, c.platform.window)
					C.XFlush(c.platform.display)
				}

			case C.DestroyNotify:
				destroyEvent := (*C.XDestroyWindowEvent)(unsafe.Pointer(&event))
				fmt.Printf("Window was destroyed. Window ID: %d\n", destroyEvent.window)

			case C.ReparentNotify:
				reparentEvent := (*C.XReparentEvent)(unsafe.Pointer(&event))
				fmt.Printf("Window changed parent. Window ID: %d, New Parent ID: %d\n", reparentEvent.window, reparentEvent.parent)
			case C.ResizeRequest:
				resizeEvent := (*C.XResizeRequestEvent)(unsafe.Pointer(&event))
				fmt.Printf("Resize request received: Width=%d, Height=%d\n", resizeEvent.width, resizeEvent.height)

				c.windowWidth = int(resizeEvent.width)
				c.windowHeight = int(resizeEvent.height)

				//c.Update()

			case C.ConfigureNotify:
				configureEvent := (*C.XConfigureEvent)(unsafe.Pointer(&event))

				prevWindowPosX := c.windowPosX
				prevWindowPosY := c.windowPosY

				c.updateWindowPos()

				if configureEvent.send_event == 1 && (c.windowPosX != prevWindowPosX || c.windowPosY != prevWindowPosY) {
					if c.onMove != nil {
						c.onMove(c.windowPosX, c.windowPosY)
					}
				}

				if configureEvent.send_event == 0 && (c.windowWidth != int(configureEvent.width) || c.windowHeight != int(configureEvent.height)) {
					c.windowWidth = int(configureEvent.width)
					c.windowHeight = int(configureEvent.height)
					if c.onResize != nil {
						c.onResize(c.windowWidth, c.windowHeight)
					}
				}

				c.Update()

			case C.KeyPress:
				keyEvent := (*C.XKeyEvent)(unsafe.Pointer(&event))
				keySym := C.XLookupKeysym((*C.XKeyEvent)(unsafe.Pointer(&event)), 0)
				fmt.Printf("Key pressed: KeySym = %d, KeyCode = 0x%x\n", keySym, keyEvent.keycode)
				key := ConvertLinuxKeyToNuiKey(int(keyEvent.keycode))
				processed := false
				if c.onKeyDown != nil {
					processed = c.onKeyDown(key, c.getModifierState())
				}

				if processed {
					break
				}

				if c.platform.closed {
					break
				}

				var buf [32]C.char
				var sym C.KeySym

				xkey := (*C.XKeyEvent)(unsafe.Pointer(&event))

				n := C.XLookupString(
					xkey,
					&buf[0],
					C.int(len(buf)),
					&sym,
					nil,
				)

				if n > 0 {
					text := C.GoStringN(&buf[0], n)
					fmt.Printf("Text input: %s\n", text)

					firstRune, _ := utf8.DecodeRuneInString(text)
					if firstRune > 0 && firstRune != 127 {
						if c.onChar != nil {
							c.onChar(firstRune)
						}
					}
				}

			case C.KeyRelease:
				keyEvent := (*C.XKeyEvent)(unsafe.Pointer(&event))
				keySym := C.XLookupKeysym(keyEvent, 0)
				fmt.Printf("Key released: KeySym = %d, KeyCode = 0x%x\n", keySym, keyEvent.keycode)
				key := ConvertLinuxKeyToNuiKey(int(keyEvent.keycode))
				if c.onKeyUp != nil {
					c.onKeyUp(key, c.getModifierState())
				}

			case C.EnterNotify:
				if c.onMouseEnter != nil {
					c.onMouseEnter()
				}

			case C.LeaveNotify:
				if c.onMouseLeave != nil {
					c.onMouseLeave()
				}

			case C.MotionNotify:
				motionEvent := (*C.XMotionEvent)(unsafe.Pointer(&event))
				if c.onMouseMove != nil {
					c.onMouseMove(int(motionEvent.x), int(motionEvent.y))
				}

			case C.ButtonPress:
				buttonEvent := (*C.XButtonEvent)(unsafe.Pointer(&event))
				//fmt.Printf("Mouse button %d pressed at (%d, %d)\n", buttonEvent.button, buttonEvent.x, buttonEvent.y)

				x := int(buttonEvent.x)
				y := int(buttonEvent.y)

				switch buttonEvent.button {
				case 1:
					if c.onMouseButtonDown != nil {
						c.onMouseButtonDown(nuimouse.MouseButtonLeft, x, y)
					}
				case 2:
					if c.onMouseButtonDown != nil {
						c.onMouseButtonDown(nuimouse.MouseButtonMiddle, x, y)
					}
				case 3:
					if c.onMouseButtonDown != nil {
						c.onMouseButtonDown(nuimouse.MouseButtonRight, x, y)
					}
				case 4:
					if c.onMouseWheel != nil {
						c.onMouseWheel(0, 1)
					}
				case 5:
					if c.onMouseWheel != nil {
						c.onMouseWheel(0, -1)
					}
				case 6:
					if c.onMouseWheel != nil {
						c.onMouseWheel(1, 0)
					}
				case 7:
					if c.onMouseWheel != nil {
						c.onMouseWheel(-1, 0)
					}
				}

				dblClickDetected := false
				// Double click detection
				if buttonEvent.button == 1 || buttonEvent.button == 2 || buttonEvent.button == 3 {
					if c.lastMouseButton == nuimouse.MouseButton(buttonEvent.button) {
						timeSinceLastClick := time.Since(c.lastMouseDownTime)
						distanceX := int(buttonEvent.x) - c.lastMouseDownX
						distanceY := int(buttonEvent.y) - c.lastMouseDownY
						distanceSquared := distanceX*distanceX + distanceY*distanceY
						if timeSinceLastClick < 500*time.Millisecond && distanceSquared < 25 {
							// Detected double click
							if c.onMouseButtonDblClick != nil {
								var btn nuimouse.MouseButton
								switch buttonEvent.button {
								case 1:
									btn = nuimouse.MouseButtonLeft
								case 2:
									btn = nuimouse.MouseButtonMiddle
								case 3:
									btn = nuimouse.MouseButtonRight
								}
								c.onMouseButtonDblClick(btn, x, y)
							}
							// fmt.Println("dbl click detected")
							dblClickDetected = true
						}
					}
				}

				if !dblClickDetected {
					// Update last mouse down info
					c.lastMouseDownX = int(buttonEvent.x)
					c.lastMouseDownY = int(buttonEvent.y)
					c.lastMouseButton = nuimouse.MouseButton(buttonEvent.button)
					c.lastMouseDownTime = time.Now()
				} else {
					// Reset last mouse down info to avoid triple click detection
					c.lastMouseDownX = 0
					c.lastMouseDownY = 0
					c.lastMouseButton = nuimouse.MouseButton(0)
					c.lastMouseDownTime = time.Time{}
				}

			case C.ButtonRelease:
				buttonEvent := (*C.XButtonEvent)(unsafe.Pointer(&event))
				//fmt.Printf("Mouse button %d released at (%d, %d)\n", buttonEvent.button, buttonEvent.x, buttonEvent.y)

				x := int(buttonEvent.x)
				y := int(buttonEvent.y)

				switch buttonEvent.button {
				case 1:
					if c.onMouseButtonUp != nil {
						c.onMouseButtonUp(nuimouse.MouseButtonLeft, x, y)
					}
				case 2:
					if c.onMouseButtonUp != nil {
						c.onMouseButtonUp(nuimouse.MouseButtonMiddle, x, y)
					}
				case 3:
					if c.onMouseButtonUp != nil {
						c.onMouseButtonUp(nuimouse.MouseButtonRight, x, y)
					}
				}

			case C.ClientMessage:
				xclient := (*C.XClientMessageEvent)(unsafe.Pointer(&event))
				data0 := *(*C.long)(unsafe.Pointer(&xclient.data[0]))

				if xclient.message_type == c.platform.wmProtocols &&
					C.Atom(data0) == c.platform.wmDeleteWindow {

					if c.inputBlocked() {
						// A modal dialog owned by this window is open - refuse
						// the close request outright, same as native modal
						// dialogs do, without even asking onCloseRequest.
						break
					}

					allowClose := true
					if c.onCloseRequest != nil {
						allowClose = c.onCloseRequest()
					}

					if allowClose {
						// Close (not a bare XDestroyWindow) so the request is
						// actually flushed before this event loop stops pumping.
						c.Close()
					}
				}

			case C.PropertyNotify:
				propEvent := (*C.XPropertyEvent)(unsafe.Pointer(&event))
				if propEvent.atom == c.platform.netWMState && !c.platform.allowMaximize && c.IsMaximized() {
					// Same idea as the UnmapNotify case above: the WM just
					// maximized us (button, window menu, double-click on the
					// titlebar, drag-to-edge, ...) despite
					// SetAllowMaximize(false), so ask it to un-maximize
					// again right away rather than relying on it to have
					// refused the click in the first place.
					C.restoreWindow(c.platform.display, c.platform.window)
				}
			}

		}

		if !c.platform.closed {
			select {
			case <-ticker.C:
				{
					//fmt.Println("Timer event: 10ms tick")
					if c.platform.needUpdateInTimer {
						c.Update()
						c.platform.needUpdateInTimer = false
					}
					if c.onTimer != nil {
						c.onTimer()
						c.Update()
					}
				}
			default:
			}
		}
	}
}

// Close requests the window to close and is safe to call from any goroutine
// (e.g. a parent window closing a dialog it owns). The actual XDestroyWindow/
// XCloseDisplay teardown always runs on the window's own Exec goroutine
// (see doClose), since closing the Display while that goroutine might still
// be mid-call on it (XPending/XNextEvent) would be a use-after-free.
//
// Refuses to close a window that still has a modal dialog open on top of
// it (inputBlocked), the same as the WM_DELETE_WINDOW handler in
// pumpEvents - covers Close() being called directly (e.g. from a menu
// action) rather than only via the titlebar close button.
func (c *nativeWindow) Close() {
	if c.inputBlocked() {
		return
	}
	atomic.StoreInt32(&c.platform.closeRequested, 1)
}

// doClose performs the actual Xlib teardown. Must only be called from the
// window's own Exec goroutine.
func (c *nativeWindow) doClose() {
	C.XDestroyWindow(c.platform.display, c.platform.window)
	C.XCloseDisplay(c.platform.display)
	c.platform.closed = true

	if c.platform.modalParent != nil {
		atomic.AddInt32(&c.platform.modalParent.platform.modalChildCount, -1)
		c.platform.modalParent = nil
	}

	hwndsMu.Lock()
	delete(hwnds, windowId(c.platform.window))
	hwndsMu.Unlock()
}

func (c *nativeWindow) SetTitle(title string) {
	cstr := C.CString(title)
	defer C.free(unsafe.Pointer(cstr))
	C.XStoreName(c.platform.display, c.platform.window, cstr)
}

func (c *nativeWindow) Move(x, y int) {
	c.platform.prevSetPosX = x
	c.platform.prevSetPosY = y
	left, _, top, _, ok := c.getFrameExtents()
	if ok {
		x -= left
		y -= top
	}
	//fmt.Println("LINUX MOVE to:", c.platform.prevSetPosX, c.platform.prevSetPosY)
	//fmt.Println("LINUX MOVE top:", top)

	C.XMoveWindow(c.platform.display, c.platform.window, C.int(x), C.int(y))
}

func getScreenSize() (width, height int) {
	display := C.XOpenDisplay(nil)
	screen := C.XDefaultScreen(display)
	width = int(C.XDisplayWidth(display, screen))
	height = int(C.XDisplayHeight(display, screen))
	C.XCloseDisplay(display)
	return
}

func (c *nativeWindow) MoveToCenterOfScreen() {
	screenWidth, screenHeight := getScreenSize()
	windowWidth, windowHeight := c.Size()
	x := (screenWidth - windowWidth) / 2
	y := (screenHeight - windowHeight) / 2
	c.Move(int(x), int(y))
}

func (c *nativeWindow) Resize(width, height int) {
	C.XResizeWindow(c.platform.display, c.platform.window, C.uint(width), C.uint(height))
}

func (c *nativeWindow) PosX() int {
	return c.windowPosX
}

func (c *nativeWindow) PosY() int {
	return c.windowPosY
}

func (c *nativeWindow) Pos() (x, y int) {
	return c.windowPosX, c.windowPosY
}

func (c *nativeWindow) Size() (width, height int) {
	return c.windowWidth, c.windowHeight
}

func (c *nativeWindow) Width() int {
	return c.windowWidth
}

func (c *nativeWindow) Height() int {
	return c.windowHeight
}

func (c *nativeWindow) IsMaximized() bool {
	display := c.platform.display
	window := c.platform.window

	var actualType C.Atom
	var actualFormat C.int
	var nitems C.ulong
	var bytesAfter C.ulong
	var prop *C.uchar

	if c.platform.closed {
		return false
	}

	status := C.XGetWindowProperty(
		display,
		window,
		c.platform.netWMState,
		0,
		1024,
		C.False,
		C.XA_ATOM,
		&actualType,
		&actualFormat,
		&nitems,
		&bytesAfter,
		&prop,
	)

	if status != C.Success || prop == nil {
		return false
	}
	defer C.XFree(unsafe.Pointer(prop))

	if actualType != C.XA_ATOM || actualFormat != 32 || nitems == 0 {
		return false
	}

	atoms := unsafe.Slice((*C.Atom)(unsafe.Pointer(prop)), int(nitems))

	hasHorz := false
	hasVert := false

	for _, atom := range atoms {
		if atom == c.platform.netWMStateMaximizedHorz {
			hasHorz = true
		}
		if atom == c.platform.netWMStateMaximizedVert {
			hasVert = true
		}
	}

	return hasHorz && hasVert
}

func (c *nativeWindow) KeyModifiers() nuikey.KeyModifiers {
	return c.getModifierState()
}

func (c *nativeWindow) DrawTimeUs() int64 {
	drawTimeAvg := int64(0)
	count := 0
	for _, t := range c.drawTimes {
		if t == 0 {
			continue
		}
		drawTimeAvg += t
		count++
	}
	if count == 0 {
		return 0
	}
	drawTimeAvg = drawTimeAvg / int64(count)
	return drawTimeAvg
}

func (c *nativeWindow) SetBackgroundColor(color color.RGBA) {
	c.platform.bgColor = color
	c.Update()
}

func (c *nativeWindow) SetMouseCursor(cursor nuimouse.MouseCursor) {
	if c.currentCursor == cursor {
		return
	}
	c.currentCursor = cursor
	c.changeMouseCursor(cursor)
}

func (c *nativeWindow) changeMouseCursor(mouseCursor nuimouse.MouseCursor) bool {
	var cursorShape uint

	const (
		CursorArrow = 132
		CursorCross = 34
		CursorWait  = 150
		CursorIBeam = 152
		CursorHand  = 60
		CursorBlank = 0

		CursorResizeVertical   = 116 // XC_sb_v_double_arrow
		CursorResizeHorizontal = 108 // XC_sb_h_double_arrow
	)

	switch mouseCursor {
	case nuimouse.MouseCursorNotDefined:
	case nuimouse.MouseCursorArrow:
		cursorShape = CursorArrow
	case nuimouse.MouseCursorPointer:
		cursorShape = CursorHand
	case nuimouse.MouseCursorResizeHor:
		cursorShape = CursorResizeHorizontal
	case nuimouse.MouseCursorResizeVer:
		cursorShape = CursorResizeVertical
	case nuimouse.MouseCursorIBeam:
		cursorShape = CursorIBeam
	}

	cursor := C.XCreateFontCursor(c.platform.display, C.uint(cursorShape))
	C.XDefineCursor(c.platform.display, c.platform.window, cursor)
	C.XFlush(c.platform.display)
	return true
}

func (c *nativeWindow) MinimizeWindow() {
	C.minimizeWindow(c.platform.display, c.platform.window)
}

func (c *nativeWindow) MaximizeWindow() {
	C.maximizeWindow(c.platform.display, c.platform.window)
}

// SetAllowMinimize shows or hides the titlebar's minimize button via
// _MOTIF_WM_HINTS, e.g. for dialog-style windows that shouldn't offer it.
func (c *nativeWindow) SetAllowMinimize(allow bool) {
	c.platform.allowMinimize = allow
	c.applyWindowDecorations()
}

// SetAllowMaximize shows or hides the titlebar's maximize button via
// _MOTIF_WM_HINTS, e.g. for dialog-style windows that shouldn't offer it.
func (c *nativeWindow) SetAllowMaximize(allow bool) {
	c.platform.allowMaximize = allow
	c.applyWindowDecorations()
}

func (c *nativeWindow) applyWindowDecorations() {
	allowMin, allowMax := C.int(0), C.int(0)
	if c.platform.allowMinimize {
		allowMin = 1
	}
	if c.platform.allowMaximize {
		allowMax = 1
	}
	C.setWindowDecorations(c.platform.display, c.platform.window, allowMin, allowMax)
}

// ShowModal marks the window as a modal dialog owned by parent (WM_TRANSIENT_FOR +
// _NET_WM_STATE_MODAL) and shows it. Show() already runs the event pump on its
// own goroutine, so parent's event loop/timer keep running. The WM (verified
// against kwin_x11/KDE) only honors this for keyboard focus, not pointer
// input, so parent's own pump loop also drops keyboard/mouse callbacks for
// as long as this dialog - tracked via parent's modalChildCount - stays open.
func (c *nativeWindow) ShowModal(parent Window) {
	// Map first: setWindowModal's _NET_WM_STATE_MODAL request is a
	// ClientMessage sent to root, which a WM only honors for a window it
	// already manages (i.e. already mapped). Sending it before Show() maps
	// the window means the WM silently drops it - the window still opens,
	// just non-modal, since it's asking about a window it doesn't know yet.
	c.Show()
	if p, ok := parent.(*nativeWindow); ok && p != nil {
		c.platform.modalParent = p
		atomic.AddInt32(&p.platform.modalChildCount, 1)
		C.setWindowModal(c.platform.display, c.platform.window, p.platform.window)
	}
}

// inputBlocked reports whether c should drop keyboard/mouse callbacks
// because a modal dialog it owns is currently open (see modalChildCount).
func (c *nativeWindow) inputBlocked() bool {
	return atomic.LoadInt32(&c.platform.modalChildCount) > 0
}

func (c *nativeWindow) SetAppIcon(icon *image.RGBA) {
	width := icon.Bounds().Dx()
	height := icon.Bounds().Dy()

	// _NET_WM_ICON: [width, height, pixels...]
	dataLen := 2 + width*height
	data := make([]C.ulong, dataLen)
	data[0] = C.ulong(width)
	data[1] = C.ulong(height)

	i := 2
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := icon.PixOffset(x, y)
			r := icon.Pix[offset]
			g := icon.Pix[offset+1]
			b := icon.Pix[offset+2]
			a := icon.Pix[offset+3]

			argb := (uint32(a) << 24) | (uint32(r) << 16) | (uint32(g) << 8) | uint32(b)
			data[i] = C.ulong(argb)
			i++
		}
	}

	atom := C.XInternAtom(c.platform.display, C.CString("_NET_WM_ICON"), C.False)
	typ := C.Atom(C.XA_CARDINAL)
	format := 32

	C.XChangeProperty(
		c.platform.display,
		c.platform.window,
		atom,
		typ,
		C.int(format),
		C.PropModeReplace,
		(*C.uchar)(unsafe.Pointer(&data[0])),
		C.int(len(data)),
	)
}

func (c *nativeWindow) drawImageRGBA(display *C.Display, window C.Window, img image.Image) {
	width := c.windowWidth
	height := c.windowHeight

	dataSize := width * height * 4

	rgba := img.(*image.RGBA).Pix

	// RGBA->BGRA
	pixelsCount := width * height
	for i := 0; i < pixelsCount; i++ {
		rgba[i*4], rgba[i*4+2] = rgba[i*4+2], rgba[i*4]
	}

	cBuffer := C.malloc(C.size_t(dataSize))
	C.memcpy(cBuffer, unsafe.Pointer(&rgba[0]), C.size_t(dataSize))

	ximage := C.XCreateImage(
		display,
		C.XDefaultVisual(display, C.XDefaultScreen(display)),
		24,
		C.ZPixmap,
		0,
		(*C.char)(cBuffer),
		C.uint(width),
		C.uint(height),
		32,
		0,
	)

	//C.DestroyXImage(ximage) // TODO:

	gc := C.XCreateGC(display, C.Drawable(window), 0, nil)
	defer C.XFreeGC(display, gc) // TODO:

	C.XPutImage(display, C.Drawable(window), gc, ximage, 0, 0, 0, 0, C.uint(width), C.uint(height))

	C.destroy_ximage(ximage)
}

/*func drawBlue(display *C.Display, window C.Window, screen C.int) {
	gc := C.XCreateGC(display, C.Drawable(window), 0, nil)
	defer C.XFreeGC(display, gc)
	colorName := C.CString("blue")
	defer C.free(unsafe.Pointer(colorName))

	var exactColor, screenColor C.XColor
	C.XAllocNamedColor(display, C.XDefaultColormap(display, screen), colorName, &screenColor, &exactColor)

	C.XSetForeground(display, gc, screenColor.pixel)

	C.XFillRectangle(display, C.Drawable(window), gc, 0, 0, width/2, height/2)
}
*/

func (c *nativeWindow) SystemHandle() any {
	return nil
}

func (c *nativeWindow) getModifierState() nuikey.KeyModifiers {
	display := (*C.Display)(c.platform.display)
	window := (C.Window)(c.platform.window)

	var rootRet, childRet C.Window
	var rootX, rootY, winX, winY C.int
	var mask C.uint

	C.XQueryPointer(
		display,
		window,
		&rootRet,
		&childRet,
		&rootX, &rootY,
		&winX, &winY,
		&mask,
	)

	return nuikey.KeyModifiers{
		Shift: (mask & C.ShiftMask) != 0,
		Ctrl:  (mask & C.ControlMask) != 0,
		Alt:   (mask & C.Mod1Mask) != 0,
	}
}

func (c *nativeWindow) getFrameExtents() (left, right, top, bottom int, ok bool) {
	display := c.platform.display
	window := c.platform.window

	name := C.CString("_NET_FRAME_EXTENTS")
	defer C.free(unsafe.Pointer(name))

	atom := C.XInternAtom(display, name, C.False)

	var actualType C.Atom
	var actualFormat C.int
	var nitems C.ulong
	var bytesAfter C.ulong
	var prop *C.uchar

	if c.platform.closed {
		return 0, 0, 0, 0, false
	}

	status := C.XGetWindowProperty(
		display,
		window,
		atom,
		0,
		4,
		C.False,
		C.XA_CARDINAL,
		&actualType,
		&actualFormat,
		&nitems,
		&bytesAfter,
		&prop,
	)

	if status != C.Success || prop == nil {
		return 0, 0, 0, 0, false
	}
	defer C.XFree(unsafe.Pointer(prop))

	if actualType != C.XA_CARDINAL || actualFormat != 32 || nitems < 4 {
		return 0, 0, 0, 0, false
	}

	data := (*[4]C.ulong)(unsafe.Pointer(prop))

	left = int(data[0])
	right = int(data[1])
	top = int(data[2])
	bottom = int(data[3])

	return left, right, top, bottom, true
}

func (c *nativeWindow) updateWindowPos() {
	display := c.platform.display
	window := c.platform.window
	root := C.XRootWindow(display, c.platform.screen)

	var x, y C.int
	var child C.Window

	if C.XTranslateCoordinates(
		display,
		window,
		root,
		0, 0,
		&x, &y,
		&child,
	) == 0 {
		return
	}

	c.windowPosX = int(x)
	c.windowPosY = int(y)
}
