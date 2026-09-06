package nui

import (
	"image"
	"image/color"
	"math/rand"
	"strconv"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/u00io/nui/nuikey"
	"github.com/u00io/nui/nuimouse"
)

type windowId syscall.Handle

// Guards app.windows: multiple windows each run their own goroutine/OS
// thread and message loop (see EventLoop), so creating one window while
// another's wndProc looks up its handle is a real concurrent map access.
var appWindowsMu sync.Mutex

type nativeWindowPlatform struct {
}

// ///////////////////////////////////////////////////
// Window creation and management

func createWindow(title string, posX int, posY int, width int, height int, center bool, maximized bool) *nativeWindow {
	var c nativeWindow
	c.dblClickTime = 300 * time.Millisecond
	c.showMaximized = maximized

	// Create a unique class name
	dt := time.Now().Format("2006-01-02-15-04-05")
	randomNumber := rand.Intn(1024 * 1024)
	tempClassName := "WCL" + dt + strconv.Itoa(randomNumber)
	className, _ := syscall.UTF16PtrFromString(tempClassName)

	initCanvasBufferBackground(color.RGBA{0x1F, 0x1F, 0x1F, 255})

	// Set default window title
	windowTitle, _ := syscall.UTF16PtrFromString(title)

	// Set default cursor
	c.currentCursor = nuimouse.MouseCursorArrow

	// Get the instance handle
	hInstance, _, _ := procGetModuleHandleW.Call(0)

	// Register the window class
	wndClass := t_WNDCLASSEXW{
		cbSize:        uint32(unsafe.Sizeof(t_WNDCLASSEXW{})),
		style:         c_CS_OWNDC, /*| c_CS_DBLCLKS*/
		lpfnWndProc:   syscall.NewCallback(wndProc),
		hInstance:     syscall.Handle(hInstance),
		hCursor:       0,
		hbrBackground: 5,
		lpszClassName: className,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wndClass)))

	windowFlags := uint32(c_WS_OVERLAPPEDWINDOW)
	if c.showMaximized {
		windowFlags |= c_WS_MAXIMIZE
	}

	// Create the window
	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowTitle)),
		uintptr(windowFlags),
		c_CW_USEDEFAULT,
		c_CW_USEDEFAULT,
		uintptr(width),
		uintptr(height),
		0,
		0,
		hInstance,
		0,
	)

	c.windowWidth = width
	c.windowHeight = height

	// Store the window handle
	c.hwnd = windowId(syscall.Handle(hwnd))
	appWindowsMu.Lock()
	app.windows[c.hwnd] = &c
	appWindowsMu.Unlock()

	// Set default icon
	icon := image.NewRGBA(image.Rect(0, 0, 32, 32))
	c.SetAppIcon(icon)

	if center && !maximized {
		c.MoveToCenterOfScreen()
	}

	setDarkMode(hwnd, true)

	return &c
}

func (c *nativeWindow) Show() {
	if c.showMaximized {
		procShowWindow.Call(uintptr(c.hwnd), c_SW_SHOWMAXIMIZED)
	} else {
		procShowWindow.Call(uintptr(c.hwnd), c_SW_SHOWDEFAULT)
	}
	procInvalidateRect.Call(uintptr(c.hwnd), 0, 0)
	procUpdateWindow.Call(uintptr(c.hwnd))
}

func (c *nativeWindow) Update() {
	// Update the window
	procInvalidateRect.Call(uintptr(c.hwnd), 0, 0)
	procUpdateWindow.Call(uintptr(c.hwnd))
}

func (c *nativeWindow) EventLoop() {
	var msg t_MSG

	procSetTimer.Call(
		uintptr(c.hwnd),
		timerID1ms,
		1,
		0,
	)

	procInvalidateRect.Call(uintptr(c.hwnd), 0, 0)
	for {
		ret, _, err := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		e := err.(syscall.Errno)
		if e != 0 {
			//fmt.Println("Error:", e)
		}

		if ret == 0 {
			//fmt.Println("Exiting...")
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func (c *nativeWindow) Close() {
	procPostMessageW.Call(uintptr(c.hwnd), c_WM_DESTROY, 0, 0)
}

// ShowModal marks this window as owned by parent (GWLP_HWNDPARENT, so it
// stacks above parent, minimizes with it, and gets no separate taskbar
// button) and disables parent's HWND for the duration, the classic Win32
// technique dialogs use internally. It runs on its own goroutine/thread like
// a non-modal window, so parent's own event loop (repaint, timers) keeps
// running - matching the Linux contract documented on Window.ShowModal.
func (c *nativeWindow) ShowModal(parent Window) {
	var hwndOwner uintptr
	if p, ok := parent.(*nativeWindow); ok && p != nil {
		hwndOwner = uintptr(p.hwnd)
	}

	if hwndOwner != 0 {
		procSetWindowLongPtrW.Call(uintptr(c.hwnd), gwlHwndParentIndex(), hwndOwner)
		procEnableWindow.Call(hwndOwner, 0)
	}

	go func() {
		c.Exec()
		if hwndOwner != 0 {
			procEnableWindow.Call(hwndOwner, 1)
			procSetForegroundWindow.Call(hwndOwner)
		}
	}()
}

// gwlHwndParentIndex returns GWLP_HWNDPARENT (-8) sign-extended to uintptr.
// Converting the negative literal straight to uintptr is a compile error
// (Go constant-conversion rules reject negative-to-unsigned even when typed);
// routing it through an int32 function parameter forces a runtime
// conversion, which correctly sign-extends instead.
func gwlHwndParentIndex() uintptr {
	return gwlIndexToUintptr(-8)
}

func gwlIndexToUintptr(n int32) uintptr {
	return uintptr(n)
}

///////////////////////////////////////////////////
// Window appearance

func (c *nativeWindow) SetTitle(title string) {
	strPtr, _ := syscall.UTF16PtrFromString(title)
	procSetWindowTextW.Call(
		uintptr(c.hwnd),
		uintptr(unsafe.Pointer(strPtr)),
	)
}

func (c *nativeWindow) SetAppIcon(icon *image.RGBA) {
	hIcon := createHICONFromRGBA(icon)
	if hIcon == 0 {
		//fmt.Println("failed to create icon")
		return
	}

	procSendMessageW.Call(uintptr(c.hwnd), c_WM_SETICON, c_ICON_BIG, uintptr(hIcon))
	procSendMessageW.Call(uintptr(c.hwnd), c_WM_SETICON, c_ICON_SMALL, uintptr(hIcon))
}

func (c *nativeWindow) SetBackgroundColor(color color.RGBA) {
	initCanvasBufferBackground(color)
	c.Update()
}

func (c *nativeWindow) SetMouseCursor(cursor nuimouse.MouseCursor) {
	if c.currentCursor == cursor {
		return
	}
	c.currentCursor = cursor
	c.changeMouseCursor(cursor)
}

/////////////////////////////////////////////////////
// Window position and size

func (c *nativeWindow) Move(x, y int) {
	flags := c_SWP_NOSIZE | c_SWP_NOZORDER

	procSetWindowPos.Call(
		uintptr(c.hwnd),
		0,
		uintptr(x), uintptr(y),
		0, 0,
		uintptr(flags),
	)
}

func (c *nativeWindow) MoveToCenterOfScreen() {
	screenWidth, screenHeight := getScreenSize()
	windowWidth, windowHeight := c.Size()
	x := (screenWidth - windowWidth) / 2
	y := (screenHeight - windowHeight) / 2
	c.Move(int(x), int(y))
}

func (c *nativeWindow) Resize(width, height int) {
	flags := c_SWP_NOMOVE | c_SWP_NOZORDER

	procSetWindowPos.Call(
		uintptr(c.hwnd),
		0,
		0, 0,
		uintptr(width),
		uintptr(height),
		uintptr(flags),
	)
}

func (c *nativeWindow) MinimizeWindow() {
	procShowWindow.Call(uintptr(c.hwnd), c_SW_SHOWMINIMIZED)
}

func (c *nativeWindow) MaximizeWindow() {
	procShowWindow.Call(uintptr(c.hwnd), c_SW_SHOWMAXIMIZED)
}

//////////////////////////////////////////////////
// Window information

func (c *nativeWindow) Size() (width, height int) {
	return c.windowWidth, c.windowHeight
}

func (c *nativeWindow) Pos() (x, y int) {
	return c.windowPosX, c.windowPosY
}

func (c *nativeWindow) PosX() int {
	return c.windowPosX
}

func (c *nativeWindow) PosY() int {
	return c.windowPosY
}

func (c *nativeWindow) Width() int {
	return c.windowWidth
}

func (c *nativeWindow) Height() int {
	return c.windowHeight
}

func (c *nativeWindow) IsMaximized() bool {
	ret, _, _ := procIsZoomed.Call(uintptr(c.hwnd))
	return ret != 0
}

func (c *nativeWindow) KeyModifiers() nuikey.KeyModifiers {
	return getModifierState()
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

func (c *nativeWindow) SystemHandle() any {
	return syscall.Handle(c.hwnd)
}
