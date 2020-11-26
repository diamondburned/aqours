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

var seekBarCSS = css.PrepareClass("seek-bar", ``)

var CleanScaleCSS = css.PrepareClass("clean-scale", `
	scale {
		margin: -2px 0px;
	}

	scale trough,
	scale highlight {
		border-radius: 9999px;
	}

	scale slider {
		padding:    1px;
		background: none;
		transition: linear 75ms background;
	}

	scale:hover slider {
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

type Seek struct {
	gtk.Box
	Position  *gtk.Label
	SeekBar   *gtk.Scale
	TotalTime *gtk.Label

	total float64
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

	bar, _ := gtk.ScaleNewWithRange(gtk.ORIENTATION_HORIZONTAL, 0, 1, 1)
	bar.SetDrawValue(false)
	bar.SetVAlign(gtk.ALIGN_CENTER)
	bar.Show()
	CleanScaleCSS(bar)
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
	s.setTotal(total)
	s.SeekBar.SetValue(pos)

	posDuration := time.Duration(pos * secondFloat)
	totalDuration := time.Duration(total * secondFloat)

	s.Position.SetMarkup(smallText(durafmt.Format(posDuration)))
	s.TotalTime.SetMarkup(smallText(durafmt.Format(totalDuration)))
}

func (s *Seek) setTotal(total float64) {
	if s.total != total {
		s.total = total

		// Pretend that total is 1 if it is 0.
		if total == 0 {
			total = 1
		}

		s.SeekBar.SetRange(0, total)
		s.SeekBar.SetIncrements(total/100, total/10)
	}
}

func smallText(text string) string {
	return fmt.Sprintf(
		`<span size="small" alpha="80%%">%s</span>`,
		html.EscapeString(text),
	)
}
