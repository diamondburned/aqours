package bar

import (
	"html"
	"strings"

	"github.com/diamondburned/aqours/internal/state"
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

var nowPlayingCSS = css.PrepareClass("now-playing", `
	button.now-playing {
		margin:  8px;
		padding: 0;
		background: none;
		box-shadow: none;
		transition: linear 45ms;
	}
	button.now-playing:active {
		margin-bottom: 6px;
	}
`)

type NowPlaying struct {
	gtk.Button
	Container *gtk.Box

	Title     *gtk.Label
	SubReveal *gtk.Revealer
	Subtitle  *gtk.Label
}

func NewNowPlaying(parent ParentController) *NowPlaying {
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

	btn, _ := gtk.ButtonNew()
	btn.SetRelief(gtk.RELIEF_NONE)
	btn.Add(box)
	btn.Show()
	btn.Connect("clicked", parent.ScrollToPlaying)
	nowPlayingCSS(btn)

	np := &NowPlaying{
		Button:    *btn,
		Container: box,
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

func (np *NowPlaying) SetTrack(track *state.Track) {
	metadata := track.Metadata()

	np.Title.SetText(metadata.Title)

	var markup strings.Builder
	if metadata.Artist != "" {
		markup.WriteString(`<span alpha="95%">`)
		markup.WriteString(html.EscapeString(metadata.Artist))
		markup.WriteString("</span>")
	}

	if metadata.Album != "" {
		markup.WriteByte(' ')
		markup.WriteString(`<span alpha="70%" size="small">`)

		if metadata.Artist != "" {
			markup.WriteString("- ")
		}

		markup.WriteString(html.EscapeString(metadata.Album))
		markup.WriteString("</span>")
	}

	np.SubReveal.SetRevealChild(markup.Len() > 0)
	np.Subtitle.SetMarkup(markup.String())
}
