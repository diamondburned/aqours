// +build !catnip

package bar

// HasVisualizer is false, indicating aqours was not built with catnip.
const HasVisualizer = false

func newCatnip(c *Container, backend, device string) PausedSetter {
	return nullVisualizer{}
}

type nullVisualizer struct{}

func (nullVisualizer) SetPaused(paused bool) {}

func (nullVisualizer) stub() {}
