package content

import (
	"github.com/diamondburned/aqours/internal/ui/content/bar"
	"github.com/diamondburned/aqours/internal/ui/content/body"
	"github.com/gotk3/gotk3/gtk"
)

type ParentController interface {
	body.ParentController
	bar.ParentController
}

type Container struct {
	ContentBox *gtk.Box

	Body *body.Container
	Bar  *bar.Container
	Vis  *bar.Visualizer
}

func NewContainer(parent ParentController) Container {
	body := body.NewContainer(parent)
	body.Show()

	separator, _ := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	separator.Show()

	vis := bar.NewVisualizer(parent)
	vis.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.PackStart(body, true, true, 0)
	box.PackStart(separator, false, false, 0)
	box.PackStart(vis, false, false, 0)
	box.SetHExpand(true)
	box.Show()

	return Container{
		ContentBox: box,
		Body:       body,
		Bar:        vis.Container,
		Vis:        vis,
	}
}
