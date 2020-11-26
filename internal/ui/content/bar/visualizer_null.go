// +build nocatnip

package bar

func newCatnip(c *Container, backend, device string) PausedSetter {
	return nullVisualizer{}
}

type nullVisualizer struct{}

func (nullVisualizer) SetPaused(paused bool) {}
