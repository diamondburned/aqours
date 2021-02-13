package bar

import (
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/gotk3/gotk3/gtk"
)

// PausedSetter is an interface that catnip.Drawer satisfies.
type PausedSetter interface {
	SetPaused(paused bool)
}

type Visualizer struct {
	*Container
	pausedBlurStyles []*gtk.StyleContext

	ParentController

	Drawer PausedSetter
	paused bool
}

var pausedBlurCSS = css.Prepare(`
	.paused-blur { transition: linear 75ms; }
	.paused-blur.paused {
		transition-delay: 2s;
		opacity: 0.35;
	}
	.paused-blur:hover {
		opacity: 1;
	}
`)

func NewVisualizer(parent ParentController) *Visualizer {
	v := &Visualizer{
		ParentController: parent,
	}

	v.Container = NewContainer(v)
	v.Container.Show()

	v.Drawer = newCatnip(v.Container, "parec", "")

	// class: now-playing
	v.pausedBlurStyles = []*gtk.StyleContext{
		css.StyleContext(v.NowPlaying),
		css.StyleContext(v.Controls.Seek),
		css.StyleContext(v.Volume),
	}

	for _, style := range v.pausedBlurStyles {
		style.AddClass("paused-blur")
		style.AddProvider(pausedBlurCSS, gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)
	}

	// Paused by default.
	v.SetPaused(true)

	return v
}

// SetPaused sets whether or not the visualizer is paused.
func (v *Visualizer) SetPaused(paused bool) {
	v.paused = paused
	v.SetVisualize(v.Volume.VisualizerStatus())

	if paused {
		for _, style := range v.pausedBlurStyles {
			style.AddClass("paused")
		}
	} else {
		for _, style := range v.pausedBlurStyles {
			style.RemoveClass("paused")
		}
	}
}

// SetVisualize sets whether or not to draw the visualizer.
func (v *Visualizer) SetVisualize(vis VisualizerStatus) {
	v.Drawer.SetPaused(vis.IsPaused(v.paused))
}
