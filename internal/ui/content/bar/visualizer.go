package bar

// PausedSetter is an interface that catnip.Drawer satisfies.
type PausedSetter interface {
	SetPaused(paused bool)
}

type Visualizer struct {
	*Container

	ParentController

	Drawer PausedSetter
	paused bool
}

func NewVisualizer(parent ParentController) *Visualizer {
	v := &Visualizer{
		ParentController: parent,
	}

	v.Container = NewContainer(v)
	v.Container.Show()

	v.Drawer = newCatnip(v.Container, "parec", "")
	v.Drawer.SetPaused(true)

	return v
}

// SetPaused sets whether or not the visualizer is paused.
func (v *Visualizer) SetPaused(paused bool) {
	v.paused = paused
	v.SetVisualize(v.Volume.VisualizerStatus())
}

// SetVisualize sets whether or not to draw the visualizer.
func (v *Visualizer) SetVisualize(vis VisualizerStatus) {
	v.Drawer.SetPaused(vis.IsPaused(v.paused))
}
