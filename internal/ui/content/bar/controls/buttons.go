package controls

import (
	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

var playbackButtonCSS = css.PrepareClass("playback-button", `
	.playback-button {
		margin: 2px 8px;
		margin-top: 12px;
		border-radius: 9999px;

		color:   @theme_fg_color;
		opacity: 0.5;

		box-shadow: none;
		background: none;
	}
	.playback-button:hover {
		opacity: 1;
	}
`)

var prevCSS = css.PrepareClass("previous", ``)

var nextCSS = css.PrepareClass("next", ``)

var playPauseCSS = css.PrepareClass("playpause", `
	.playpause {
		opacity: 0.75;
		border: 1px solid alpha(@theme_fg_color, 0.45);
	}
	.playpause:hover {
		border: 1px solid alpha(@theme_fg_color, 0.85);
	}
`)

var repeatShuffleButtonCSS = css.PrepareClass("repeat-shuffle", `
	.repeat-shuffle:checked {
		color:   @theme_selected_bg_color;
		opacity: 0.8;
	}
	.repeat-shuffle:hover {
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
	shuf := gtk.NewToggleButton()
	shuf.SetChild(gtk.NewImageFromIconName("media-playlist-shuffle-symbolic"))
	shuf.SetVAlign(gtk.AlignCenter)
	shuf.ConnectToggled(func() { parent.SetShuffle(shuf.Active()) })
	playbackButtonCSS(shuf)
	repeatShuffleButtonCSS(shuf)

	prev := gtk.NewButton()
	prev.SetChild(gtk.NewImageFromIconName("media-skip-backward"))
	prev.SetVAlign(gtk.AlignCenter)
	prev.ConnectClicked(parent.Previous)
	playbackButtonCSS(prev)
	prevCSS(prev)

	pp := NewPlayPause(parent)
	playbackButtonCSS(pp)
	playPauseCSS(pp)

	next := gtk.NewButton()
	next.SetChild(gtk.NewImageFromIconName("media-skip-forward"))
	next.SetVAlign(gtk.AlignCenter)
	next.ConnectClicked(parent.Next)
	playbackButtonCSS(next)
	nextCSS(next)

	repeat := NewRepeat(parent)
	playbackButtonCSS(repeat)
	repeatShuffleButtonCSS(repeat)

	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	box.Append(shuf)
	box.Append(prev)
	box.Append(pp)
	box.Append(next)
	box.Append(repeat)

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
func (b *Buttons) SetRepeat(mode state.RepeatMode, callback bool) {
	b.Repeat.SetRepeat(mode, callback)
}

type Repeat struct {
	gtk.ToggleButton
	parent   ParentController
	handleID glib.SignalHandle

	state      state.RepeatMode
	icon       *gtk.Image
	singleIcon *gtk.Image
}

func NewRepeat(parent ParentController) *Repeat {
	icon := gtk.NewImageFromIconName("media-playlist-repeat-symbolic")
	singleIcon := gtk.NewImageFromIconName("media-playlist-repeat-song-symbolic")

	button := gtk.NewToggleButton()
	button.SetVAlign(gtk.AlignCenter)
	button.SetChild(icon)
	button.SetActive(false)

	repeat := &Repeat{
		ToggleButton: *button,
		parent:       parent,
		state:        state.RepeatNone,
		icon:         icon,
		singleIcon:   singleIcon,
	}

	repeat.handleID = button.Connect("toggled", func() {
		parent.SetRepeat(repeat.state.Cycle())
	})

	return repeat
}

func (r *Repeat) SetRepeat(mode state.RepeatMode, callback bool) {
	// We should disable the handler here, as we don't want the callback to have
	// a feedback loop.
	if !callback {
		r.HandlerBlock(r.handleID)
		defer r.HandlerUnblock(r.handleID)
	}

	r.state = mode

	switch mode {
	case state.RepeatNone:
		r.SetActive(false)
		r.SetChild(r.icon)

	case state.RepeatSingle:
		r.SetActive(true)
		r.SetChild(r.singleIcon)

	case state.RepeatAll:
		r.SetActive(true)
		r.SetChild(r.icon)
	}
}

type PlayPause struct {
	gtk.Button
	playing bool

	playIcon  *gtk.Image
	pauseIcon *gtk.Image
}

func NewPlayPause(parent ParentController) *PlayPause {
	play := gtk.NewImageFromIconName("media-playback-start-symbolic")
	pause := gtk.NewImageFromIconName("media-playback-pause-symbolic")

	pp := &PlayPause{
		playIcon:  play,
		pauseIcon: pause,
	}

	btn := gtk.NewButton()
	btn.SetChild(pause)
	btn.SetVAlign(gtk.AlignCenter)

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
		pp.SetChild(pp.pauseIcon)
		pp.SetTooltipText("Pause")
	} else {
		pp.SetChild(pp.playIcon)
		pp.SetTooltipText("Play")
	}
}
