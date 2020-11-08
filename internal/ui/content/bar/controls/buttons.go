package controls

import (
	"github.com/diamondburned/aqours/internal/muse"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
)

func newIconImage(symbolicName string) *gtk.Image {
	image, _ := gtk.ImageNewFromIconName(symbolicName, gtk.ICON_SIZE_BUTTON)
	image.Show()
	return image
}

var playbackButtonCSS = css.PrepareClass("playback-button", `
	button {
		margin: 2px 8px;
		margin-top: 12px;

		color:   @theme_fg_color;
		opacity: 0.5;

		box-shadow: none;
		background: none;
	}

	button:hover {
		opacity: 1;
	}
`)

var prevCSS = css.PrepareClass("previous", ``)

var nextCSS = css.PrepareClass("next", ``)

var playPauseCSS = css.PrepareClass("playpause", `
	button {
		opacity: 0.75;
		border: 1px solid alpha(@theme_fg_color, 0.45);
	}
	button:hover {
		border: 1px solid alpha(@theme_fg_color, 0.85);
	}
`)

var repeatShuffleButtonCSS = css.PrepareClass("repeat-shuffle", `
	button:checked {
		color:   @theme_selected_bg_color;
		opacity: 0.75;
	}
	button:checked:hover {
		opacity: 1;
	}
`)

type Buttons struct {
	gtk.Box
	Shuffle *gtk.ToggleButton
	Prev    *gtk.Button
	Play    *PlayPause // Pause
	Next    *gtk.Button
	Repeat  *Repeat
}

func NewButtons(parent ParentController) *Buttons {
	shuf, _ := gtk.ToggleButtonNew()
	shuf.SetRelief(gtk.RELIEF_NONE)
	shuf.SetImage(newIconImage("media-playlist-shuffle-symbolic"))
	shuf.SetVAlign(gtk.ALIGN_CENTER)
	shuf.Connect("toggled", func() { parent.SetShuffle(shuf.GetActive()) })
	shuf.Show()
	playbackButtonCSS(shuf)
	repeatShuffleButtonCSS(shuf)

	prev, _ := gtk.ButtonNew()
	prev.SetRelief(gtk.RELIEF_NONE)
	prev.SetImage(newIconImage("go-first-symbolic"))
	prev.SetVAlign(gtk.ALIGN_CENTER)
	prev.Connect("clicked", parent.Previous)
	prev.Show()
	playbackButtonCSS(prev)
	prevCSS(prev)

	pp := NewPlayPause(parent)
	pp.Show()
	playbackButtonCSS(pp)
	playPauseCSS(pp)

	next, _ := gtk.ButtonNew()
	next.SetRelief(gtk.RELIEF_NONE)
	next.SetImage(newIconImage("go-last-symbolic"))
	next.SetVAlign(gtk.ALIGN_CENTER)
	next.Connect("clicked", parent.Next)
	next.Show()
	playbackButtonCSS(next)
	nextCSS(next)

	repeat := NewRepeat(parent)
	repeat.Show()
	playbackButtonCSS(repeat)
	repeatShuffleButtonCSS(repeat)

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.PackStart(shuf, false, false, 0)
	box.PackStart(prev, false, false, 0)
	box.PackStart(pp, false, false, 0)
	box.PackStart(next, false, false, 0)
	box.PackStart(repeat, false, false, 0)
	box.Show()

	return &Buttons{
		Box:     *box,
		Shuffle: shuf,
		Prev:    prev,
		Play:    pp,
		Next:    next,
		Repeat:  repeat,
	}
}

// SetShuffle controls the shuffle button's state and triggers the callback.
func (b *Buttons) SetShuffle(shuffle bool) {
	b.Shuffle.SetActive(shuffle)
}

// SetRepeat sets Repeat's mode. It does NOT trigger a callback to the parent.
func (b *Buttons) SetRepeat(mode muse.RepeatMode, callback bool) {
	b.Repeat.SetRepeat(mode, callback)
}

type Repeat struct {
	gtk.ToggleButton
	parent   ParentController
	handleID glib.SignalHandle

	state      muse.RepeatMode
	icon       *gtk.Image
	singleIcon *gtk.Image
}

func NewRepeat(parent ParentController) *Repeat {
	icon := newIconImage("media-playlist-repeat-symbolic")
	icon.Show()
	singleIcon := newIconImage("media-playlist-repeat-song-symbolic")
	singleIcon.Show()

	button, _ := gtk.ToggleButtonNew()
	button.SetRelief(gtk.RELIEF_NONE)
	button.SetVAlign(gtk.ALIGN_CENTER)
	button.SetImage(icon)
	button.SetActive(false)
	button.Show()

	repeat := &Repeat{
		ToggleButton: *button,
		parent:       parent,
		state:        muse.RepeatNone,
		icon:         icon,
		singleIcon:   singleIcon,
	}

	repeat.handleID, _ = button.Connect("toggled", repeat.CycleState)

	return repeat
}

func (r *Repeat) SetRepeat(mode muse.RepeatMode, callback bool) {
	// We should disable the handler here, as we don't want the callback to have
	// a feedback loop.
	if !callback {
		r.HandlerBlock(r.handleID)
		defer r.HandlerUnblock(r.handleID)
	}

	r.state = mode

	switch mode {
	case muse.RepeatNone:
		r.SetActive(false)
		r.SetImage(r.icon)

	case muse.RepeatSingle:
		r.SetActive(true)
		r.SetImage(r.singleIcon)

	case muse.RepeatAll:
		r.SetActive(true)
		r.SetImage(r.icon)
	}
}

func (r *Repeat) CycleState() {
	r.parent.SetRepeat(r.state.Cycle())
}

type PlayPause struct {
	gtk.Button
	playing bool

	playIcon  *gtk.Image
	pauseIcon *gtk.Image
}

func NewPlayPause(parent ParentController) *PlayPause {
	play := newIconImage("media-playback-start-symbolic")
	play.Show()
	pause := newIconImage("media-playback-pause-symbolic")
	pause.Show()

	pp := &PlayPause{
		playIcon:  play,
		pauseIcon: pause,
	}

	btn, _ := gtk.ButtonNew()
	btn.SetRelief(gtk.RELIEF_NONE)
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
