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
	gtk.Box

	Body *body.Container
	Bar  *bar.Container
}

func NewContainer(parent ParentController) *Container {
	body := body.NewContainer(parent)
	body.Show()

	separator, _ := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	separator.Show()

	bar := bar.NewContainer(parent)
	bar.Show()

	box, _ := gtk.BoxNew(gtk.ORIENTATION_VERTICAL, 0)
	box.PackStart(body, true, true, 0)
	box.PackStart(separator, false, false, 0)
	box.PackStart(bar, false, false, 0)
	box.Show()

	return &Container{
		Box:  *box,
		Body: body,
		Bar:  bar,
	}
}
