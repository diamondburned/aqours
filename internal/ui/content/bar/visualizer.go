package bar

// PausedSetter is an interface that catnip.Drawer satisfies.
type PausedSetter interface {
	SetPaused(paused bool)
}

type Visualizer struct {
	*Container
	Drawer PausedSetter
}

func NewVisualizer(parent ParentController) *Visualizer {
	container := NewContainer(parent)
	container.Show()

	drawer := newCatnip(container, "parec", "")
	drawer.SetPaused(true)

	return &Visualizer{
		Container: container,
		Drawer:    drawer,
	}
}
