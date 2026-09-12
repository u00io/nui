package example05multiwindow

import (
	"image"
	"image/color"

	"github.com/u00io/nui/nui"
	"github.com/u00io/nui/nuicanvas"
	"github.com/u00io/nui/nuikey"
)

func paintLabel(rgba *image.RGBA, text string) {
	cnv := nuicanvas.NewCanvas(rgba)
	cnv.SetColor(color.RGBA{230, 230, 230, 255})
	cnv.DrawFixedString(10, 10, text, 2)
}

// Run demonstrates two independent windows running concurrently via nui.Run,
// plus a modal dialog opened from the first window with ShowModal.
func Run() {
	win1 := nui.CreateWindow("Window 1 (press Space for modal dialog)", 100, 100, 500, 300, true, false)
	win2 := nui.CreateWindow("Window 2", 650, 100, 500, 300, true, false)

	win1.OnPaint(func(rgba *image.RGBA) {
		paintLabel(rgba, "Window 1")
	})
	win2.OnPaint(func(rgba *image.RGBA) {
		paintLabel(rgba, "Window 2")
	})

	win1.OnKeyDown(func(keyCode nuikey.Key, mods nuikey.KeyModifiers) bool {
		if keyCode == nuikey.KeySpace {
			dlg := nui.CreateWindow("Modal dialog", 0, 0, 300, 150, true, false)
			// Dialog-style window: no reason to offer minimize/maximize.
			dlg.SetAllowMinimize(false)
			dlg.SetAllowMaximize(false)
			dlg.OnPaint(func(rgba *image.RGBA) {
				paintLabel(rgba, "Modal dialog (Esc to close)")
			})
			dlg.OnKeyDown(func(keyCode nuikey.Key, mods nuikey.KeyModifiers) bool {
				if keyCode == nuikey.KeyEsc {
					dlg.Close()
				}
				return true
			})
			dlg.ShowModal(win1)
		}
		return true
	})

	nui.Run(win1, win2)
}
