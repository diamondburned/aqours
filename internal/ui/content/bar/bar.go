// Package bar contains the control bar.
package bar

import (
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/gotk3/gotk3/gtk"
)

type ParentController interface {
	Previous()
	SetPlay(playing bool)
	Next()
}

var nowPlayingCSS = css.PrepareClass("now-playing", `
	box.now-playing {
		margin: 8px;
	}
`)

type Container struct {
	gtk.Box

	// TODO: seek bar
	NowPlaying *NowPlaying
	Controls   *Controls
}

func NewContainer(parent ParentController) *Container {
	nowpl := NewNowPlaying()
	nowPlayingCSS(nowpl)
	nowpl.Show()

	controls := NewControls(parent)
	controls.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.PackStart(nowpl, false, false, 0)
	box.PackStart(controls, false, false, 0)
	box.Show()

	return &Container{
		Box:        *box,
		NowPlaying: nowpl,
		Controls:   controls,
	}
}
