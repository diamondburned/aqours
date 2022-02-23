// Package bar contains the control bar.
package bar

import (
	"log"

	"github.com/diamondburned/aqours/internal/ui/content/bar/controls"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type ParentController interface {
	controls.ParentController
	ScrollToPlaying()
	SetVolume(perc float64)
	SetMute(muted bool)
}

type Container struct {
	gtk.Grid
	ParentController

	NowPlaying *NowPlaying
	Controls   *controls.Container
	Volume     *Volume
}

func NewContainer(parent ParentController) *Container {
	c := Container{ParentController: parent}
	c.NowPlaying = NewNowPlaying(parent)

	c.Controls = controls.NewContainer(parent)
	c.Controls.SetHExpand(true)
	c.Controls.SetHAlign(gtk.AlignFill)

	c.Volume = NewVolume(&c)

	grid := gtk.NewGrid()
	grid.SetRowHomogeneous(true)
	grid.SetColumnHomogeneous(true)
	grid.SetColumnSpacing(5)
	grid.SetHExpand(true)

	grid.Attach(c.NowPlaying, 0, 0, 2, 1) // 1st column; 2 columns
	grid.Attach(c.Controls, 3, 0, 3, 1)   // 2nd-3rd;    3 columns
	grid.Attach(c.Volume, 6, 0, 2, 1)     // 4th column; 2 columns

	c.Grid = *grid
	return &c
}

// SetPaused sets the paused state.
func (c *Container) SetPaused(paused bool) {
	// c.Vis.SetPaused(paused)
	c.Controls.Buttons.Play.SetPlaying(!paused)
}

// SetVisualize sets the visualizer status.
func (c *Container) SetVisualize(vis VisualizerStatus) {
	log.Println("visualizer unimplemented")
}
