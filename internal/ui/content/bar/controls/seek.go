package controls

import (
	"fmt"
	"html"
	"time"

	"github.com/diamondburned/aqours/internal/durafmt"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/gotk3/gotk3/gtk"
)

var timePositionCSS = css.PrepareClass("time-position", "")

var timeTotalCSS = css.PrepareClass("time-total", "")

var seekBarCSS = css.PrepareClass("seek-bar", `
	scale.seek-bar {
		margin: -2px 4px;
	}

	scale.seek-bar trough,
	scale.seek-bar highlight {
		border-radius: 9999px;
	}

	scale.seek-bar slider {
		padding:    1px;
		background: none;
		transition: linear 75ms background;
	}

	scale.seek-bar slider:hover,
	scale.seek-bar trough:hover {
		/* Shitty hack to limit background size. Thanks, Gtk. */
		background: radial-gradient(
			circle,
			@theme_selected_bg_color 0%,
			@theme_selected_bg_color 25%,
			transparent 10%,
			transparent
		);
	}
`)

var seekCSS = css.PrepareClass("seek", "")

const (
	seekResolution = 10000
	seekStep       = 250
)

type Seek struct {
	gtk.Box
	Position  *gtk.Label
	SeekBar   *gtk.Scale
	TotalTime *gtk.Label
}

func NewSeek(parent ParentController) *Seek {
	pos, _ := gtk.LabelNew("")
	pos.SetSingleLineMode(true)
	pos.SetWidthChars(5)
	pos.Show()
	timePositionCSS(pos)

	time, _ := gtk.LabelNew("")
	time.SetSingleLineMode(true)
	time.SetWidthChars(5)
	time.Show()
	timeTotalCSS(time)

	bar, _ := gtk.ScaleNewWithRange(gtk.ORIENTATION_HORIZONTAL, 0, seekResolution, seekStep)
	bar.SetDrawValue(false)
	bar.SetVAlign(gtk.ALIGN_CENTER)
	bar.Show()
	seekBarCSS(bar)

	bar.Connect("change-value", func(_ *gtk.Scale, _ gtk.ScrollType, v float64) {
		parent.Seek(v)
	})

	box, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	box.PackStart(pos, false, false, 0)
	box.PackStart(bar, true, true, 0)
	box.PackStart(time, false, false, 0)
	box.Show()
	seekCSS(box)

	return &Seek{
		Box:       *box,
		Position:  pos,
		SeekBar:   bar,
		TotalTime: time,
	}
}

const secondFloat = float64(time.Second)

func (s *Seek) UpdatePosition(pos, total float64) {
	s.SeekBar.SetRange(0, total)
	s.SeekBar.SetValue(pos)

	posDuration := time.Duration(pos * secondFloat)
	totalDuration := time.Duration(total * secondFloat)

	s.Position.SetMarkup(smallText(durafmt.Format(posDuration)))
	s.TotalTime.SetMarkup(smallText(durafmt.Format(totalDuration)))
}

func smallText(text string) string {
	return fmt.Sprintf(
		`<span size="small" alpha="80%%">%s</span>`,
		html.EscapeString(text),
	)
}
