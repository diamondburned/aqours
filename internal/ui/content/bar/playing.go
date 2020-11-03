package bar

import (
	"fmt"
	"html"

	"github.com/diamondburned/aqours/internal/muse/playlist"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
)

var titleCSS = css.PrepareClass("title", "")

var subtitleCSS = css.PrepareClass("subtitle", `
	label.now-playing.subtitle {
		color: mix(@theme_bg_color, @theme_fg_color, 0.8);
	}
`)

type NowPlaying struct {
	gtk.Box
	Title     *gtk.Label
	SubReveal *gtk.Revealer
	Subtitle  *gtk.Label
}

func NewNowPlaying() *NowPlaying {
	title, _ := gtk.LabelNew("")
	title.SetEllipsize(pango.ELLIPSIZE_END)
	title.SetXAlign(0)
	title.Show()
	nowPlayingCSS(title)
	titleCSS(title)

	subtitle, _ := gtk.LabelNew("")
	subtitle.SetEllipsize(pango.ELLIPSIZE_END)
	subtitle.SetXAlign(0)
	subtitle.Show()
	nowPlayingCSS(subtitle)
	subtitleCSS(subtitle)

	subrev, _ := gtk.RevealerNew()
	subrev.SetTransitionDuration(100)
	subrev.SetTransitionType(gtk.REVEALER_TRANSITION_TYPE_SLIDE_DOWN)
	subrev.Add(subtitle)
	subrev.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.PackStart(title, false, false, 0)
	box.PackStart(subrev, false, false, 0)
	box.SetVAlign(gtk.ALIGN_CENTER)
	box.Show()

	np := &NowPlaying{
		Box:       *box,
		Title:     title,
		SubReveal: subrev,
		Subtitle:  subtitle,
	}

	np.StopPlaying()

	return np
}

func (np *NowPlaying) StopPlaying() {
	np.Title.SetLabel("Not playing.")
	np.SubReveal.SetRevealChild(false)
}

func (np *NowPlaying) SetTrack(track *playlist.Track) {
	np.Title.SetLabel(track.Title)
	np.SubReveal.SetRevealChild(true)
	np.Subtitle.SetMarkup(fmt.Sprintf(
		`<span size="small">%s - %s</span>`,
		html.EscapeString(track.Artist), html.EscapeString(track.Album),
	))
}
