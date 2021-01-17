// Package bar contains the control bar.
package bar

import (
	"github.com/diamondburned/aqours/internal/ui/content/bar/controls"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/gotk3/gotk3/gtk"
)

type ParentController interface {
	controls.ParentController
	ScrollToPlaying()
	SetVolume(perc float64)
	SetMute(muted bool)
}

var volumeCSS = css.PrepareClass("volume", "")

type Container struct {
	gtk.Grid
	NowPlaying *NowPlaying
	Controls   *controls.Container
	Volume     *Volume
}

func NewContainer(parent VisualizerController) *Container {
	nowpl := NewNowPlaying(parent)
	nowpl.Show()

	controls := controls.NewContainer(parent)
	controls.SetHExpand(true)
	controls.SetHAlign(gtk.ALIGN_FILL)
	controls.Show()

	vol := NewVolume(parent)
	vol.Show()
	volumeCSS(vol)

	grid, _ := gtk.GridNew()
	grid.SetRowHomogeneous(true)
	grid.SetColumnHomogeneous(true)
	grid.SetColumnSpacing(5)
	grid.SetHExpand(true)

	grid.Attach(nowpl, 0, 0, 1, 1)    // 1st column
	grid.Attach(controls, 1, 0, 2, 1) // 2nd-3rd; span 2 columns
	grid.Attach(vol, 3, 0, 1, 1)      // 4th column
	grid.Show()

	return &Container{
		Grid:       *grid,
		NowPlaying: nowpl,
		Controls:   controls,
		Volume:     vol,
	}
}
