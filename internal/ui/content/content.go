package content

import (
	"github.com/diamondburned/aqours/internal/ui/content/bar"
	"github.com/diamondburned/aqours/internal/ui/content/body"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type ParentController interface {
	body.ParentController
	bar.ParentController
}

type Container struct {
	ContentBox *gtk.Box

	Body *body.Container
	Bar  *bar.Container
}

func NewContainer(parent ParentController) Container {
	body := body.NewContainer(parent)
	body.SetHExpand(true)

	separator := gtk.NewSeparator(gtk.OrientationVertical)

	bar := bar.NewContainer(parent)

	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.SetHExpand(true)
	box.Append(body)
	box.Append(separator)
	box.Append(bar)

	return Container{
		ContentBox: box,
		Body:       body,
		Bar:        bar,
	}
}
