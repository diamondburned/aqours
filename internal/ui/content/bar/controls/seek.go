package controls

import (
	"fmt"
	"html"
	"math"
	"time"

	"github.com/diamondburned/aqours/internal/durafmt"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

var timePositionCSS = css.PrepareClass("time-position", "")

var timeTotalCSS = css.PrepareClass("time-total", "")

var seekBarCSS = css.PrepareClass("seek-bar", ``)

var CleanScaleCSS = css.PrepareClass("clean-scale", `
	.clean-scale {
		margin: -2px 0px;
	}
	.clean-scale trough,
	.clean-scale highlight {
		border-radius: 9999px;
	}
	.clean-scale slider {
		padding:    1px;
		background: none;
		transition: linear 75ms background;
	}
	.clean-scale:hover slider {
		/* Shitty hack to limit background size. Thanks, GTK. */
		background: radial-gradient(
			circle,
			@theme_selected_bg_color 0%,
			@theme_selected_bg_color 25%,
			transparent 30%,
			transparent
		);
	}
`)

var seekCSS = css.PrepareClass("seek", "")

const updateSeekEvery = 4 // update once every 4 spins

type Seek struct {
	gtk.Box
	Position  *gtk.Label
	SeekBar   *gtk.Scale
	TotalTime *gtk.Label

	adj   *gtk.Adjustment
	total float64 // rounded
}

func NewSeek(parent ParentController) *Seek {
	pos := gtk.NewLabel("")
	pos.SetSingleLineMode(true)
	pos.SetWidthChars(5)

	timePositionCSS(pos)

	time := gtk.NewLabel("")
	time.SetSingleLineMode(true)
	time.SetWidthChars(5)

	timeTotalCSS(time)

	adj := gtk.NewAdjustment(0, 0, 1, 1, 1, 0)

	bar := gtk.NewScale(gtk.OrientationHorizontal, adj)
	bar.SetDrawValue(false)
	bar.SetVAlign(gtk.AlignCenter)
	bar.SetHExpand(true)
	CleanScaleCSS(bar)
	seekBarCSS(bar)

	bar.Connect("change-value", func(_ *gtk.Scale, _ gtk.ScrollType, v float64) {
		parent.Seek(v)
	})

	box := gtk.NewBox(gtk.OrientationHorizontal, 0)
	box.Append(pos)
	box.Append(bar)
	box.Append(time)

	seekCSS(box)

	return &Seek{
		Box:       *box,
		Position:  pos,
		SeekBar:   bar,
		TotalTime: time,

		adj: adj,
	}
}

const secondFloat = float64(time.Second)

func (s *Seek) UpdatePosition(pos, total float64) {
	s.setTotal(math.Round(total))
	s.adj.SetValue(math.Min(pos, s.total))

	posDuration := time.Duration(pos * secondFloat)
	s.Position.SetMarkup(smallText(durafmt.Format(posDuration)))
}

func (s *Seek) setTotal(total float64) {
	if s.total != total {
		s.total = total

		s.adj.SetUpper(total)
		s.adj.SetPageIncrement(total / 10)
		s.adj.SetStepIncrement(total / 100)

		totalDuration := time.Duration(total * secondFloat)
		s.TotalTime.SetMarkup(smallText(durafmt.Format(totalDuration)))
	}
}

func smallText(text string) string {
	return fmt.Sprintf(
		`<span size="small" alpha="80%%">%s</span>`,
		html.EscapeString(text),
	)
}
