package nui

import (
	"image"
	"math"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego/objc"
)

const (
	nsWindowStyleMaskBorderless = 0
	nsPopUpMenuWindowLevel      = 101
)

var (
	selSetLevel              = objc.RegisterName("setLevel:")
	selSetIgnoresMouseEvents = objc.RegisterName("setIgnoresMouseEvents:")
	selSetHasShadow          = objc.RegisterName("setHasShadow:")
	selSetReleasedWhenClosed = objc.RegisterName("setReleasedWhenClosed:")
	selOrderFront            = objc.RegisterName("orderFront:")
	selOrderOut              = objc.RegisterName("orderOut:")
	selClose                 = objc.RegisterName("close")
	selAcceptsFirstMouse     = objc.RegisterName("acceptsFirstMouse:")

	nuiPopupViewClass     objc.Class
	nuiPopupViewClassOnce sync.Once
)

// popupWindow is a borderless NSWindow: borderless windows can't become key,
// so showing it never takes focus from the owner. It isn't in cocoaWindows
// and has no delegate, so none of the window close/app quit logic sees it.
// Like the rest of the Cocoa code, it's only touched on the main thread.
type popupWindow struct {
	popupCallbacks

	owner  windowId
	win    objc.ID
	view   objc.ID
	closed bool
}

// Keyed by the popup's content view, which drawRect: gets as self.
// Main thread only, like cocoaWindows.
var popups = map[objc.ID]*popupWindow{}

func createPopupWindow(owner Window, interactive bool) PopupWindow {
	o, ok := owner.(*nativeWindow)
	if !ok || o == nil {
		return nil
	}
	nuiPopupViewClassOnce.Do(registerPopupViewClass)
	if nuiPopupViewClass == 0 {
		return nil
	}

	p := &popupWindow{owner: o.hwnd}
	runOnMainSync(func() {
		withAutoreleasePool(func() {
			frame := nsRect{nsPoint{0, 0}, nsSize{1, 1}}
			win := objc.ID(clsNSWindow).Send(selAlloc)
			win = win.Send(selInitWithContentRectStyleMaskBackingDefer, frame, nsWindowStyleMaskBorderless, nsBackingStoreBuffered, false)
			win.Send(selSetReleasedWhenClosed, false)
			win.Send(selSetLevel, nsPopUpMenuWindowLevel)
			// A tooltip lets the mouse through to the owner, which keeps its
			// hover state
			win.Send(selSetIgnoresMouseEvents, !interactive)
			win.Send(selSetHasShadow, true)

			view := objc.ID(nuiPopupViewClass).Send(selAlloc).Send(selInitWithFrame, frame)
			win.Send(selSetContentView, view)
			view.Send(selRelease) // the window keeps it

			p.win = win
			p.view = view
			popups[view] = p
		})
	})
	return p
}

func registerPopupViewClass() {
	cls, err := objc.RegisterClass(
		"NUIPopupView",
		objc.GetClass("NSView"),
		nil, nil,
		[]objc.MethodDef{
			{Cmd: selIsFlipped, Fn: nuiViewIsFlipped},
			{Cmd: selDrawRect, Fn: nuiPopupViewDrawRect},
			// The popup never becomes key, so without this the first click
			// would only be used to "activate" it
			{Cmd: selAcceptsFirstMouse, Fn: nuiPopupViewAcceptsFirstMouse},
			{Cmd: selMouseDown, Fn: nuiPopupViewMouseDown},
			{Cmd: selRightMouseDown, Fn: nuiPopupViewRightMouseDown},
			{Cmd: selOtherMouseDown, Fn: nuiPopupViewOtherMouseDown},
			{Cmd: selMouseUp, Fn: nuiPopupViewMouseUp},
			{Cmd: selRightMouseUp, Fn: nuiPopupViewRightMouseUp},
			{Cmd: selOtherMouseUp, Fn: nuiPopupViewOtherMouseUp},
			{Cmd: selMouseMoved, Fn: nuiPopupViewMouseMoved},
			{Cmd: selMouseDragged, Fn: nuiPopupViewMouseMoved},
			{Cmd: selRightMouseDragged, Fn: nuiPopupViewMouseMoved},
			{Cmd: selMouseExited, Fn: nuiPopupViewMouseExited},
			{Cmd: selScrollWheel, Fn: nuiPopupViewScrollWheel},
			{Cmd: selUpdateTrackingAreas, Fn: nuiViewUpdateTrackingAreas},
		},
	)
	if err != nil {
		return
	}
	nuiPopupViewClass = cls
}

// closePopupsOf closes the popups of a window that is closing.
// Called on the main thread from nuiWindowWillClose.
func closePopupsOf(owner windowId) {
	for _, p := range popups {
		if p.owner == owner {
			p.close()
		}
	}
}

// nuiPopupViewDrawRect: see nuiViewDrawRect for the flattened NSRect args
// and why the Go buffer needs no release callback.
func nuiPopupViewDrawRect(self objc.ID, _ objc.SEL, _, _, _, _ float64) {
	p, ok := popups[self]
	if !ok {
		return
	}

	bounds := objc.Send[nsRect](self, selBounds)
	width := int(math.Max(1, math.Floor(bounds.Size.Width)))
	height := int(math.Max(1, math.Floor(bounds.Size.Height)))
	stride := width * 4
	dataSize := stride * height

	buf := make([]byte, dataSize)
	img := &image.RGBA{
		Pix:    buf,
		Stride: stride,
		Rect:   image.Rect(0, 0, width, height),
	}
	for i := 3; i < dataSize; i += 4 {
		buf[i] = 255 // opaque black
	}
	if p.onPaint != nil {
		p.onPaint(img)
	}

	ctx := objc.Send[uintptr](objc.ID(clsNSGraphicsContext).Send(selCurrentContext), selCGContext)
	colorSpace := cgColorSpaceCreateDeviceRGB()
	provider := cgDataProviderCreateWithData(0, unsafe.Pointer(&buf[0]), uintptr(dataSize), 0)
	cgImg := cgImageCreate(uintptr(width), uintptr(height), 8, 32, uintptr(stride), colorSpace,
		cgImageAlphaPremultipliedLast|cgBitmapByteOrder32Big, provider, 0, false, cgRenderingIntentDefault)

	dest := nsRect{bounds.Origin, nsSize{float64(width), float64(height)}}
	cgContextDrawImage(ctx, dest, cgImg)

	cgImageRelease(cgImg)
	cgDataProviderRelease(provider)
	cgColorSpaceRelease(colorSpace)
}

func nuiPopupViewAcceptsFirstMouse(self objc.ID, _ objc.SEL, _ objc.ID) bool { return true }

// popupEventLocation returns the event position in top-down view coordinates.
func popupEventLocation(self, event objc.ID) (int, int) {
	pt := eventLocationInView(self, event)
	bounds := objc.Send[nsRect](self, selBounds)
	return int(pt.X), int(bounds.Size.Height - pt.Y)
}

func nuiPopupHandleMouseButton(self, event objc.ID, button int, down bool) {
	p, ok := popups[self]
	if !ok {
		return
	}
	x, y := popupEventLocation(self, event)
	if down {
		p.mouseButtonDown(convertMacMouseButtons(button), x, y)
	} else {
		p.mouseButtonUp(convertMacMouseButtons(button), x, y)
	}
}

func nuiPopupViewMouseDown(self objc.ID, _ objc.SEL, event objc.ID) {
	nuiPopupHandleMouseButton(self, event, 0, true)
}
func nuiPopupViewRightMouseDown(self objc.ID, _ objc.SEL, event objc.ID) {
	nuiPopupHandleMouseButton(self, event, 1, true)
}
func nuiPopupViewOtherMouseDown(self objc.ID, _ objc.SEL, event objc.ID) {
	nuiPopupHandleMouseButton(self, event, 2, true)
}
func nuiPopupViewMouseUp(self objc.ID, _ objc.SEL, event objc.ID) {
	nuiPopupHandleMouseButton(self, event, 0, false)
}
func nuiPopupViewRightMouseUp(self objc.ID, _ objc.SEL, event objc.ID) {
	nuiPopupHandleMouseButton(self, event, 1, false)
}
func nuiPopupViewOtherMouseUp(self objc.ID, _ objc.SEL, event objc.ID) {
	nuiPopupHandleMouseButton(self, event, 2, false)
}

func nuiPopupViewMouseMoved(self objc.ID, _ objc.SEL, event objc.ID) {
	if p, ok := popups[self]; ok {
		p.mouseMove(popupEventLocation(self, event))
	}
}

func nuiPopupViewScrollWheel(self objc.ID, _ objc.SEL, event objc.ID) {
	p, ok := popups[self]
	if !ok {
		return
	}
	dx := wheelStep(objc.Send[float64](event, selDeltaX))
	dy := wheelStep(objc.Send[float64](event, selDeltaY))
	if dx != 0 || dy != 0 {
		p.mouseWheel(dx, dy)
	}
}

func nuiPopupViewMouseExited(self objc.ID, _ objc.SEL, _ objc.ID) {
	if p, ok := popups[self]; ok {
		p.mouseLeave()
	}
}

// ShowAt takes top-down global coordinates (see cocoaToTopDownY).
func (p *popupWindow) ShowAt(x, y, width, height int) {
	runOnMainSync(func() {
		if p.closed || width <= 0 || height <= 0 {
			return
		}
		bottom := cocoaToTopDownY(float64(y + height))
		frame := nsRect{nsPoint{float64(x), bottom}, nsSize{float64(width), float64(height)}}
		p.win.Send(selSetFrameDisplayAnimate, frame, false, false)
		p.view.Send(selSetNeedsDisplay, true)
		p.win.Send(selOrderFront, objc.ID(0))
	})
}

func (p *popupWindow) Hide() {
	runOnMainSync(func() {
		if p.closed {
			return
		}
		p.win.Send(selOrderOut, objc.ID(0))
	})
}

func (p *popupWindow) Update() {
	runOnMainSync(func() {
		if p.closed {
			return
		}
		p.view.Send(selSetNeedsDisplay, true)
	})
}

func (p *popupWindow) Close() {
	runOnMainSync(p.close)
}

func (p *popupWindow) close() {
	if p.closed {
		return
	}
	p.closed = true
	delete(popups, p.view)
	// Not released, same as regular windows (see nuiWindowWillClose): one
	// small leak per form is better than a crash inside AppKit
	p.win.Send(selClose)
}
