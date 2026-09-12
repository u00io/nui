package example01base

import (
	"github.com/u00io/nui/nui"
)

func Run() {
	w := nui.CreateDefaultWindow()
	w.Show()
	w.Exec()
}
