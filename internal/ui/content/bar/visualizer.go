package bar

import (
	"image/color"
	"log"

	"github.com/diamondburned/catnip-gtk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"

	_ "github.com/noriah/catnip/input/ffmpeg"
	_ "github.com/noriah/catnip/input/parec"
	_ "github.com/noriah/catnip/input/portaudio"
)

type Visualizer struct {
	*Container
	Drawer *catnip.Drawer
}

func NewVisualizer(parent ParentController) *Visualizer {
	container := NewContainer(parent)
	container.Show()

	config := catnip.NewConfig()
	config.SampleRate = 44100
	config.SampleSize = int(config.SampleRate / 60) // 70fps
	config.Backend = "parec"
	config.BarWidth = 4
	config.SpaceWidth = 1
	config.SmoothFactor = 39.29
	config.MinimumClamp = 4
	config.Scaling.StaticScale = 12.00 // magic number!
	// config.Monophonic = true

	drawer := initializeCatnip(container, config)
	drawer.SetPaused(true)

	return &Visualizer{
		Container: container,
		Drawer:    drawer,
	}
}

func initializeCatnip(container *Container, config catnip.Config) *catnip.Drawer {
	// Make the foreground transparent.
	styleCtx, _ := container.GetStyleContext()
	foregroundC := styleCtx.GetColor(gtk.STATE_FLAG_NORMAL).Floats()

	config.ForegroundColor = color.RGBA{
		R: uint8(foregroundC[0] * 0xFF),
		G: uint8(foregroundC[1] * 0xFF),
		B: uint8(foregroundC[2] * 0xFF),
		A: 255 / 5, // 20%
	}

	drawer := catnip.NewDrawer(container, config)
	drawer.SetWidgetStyle(container)
	drawer.ConnectDestroy(container)
	drawer.ConnectSizeAllocate(container)

	hID, _ := drawer.ConnectDraw(container)
	destroyed := false

	// Mark the container as destroyed. This way, the handler doesn't get
	// disconnected when the container is gone.
	container.Connect("destroy", func() { destroyed = true })

	go func() {
		if err := drawer.Start(); err != nil {
			log.Println("failed to start catnip:", err)
		}

		glib.IdleAdd(func() {
			// We should only disconnect if the container is not destroyed.
			if !destroyed {
				container.HandlerDisconnect(hID)
			}
		})

		// This function can be called multiple times, so whatever.
		drawer.Stop()
	}()

	return drawer
}
