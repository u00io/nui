package example00demo

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"time"

	"github.com/u00io/nui/nui"
	"github.com/u00io/nui/nuicanvas"
	"github.com/u00io/nui/nuikey"
	"github.com/u00io/nui/nuimouse"
)

var logItems = make([]string, 0)

var allowCloseWindow = true

func log(s string) {
	dtStr := time.Now().Format("15:04:05.999")
	for len(dtStr) < 12 {
		dtStr += "0"
	}

	s = dtStr + " " + s
	logItems = append(logItems, s)
	if len(logItems) > 20 {
		logItems = logItems[1:]
	}
}

// showLabelWindow opens a small window with the given label, either as a
// modal dialog owned by parent (WM blocks input to parent) or as an
// independent non-modal window — both run concurrently with parent. Pressing
// M inside the opened window opens a further modal dialog owned by *it*,
// testing a modal stacked on top of a non-modal (or another modal) window.
func showLabelWindow(text string, modal bool, parent nui.Window) {
	w := nui.CreateWindow(text, 0, 0, 320, 160, true, false)
	// Dialog-style windows: no reason to offer minimize/maximize.
	w.SetAllowMinimize(false)
	w.SetAllowMaximize(false)
	w.OnPaint(func(rgba *image.RGBA) {
		cnv := nuicanvas.NewCanvas(rgba)
		cnv.SetColor(color.RGBA{0, 255, 0, 255})
		cnv.DrawFixedString(10, 10, text, 2)
		cnv.DrawFixedString(10, 30, "Press Esc to close", 2)
		cnv.DrawFixedString(10, 50, "Press M for a nested modal dialog", 2)
	})
	w.OnKeyDown(func(keyCode nuikey.Key, modifiers nuikey.KeyModifiers) bool {
		switch keyCode {
		case nuikey.KeyEsc:
			w.Close()
		case nuikey.KeyM:
			showLabelWindow("Nested modal dialog", true, w)
		}
		return true
	})
	if modal {
		w.ShowModal(parent)
	} else {
		w.Show()
	}
}

func Run() {
	win := nui.CreateWindow("App", 10, 10, 800, 600, false, false)

	var timerCounter = 0
	var mousePosX, mousePosY = 0, 0
	var winPosX, winPosY = 0, 0
	var winWidth, winHeight = 0, 0
	var mouseWheelX, mouseWheelY = 0, 0
	var animationOffset = 0
	var timerPeriodMs = 0

	win.OnKeyDown(func(keyCode nuikey.Key, modifiers nuikey.KeyModifiers) bool {
		modStr := modifiers.String()
		if len(modStr) > 0 {
			modStr = " + " + modStr
		}
		log("OnKeyDown: " + keyCode.String() + modStr)
		switch keyCode {
		case nuikey.KeyEsc:
			logItems = nil
		case nuikey.KeyF1:
			win.MaximizeWindow()
		case nuikey.KeyF2:
			win.MinimizeWindow()
		case nuikey.KeyF3:
			win.SetTitle("Title: " + time.Now().Format("15:04:05"))
		case nuikey.KeyF4:
			win.Resize(640, 480)
		case nuikey.KeyF5:
			win.Move(458, 141)
		case nuikey.KeyF6:
			win.MoveToCenterOfScreen()
		case nuikey.KeyF7:
			{
				iconImg := image.NewRGBA(image.Rect(0, 0, 16, 16))
				cnv := nuicanvas.NewCanvas(iconImg)
				cnv.SetColor(color.RGBA{0, 0, 0, 255})
				cnv.FillRect(0, 0, 16, 16, 1)
				cnv.SetColor(color.RGBA{0, 150, 200, 255})
				cnv.DrawFixedString(0, 4, "NUI", 1)
				win.SetAppIcon(iconImg)
			}
		case nuikey.KeyF8:
			win.SetMouseCursor(nuimouse.MouseCursorArrow)
		case nuikey.KeyF9:
			win.SetMouseCursor(nuimouse.MouseCursorPointer)
		case nuikey.KeyF10:
			win.SetMouseCursor(nuimouse.MouseCursorIBeam)
		case nuikey.KeyF12:
			win.Close()
		case nuikey.Key1:
			win.SetBackgroundColor(color.RGBA{0, 0, 0, 255})
		case nuikey.Key2:
			win.SetBackgroundColor(color.RGBA{255, 0, 0, 255})
		case nuikey.Key3:
			win.SetBackgroundColor(color.RGBA{0, 255, 0, 255})
		case nuikey.Key4:
			win.SetBackgroundColor(color.RGBA{0, 0, 255, 255})
		case nuikey.Key5:
			win.SetBackgroundColor(color.RGBA{255, 255, 255, 255})
		case nuikey.Key6:
			allowCloseWindow = false
		case nuikey.KeyM:
			showLabelWindow("Modal window", true, win)
		case nuikey.KeyN:
			showLabelWindow("Non-modal window", false, win)
		case nuikey.KeyF:
			files, err := nui.OpenFileDialog(win, nui.OpenFileDialogOptions{
				Title: "Open file",
				Filters: []nui.FileDialogFilter{
					{DisplayName: "All files", Patterns: []string{"*"}},
				},
			})
			if err != nil {
				log("OpenFileDialog error: " + err.Error())
			} else if len(files) == 0 {
				log("OpenFileDialog: cancelled")
			} else {
				log("OpenFileDialog: " + strings.Join(files, ", "))
			}
		case nuikey.KeyS:
			path, err := nui.SaveFileDialog(win, nui.SaveFileDialogOptions{
				Title:           "Save file",
				DefaultFileName: "untitled.txt",
				Filters: []nui.FileDialogFilter{
					{DisplayName: "All files", Patterns: []string{"*"}},
				},
			})
			if err != nil {
				log("SaveFileDialog error: " + err.Error())
			} else if path == "" {
				log("SaveFileDialog: cancelled")
			} else {
				log("SaveFileDialog: " + path)
			}
		case nuikey.KeyD:
			dir, err := nui.SelectDirectoryDialog(win, nui.SelectDirectoryDialogOptions{
				Title: "Select directory",
			})
			if err != nil {
				log("SelectDirectoryDialog error: " + err.Error())
			} else if dir == "" {
				log("SelectDirectoryDialog: cancelled")
			} else {
				log("SelectDirectoryDialog: " + dir)
			}
		}
		win.Update()
		return false
	})

	dtLastTimer := time.Now()

	win.OnTimer(func() {
		timerCounter++
		if timerCounter > 100 {
			timerCounter = 0
			elapsedTimeMs := time.Since(dtLastTimer).Milliseconds()
			timerPeriodMs = int(elapsedTimeMs) / 100
			dtLastTimer = time.Now()
		}

		animationOffset += 1
		if animationOffset > win.Width() {
			animationOffset = 0
		}
		win.Update()
	})

	win.OnKeyUp(func(keyCode nuikey.Key, modifiers nuikey.KeyModifiers) {
		modStr := modifiers.String()
		if len(modStr) > 0 {
			modStr = " + " + modStr
		}
		log("OnKeyUp: " + keyCode.String() + modStr)
		win.Update()
	})

	win.OnChar(func(char rune) {
		log("OnChar: " + string(char))
		win.Update()
	})

	win.OnMouseLeave(func() {
		log("OnMouseLeave")
		win.Update()
	})

	win.OnMouseEnter(func() {
		log("OnMouseEnter")
		win.Update()
	})

	win.OnMouseMove(func(x, y int) {
		mousePosX = x
		mousePosY = y
		win.Update()
	})

	win.OnMouseButtonDblClick(func(button nuimouse.MouseButton, x, y int) {
		log(fmt.Sprintf("OnMouseButtonDblClick: %s (%d, %d)", button.String(), x, y))
		win.Update()
	})

	win.OnMouseButtonDown(func(button nuimouse.MouseButton, x, y int) {
		log(fmt.Sprintf("OnMouseButtonDown: %s (%d, %d)", button.String(), x, y))
		win.Update()
	})

	win.OnMouseButtonUp(func(button nuimouse.MouseButton, x, y int) {
		log(fmt.Sprintf("OnMouseButtonUp: %s (%d, %d)", button.String(), x, y))
		win.Update()
	})

	win.OnMouseWheel(func(deltaX, deltaY int) {
		log(fmt.Sprintf("OnMouseWheel: %d %d", deltaX, deltaY))
		mouseWheelX += deltaX
		mouseWheelY += deltaY
		win.Update()
	})

	win.OnMove(func(x, y int) {
		winPosX = x
		winPosY = y
		log(fmt.Sprintf("OnMove: %d %d", x, y))
		win.Update()
	})

	win.OnResize(func(width, height int) {
		winWidth = width
		winHeight = height
		log(fmt.Sprintf("OnResize: %d %d", width, height))
		win.Update()
	})

	win.OnPaint(func(rgba *image.RGBA) {
		cnv := nuicanvas.NewCanvas(rgba)
		cnv.SetColor(color.RGBA{0, 255, 0, 255})

		// legend
		cnv.DrawFixedString(10, 10, "Press F1 to maximize window", 2)
		cnv.DrawFixedString(10, 30, "Press F2 to minimize window", 2)
		cnv.DrawFixedString(10, 50, "Press F3 to change title", 2)
		cnv.DrawFixedString(10, 70, "Press F4 to resize window", 2)
		cnv.DrawFixedString(10, 90, "Press F5 to move window", 2)
		cnv.DrawFixedString(10, 110, "Press F6 to center window", 2)
		cnv.DrawFixedString(10, 130, "Press F7 to set app icon", 2)
		cnv.DrawFixedString(10, 150, "Press F8 to set arrow cursor", 2)
		cnv.DrawFixedString(10, 170, "Press F9 to set pointer cursor", 2)
		cnv.DrawFixedString(10, 190, "Press F10 to set IBeam cursor", 2)
		cnv.DrawFixedString(10, 210, "Press F12 to close window", 2)
		cnv.DrawFixedString(10, 230, "Press M for modal window", 2)
		cnv.DrawFixedString(10, 250, "Press N for non-modal window", 2)
		cnv.DrawFixedString(10, 270, "Press F to open file dialog", 2)
		cnv.DrawFixedString(10, 290, "Press S to open save dialog", 2)
		cnv.DrawFixedString(10, 310, "Press D to select a directory", 2)

		cnv.DrawFixedString(10, 330, "Timer: "+fmt.Sprint(timerCounter), 2)
		cnv.DrawFixedString(10, 350, "MouseX: "+fmt.Sprint(mousePosX), 2)
		cnv.DrawFixedString(10, 370, "MouseY: "+fmt.Sprint(mousePosY), 2)
		cnv.DrawFixedString(10, 390, "WinX: "+fmt.Sprint(winPosX), 2)
		cnv.DrawFixedString(10, 410, "WinY: "+fmt.Sprint(winPosY), 2)
		cnv.DrawFixedString(10, 430, "WinW: "+fmt.Sprint(winWidth), 2)
		cnv.DrawFixedString(10, 450, "WinH: "+fmt.Sprint(winHeight), 2)
		cnv.DrawFixedString(10, 470, "MouseWheelX: "+fmt.Sprint(mouseWheelX), 2)
		cnv.DrawFixedString(10, 490, "MouseWheelY: "+fmt.Sprint(mouseWheelY), 2)
		cnv.DrawFixedString(10, 510, "DrawTimeMs: "+fmt.Sprint(win.DrawTimeUs()/1000), 2)
		cnv.DrawFixedString(10, 530, "TimerPeriodMs: "+fmt.Sprint(timerPeriodMs), 2)

		cnv.DrawLine(5, 550, win.Width()-5, 550, 0.5)
		cnv.DrawLine(390, 5, 390, 525, 0.5)
		cnv.FillRect(animationOffset, 540, 20, 20, 0.5)

		cnv.DrawFixedString(400, 10, "Press Esc to clear log", 2)
		for i, s := range logItems {
			cnv.DrawFixedString(400, 20+float64(10+20*i), s, 2)
		}
	})

	win.OnCloseRequest(func() bool {
		return allowCloseWindow
	})

	win.Show()

	winPosX = win.PosX()
	winPosY = win.PosY()
	winWidth = win.Width()
	winHeight = win.Height()

	win.Exec()
	winPosX = win.PosX()
	winPosY = win.PosY()
	winWidth = win.Width()
	winHeight = win.Height()
}
