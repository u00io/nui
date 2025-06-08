package example03animation

import (
	"image"
	"image/color"
	"strconv"
	"time"

	"github.com/u00io/nui/nui"
	"github.com/u00io/nui/nuicanvas"
	"github.com/u00io/nui/nuikey"
)

func fullRectOnRGBA(rgba *image.RGBA, x, y, w, h int, c color.Color) {
	for i := x; i < x+w; i++ {
		for j := y; j < y+h; j++ {
			rgba.Set(i, j, c)
		}
	}
}

func Run() {
	totalCounter := 0
	counter := 0
	speed := float64(0)
	wnd := nui.CreateWindow("App", 800, 600, true)
	wnd.Show()
	wnd.OnPaint(func(rgba *image.RGBA) {
		posX := 2 * int(time.Now().UnixMilli()%10000) / 20
		fullRectOnRGBA(rgba, posX, 10, 100, 100, color.RGBA{255, 0, 0, 255})
		cnv := nuicanvas.NewCanvas(rgba)
		cnv.SetColor(color.RGBA{200, 200, 200, 255})
		counterStr := "Counter: " + strconv.FormatInt(int64(counter), 10)
		cnv.DrawFixedString(10, 120, counterStr, 2)
		speedStr := "Speed: " + strconv.FormatFloat(speed, 'f', 2, 64)
		cnv.DrawFixedString(10, 140, speedStr, 2)
	})
	wnd.OnKeyDown(func(keyCode nuikey.Key, keyModifiers nuikey.KeyModifiers) {
		wnd.Resize(800, 600)
	})
	dtBegin := time.Now()
	lastTotalCounter := 0
	wnd.OnTimer(func() {
		counter++
		totalCounter++
		if counter > 100 {
			counter = 0
			dtEnd := time.Now()
			dt := dtEnd.Sub(dtBegin)
			duration := dt.Seconds()
			_ = duration

			speed = float64(totalCounter-lastTotalCounter) / duration
			lastTotalCounter = totalCounter

			dtBegin = time.Now()
		}
		wnd.Update()
	})
	wnd.EventLoop()
}
