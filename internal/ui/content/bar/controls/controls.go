package controls

import (
	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type ParentController interface {
	Previous()
	Next()
	Seek(position float64)
	SetPlay(playing bool)
	SetRepeat(repeatMode state.RepeatMode)
	SetShuffle(shuffle bool)
}

type Container struct {
	gtk.Box
	Buttons *Buttons
	Seek    *Seek
}

func NewContainer(parent ParentController) *Container {
	buttons := NewButtons(parent)
	buttons.SetHAlign(gtk.AlignCenter)

	seek := NewSeek(parent)
	seek.SetHAlign(gtk.AlignFill)

	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.Append(buttons)
	box.Append(seek)

	return &Container{
		Box:     *box,
		Buttons: buttons,
		Seek:    seek,
	}
}
