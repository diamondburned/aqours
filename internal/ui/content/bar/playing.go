package bar

import (
	"html"
	"strings"

	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
)

var titleCSS = css.PrepareClass("title", `
	label.now-playing.title {
		color: @theme_fg_color;
		font-weight: bold;
	}
`)

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
	title := gtk.NewLabel("")
	title.SetEllipsize(pango.EllipsizeEnd)
	title.SetXAlign(0)
	nowPlayingCSS(title)
	titleCSS(title)

	subtitle := gtk.NewLabel("")
	subtitle.SetEllipsize(pango.EllipsizeEnd)
	subtitle.SetXAlign(0)
	nowPlayingCSS(subtitle)
	subtitleCSS(subtitle)

	subrev := gtk.NewRevealer()
	subrev.SetTransitionDuration(100)
	subrev.SetTransitionType(gtk.RevealerTransitionTypeSlideDown)
	subrev.SetChild(subtitle)

	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.Append(title)
	box.Append(subrev)
	box.SetVAlign(gtk.AlignCenter)

	btn := gtk.NewButton()
	btn.SetChild(box)
	btn.ConnectClicked(parent.ScrollToPlaying)
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
