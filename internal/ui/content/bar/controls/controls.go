package controls

import (
	"github.com/diamondburned/aqours/internal/muse"
	"github.com/gotk3/gotk3/gtk"
)

type ParentController interface {
	Previous()
	Next()
	Seek(position float64)
	SetPlay(playing bool)
	SetRepeat(repeatMode muse.RepeatMode)
	SetShuffle(shuffle bool)
}

type Container struct {
	gtk.Box
	Buttons *Buttons
	Seek    *Seek
}

func NewContainer(parent ParentController) *Container {
	buttons := NewButtons(parent)
	buttons.SetHAlign(gtk.ALIGN_CENTER)
	buttons.Show()

	seek := NewSeek(parent)
	seek.SetHAlign(gtk.ALIGN_FILL)
	seek.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.PackStart(buttons, false, false, 0)
	box.PackStart(seek, false, false, 0)
	box.Show()

	return &Container{
		Box:     *box,
		Buttons: buttons,
		Seek:    seek,
	}
}
