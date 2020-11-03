package controls

import (
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/gotk3/gotk3/gtk"
)

type ParentController interface {
	Previous()
	SetPlay(playing bool)
	Next()
	Seek(position float64)
}

var playbackButtonCSS = css.PrepareClass("playback-button", `
	button.playback-button {
		margin: 2px 8px;
		margin-top: 12px;
		opacity: 0.65;
		box-shadow: none;
		background: none;
	}

	button.playback-button:hover {
		opacity: 1;
	}
`)

var prevCSS = css.PrepareClass("previous", ``)

var nextCSS = css.PrepareClass("next", ``)

var playPauseCSS = css.PrepareClass("playpause", `
	button.playpause {
		border: 1px solid alpha(@theme_fg_color, 0.35);
	}
	button.playpause:hover {
		border: 1px solid alpha(@theme_fg_color, 0.55);
	}
`)

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
