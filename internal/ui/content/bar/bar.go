// Package bar contains the control bar.
package bar

import (
	"github.com/diamondburned/aqours/internal/ui/content/bar/controls"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/gotk3/gotk3/gtk"
)

type ParentController interface {
	controls.ParentController
}

var nowPlayingCSS = css.PrepareClass("now-playing", `
	box.now-playing {
		margin: 8px;
	}
`)

type Container struct {
	gtk.Grid
	NowPlaying *NowPlaying
	Controls   *controls.Container
	TODOItem   *gtk.Label
}

func NewContainer(parent ParentController) *Container {
	nowpl := NewNowPlaying()
	nowpl.SetHAlign(gtk.ALIGN_START)
	nowpl.Show()
	nowPlayingCSS(nowpl)

	controls := controls.NewContainer(parent)
	controls.SetHExpand(true)
	controls.SetHAlign(gtk.ALIGN_FILL)
	controls.Show()

	item, _ := gtk.LabelNew("Volume")
	item.SetHAlign(gtk.ALIGN_END)
	item.Show()

	grid, _ := gtk.GridNew()
	grid.SetRowHomogeneous(true)
	grid.SetColumnHomogeneous(true)
	grid.SetColumnSpacing(5)
	grid.SetHExpand(true)

	grid.Attach(nowpl, 0, 0, 1, 1)    // 1st column
	grid.Attach(controls, 1, 0, 2, 1) // 2nd-3rd; span 2 columns
	grid.Attach(item, 3, 0, 1, 1)     // 4th column
	grid.Show()

	return &Container{
		Grid:       *grid,
		NowPlaying: nowpl,
		Controls:   controls,
		TODOItem:   item,
	}
}
