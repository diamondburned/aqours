// +build !nocatnip

package bar

import (
	"image/color"
	"log"

	"github.com/diamondburned/catnip-gtk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"gonum.org/v1/gonum/dsp/window"

	_ "github.com/noriah/catnip/input/ffmpeg"
	_ "github.com/noriah/catnip/input/parec"
	_ "github.com/noriah/catnip/input/portaudio"
)

// FrameRate is the frame rate for the visualizer. The higher it is, the less
// accurate the visualization is.
const FrameRate = 50

func newCatnip(container *Container, backend, device string) PausedSetter {
	config := catnip.NewConfig()
	config.SampleRate = 22050
	config.SampleSize = 22050 / FrameRate
	config.Backend = backend // TODO: FIXME
	config.Device = device
	config.BarWidth = 4      // decent size
	config.SpaceWidth = 1    // decent size
	config.SmoothFactor = 50 // magic number!
	config.MinimumClamp = 4  // hide bars that are too low
	config.ForceEven = true  // sharpen the bars
	config.Symmetry = catnip.Horizontal
	config.WindowFn = catnip.WrapExternalWindowFn(window.Blackman)
	config.Monophonic = false

	// Make the foreground transparent.
	styleCtx, _ := container.GetStyleContext()
	foregroundC := styleCtx.GetColor(gtk.STATE_FLAG_NORMAL).Floats()

	config.Colors.Foreground = color.RGBA{
		R: uint8(foregroundC[0] * 0xFF),
		G: uint8(foregroundC[1] * 0xFF),
		B: uint8(foregroundC[2] * 0xFF),
		A: 255 / 6, // 16.7%
	}

	drawer := catnip.NewDrawer(container, config)
	drawer.SetWidgetStyle(container)
	drawer.ConnectDestroy(container)

	hID, _ := drawer.ConnectDraw(container)
	destroyed := false

	// Mark the container as destroyed. This way, the handler doesn't get
	// disconnected when the container is gone.
	container.Connect("destroy", func() { destroyed = true })

	go func() {
		if err := drawer.Start(); err != nil {
			log.Println("failed to start catnip:", err)
		}

		// This function can be called multiple times, so whatever.
		drawer.Stop()

		glib.IdleAdd(func() {
			// We should only disconnect if the container is not destroyed.
			if !destroyed {
				container.HandlerDisconnect(hID)
			}
		})
	}()

	return drawer
}
