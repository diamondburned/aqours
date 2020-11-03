package bar

import (
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/gotk3/gotk3/gtk"
)

var playbackButtonCSS = css.PrepareClass("playback-button", `
	button.playback-button {
		margin: 16px 8px;
		margin-left: 0px;
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

type Controls struct {
	gtk.Box
	Prev *gtk.Button
	Play *PlayPause // Pause
	Next *gtk.Button
}

func NewControls(parent ParentController) *Controls {
	prev, _ := gtk.ButtonNewFromIconName("go-first-symbolic", gtk.ICON_SIZE_BUTTON)
	prev.SetVAlign(gtk.ALIGN_CENTER)
	prev.Connect("clicked", parent.Previous)
	prev.Show()
	prevCSS(prev)
	playbackButtonCSS(prev)

	pp := NewPlayPause(parent)
	pp.Show()
	playPauseCSS(pp)
	playbackButtonCSS(pp)

	next, _ := gtk.ButtonNewFromIconName("go-last-symbolic", gtk.ICON_SIZE_BUTTON)
	next.SetVAlign(gtk.ALIGN_CENTER)
	next.Connect("clicked", parent.Next)
	next.Show()
	nextCSS(next)
	playbackButtonCSS(next)

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.PackStart(prev, false, false, 0)
	box.PackStart(pp, false, false, 0)
	box.PackStart(next, false, false, 0)
	box.Show()

	return &Controls{
		Box:  *box,
		Prev: prev,
		Play: pp,
		Next: next,
	}
}

type PlayPause struct {
	gtk.Button
	playing bool

	playIcon  *gtk.Image
	pauseIcon *gtk.Image
}

func NewPlayPause(parent ParentController) *PlayPause {
	play, _ := gtk.ImageNewFromIconName("media-playback-start-symbolic", gtk.ICON_SIZE_BUTTON)
	play.Show()
	pause, _ := gtk.ImageNewFromIconName("media-playback-pause-symbolic", gtk.ICON_SIZE_BUTTON)
	pause.Show()

	pp := &PlayPause{
		playIcon:  play,
		pauseIcon: pause,
	}

	btn, _ := gtk.ButtonNew()
	btn.SetImage(pause)
	btn.SetVAlign(gtk.ALIGN_CENTER)
	btn.Show()

	pp.Button = *btn

	btn.Connect("clicked", func() { parent.SetPlay(!pp.playing) })

	return pp
}

func (pp *PlayPause) IsPlaying() bool {
	return pp.playing
}

func (pp *PlayPause) SetPlaying(playing bool) {
	pp.playing = playing

	if pp.playing {
		pp.SetImage(pp.pauseIcon)
		pp.SetTooltipText("Pause")
	} else {
		pp.SetImage(pp.playIcon)
		pp.SetTooltipText("Play")
	}
}
