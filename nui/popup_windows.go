package nui

import (
	"image"
	"image/color"
	"sync"
	"syscall"
	"unsafe"
)

var (
	procClientToScreen  = user32.NewProc("ClientToScreen")
	procGetClientRect   = user32.NewProc("GetClientRect")
	procMonitorFromRect = user32.NewProc("MonitorFromRect")
	procGetMonitorInfoW = user32.NewProc("GetMonitorInfoW")
)

const (
	c_WS_POPUP = 0x80000000

	c_WS_EX_TOPMOST    = 0x00000008
	c_WS_EX_TOOLWINDOW = 0x00000080
	c_WS_EX_NOACTIVATE = 0x08000000

	c_CS_DROPSHADOW = 0x00020000

	c_SWP_SHOWWINDOW = 0x0040
	c_SWP_HIDEWINDOW = 0x0080

	c_WM_ERASEBKGND    = 0x0014
	c_WM_NCHITTEST     = 0x0084
	c_WM_MOUSEACTIVATE = 0x0021
	c_WM_APP           = 0x8000

	c_WM_NUI_CREATE_POPUP = c_WM_APP + 1

	c_HTTRANSPARENT            = ^uintptr(0) // -1
	c_MA_NOACTIVATE            = 3
	c_HWND_TOPMOST             = ^uintptr(0) // -1
	c_MONITOR_DEFAULTTONEAREST = 2
)

type t_MONITORINFO struct {
	cbSize    uint32
	rcMonitor rect
	rcWork    rect
	dwFlags   uint32
}

type popupWindow struct {
	hwnd    syscall.Handle
	onPaint func(rgba *image.RGBA)

	canvasBuffer []byte
	pixBuffer    []byte
}

var (
	popupsMu sync.Mutex
	popups   = make(map[syscall.Handle]*popupWindow)

	popupClassOnce sync.Once
	popupClassName *uint16
)

// createPopupWindow asks the owner's thread to create the popup HWND, so the
// popup shares the owner's message loop: no extra thread, and HTTRANSPARENT
// (mouse pass-through) only works between windows of the same thread.
func createPopupWindow(owner Window) PopupWindow {
	o, ok := owner.(*nativeWindow)
	if !ok || o == nil {
		return nil
	}
	popupClassOnce.Do(registerPopupClass)

	hwnd, _, _ := procSendMessageW.Call(uintptr(o.hwnd), c_WM_NUI_CREATE_POPUP, 0, 0)
	if hwnd == 0 {
		return nil
	}

	p := &popupWindow{hwnd: syscall.Handle(hwnd)}
	popupsMu.Lock()
	popups[p.hwnd] = p
	popupsMu.Unlock()
	return p
}

func registerPopupClass() {
	popupClassName, _ = syscall.UTF16PtrFromString("NUIPopupWindow")
	hInstance, _, _ := procGetModuleHandleW.Call(0)
	wndClass := t_WNDCLASSEXW{
		cbSize:        uint32(unsafe.Sizeof(t_WNDCLASSEXW{})),
		style:         c_CS_DROPSHADOW,
		lpfnWndProc:   syscall.NewCallback(popupWndProc),
		hInstance:     syscall.Handle(hInstance),
		lpszClassName: popupClassName,
	}
	procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wndClass)))
}

// createPopupHwnd runs on the owner's thread (from its wndProc). The popup
// is created hidden, so no WM_PAINT arrives before createPopupWindow
// registers it.
func createPopupHwnd(owner syscall.Handle) uintptr {
	hInstance, _, _ := procGetModuleHandleW.Call(0)
	hwnd, _, _ := procCreateWindowExW.Call(
		c_WS_EX_TOOLWINDOW|c_WS_EX_TOPMOST|c_WS_EX_NOACTIVATE,
		uintptr(unsafe.Pointer(popupClassName)),
		0,
		c_WS_POPUP,
		0, 0, 1, 1,
		uintptr(owner),
		0,
		hInstance,
		0,
	)
	return hwnd
}

func getPopupByHandle(hwnd syscall.Handle) *popupWindow {
	popupsMu.Lock()
	defer popupsMu.Unlock()
	return popups[hwnd]
}

func popupWndProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case c_WM_NCHITTEST:
		return c_HTTRANSPARENT

	case c_WM_MOUSEACTIVATE:
		return c_MA_NOACTIVATE

	case c_WM_ERASEBKGND:
		return 1

	case c_WM_PAINT:
		var ps t_PAINTSTRUCT
		hdc, _, _ := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		if p := getPopupByHandle(hwnd); p != nil {
			p.paint(hdc)
		}
		procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		return 0

	case c_WM_DESTROY:
		// Also sent when the owner is destroyed. No PostQuitMessage: the
		// message loop belongs to the owner.
		popupsMu.Lock()
		delete(popups, hwnd)
		popupsMu.Unlock()
		return 0
	}

	ret, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

func (p *popupWindow) paint(hdc uintptr) {
	var r rect
	procGetClientRect.Call(uintptr(p.hwnd), uintptr(unsafe.Pointer(&r)))
	width, height := r.right-r.left, r.bottom-r.top
	if width <= 0 || height <= 0 {
		return
	}

	buf := growBuffer(&p.canvasBuffer, int(width*height*4))
	fillCanvasBuffer(buf, color.RGBA{0, 0, 0, 255})
	img := &image.RGBA{
		Pix:    buf,
		Stride: int(width) * 4,
		Rect:   image.Rect(0, 0, int(width), int(height)),
	}

	if p.onPaint != nil {
		p.onPaint(img)
	}

	drawRGBAToHDC(img, hdc, width, height, &p.pixBuffer)
}

func (p *popupWindow) OnPaint(f func(rgba *image.RGBA)) {
	p.onPaint = f
}

func (p *popupWindow) ShowAt(x, y, width, height int) {
	procSetWindowPos.Call(
		uintptr(p.hwnd),
		c_HWND_TOPMOST,
		uintptr(x), uintptr(y),
		uintptr(width), uintptr(height),
		c_SWP_NOACTIVATE|c_SWP_SHOWWINDOW,
	)
	procInvalidateRect.Call(uintptr(p.hwnd), 0, 0)
}

func (p *popupWindow) Hide() {
	procSetWindowPos.Call(
		uintptr(p.hwnd),
		0,
		0, 0, 0, 0,
		c_SWP_NOMOVE|c_SWP_NOSIZE|c_SWP_NOZORDER|c_SWP_NOACTIVATE|c_SWP_HIDEWINDOW,
	)
}

func (p *popupWindow) Update() {
	procInvalidateRect.Call(uintptr(p.hwnd), 0, 0)
}

// Close posts WM_CLOSE instead of calling DestroyWindow, which only works on
// the thread that created the window.
func (p *popupWindow) Close() {
	procPostMessageW.Call(uintptr(p.hwnd), c_WM_CLOSE, 0, 0)
}
