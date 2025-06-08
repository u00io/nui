package exmaple04paint

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"github.com/u00io/nui/nui"
	"github.com/u00io/nui/nuicanvas"
	"github.com/u00io/nui/nuimouse"
)

func Run() {
	doc := image.NewRGBA(image.Rect(0, 0, 1800, 1600))

	buttonPressed := false

	wnd := nui.CreateWindow("App", 800, 600, true)

	wnd.OnPaint(func(rgba *image.RGBA) {
		rect := rgba.Rect
		fmt.Println("OnPaint", rect)
		cnv := nuicanvas.NewCanvas(rgba)
		draw.Draw(rgba, rgba.Rect, doc, image.Point{}, draw.Src)
		cnv.SetColor(color.RGBA{0, 255, 0, 255})
		cnv.DrawRect(0, 0, 100, 100)
	})

	lastX, lastY := 0, 0

	wnd.OnMouseButtonDown(func(button nuimouse.MouseButton, x, y int) {
		//doc.Set(x, y, color.RGBA{255, 0, 0, 255})
		lastX, lastY = x, y
		buttonPressed = true
	})

	wnd.OnMouseButtonUp(func(button nuimouse.MouseButton, x, y int) {
		buttonPressed = false
	})

	wnd.OnMouseMove(func(x, y int) {
		if buttonPressed {
			//doc.Set(x, y, color.RGBA{255, 0, 0, 255})
			cc := nuicanvas.NewCanvas(doc)
			cc.SetColor(color.RGBA{255, 0, 0, 255})
			cc.DrawLine(lastX, lastY, x, y, 1)
		}
		lastX, lastY = x, y
		wnd.Update()

	})

	wnd.SetTitle("Example 04 - Paint")
	wnd.Show()
	wnd.Resize(800, 600)
	wnd.EventLoop()
}
