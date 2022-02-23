package body

import (
	"log"

	"github.com/diamondburned/aqours/internal/ui/content/body/sidebar"
	"github.com/diamondburned/aqours/internal/ui/content/body/tracks"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type ParentController interface {
	tracks.ParentController
	sidebar.ParentController
}

type Container struct {
	*gtk.Box
	ParentController

	Sidebar *sidebar.Container

	RightStack *gtk.Stack
	TracksView *tracks.Container
}

func NewContainer(parent ParentController) *Container {
	c := &Container{ParentController: parent}

	c.Sidebar = sidebar.NewContainer(c)
	c.Sidebar.SetHExpand(false)

	sideSeparator := gtk.NewSeparator(gtk.OrientationVertical)

	c.TracksView = tracks.NewContainer(c)
	c.TracksView.SetHExpand(true)

	idleIcon := gtk.NewImageFromIconName("folder-music-symbolic")

	idleBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	idleBox.Append(idleIcon)
	idleBox.SetVAlign(gtk.AlignCenter)
	idleBox.SetHAlign(gtk.AlignCenter)

	c.RightStack = gtk.NewStack()
	c.RightStack.AddNamed(idleBox, "idle")
	c.RightStack.AddNamed(c.TracksView, "tracks")

	c.Box = gtk.NewBox(gtk.OrientationHorizontal, 0)
	c.Box.Append(c.Sidebar)
	c.Box.Append(sideSeparator)
	c.Box.Append(c.RightStack)

	return c
}

func (c *Container) SwipeBack() {
	log.Println("TODO: SwipeBack(): REMOVE ME")
	// c.Leaflet.SetVisibleChild(c.Sidebar)
}

func (c *Container) SelectPlaylist(path string) {
	if path == "" {
		c.RightStack.SetVisibleChildName("idle")
	} else {
		c.RightStack.SetVisibleChildName("tracks")
	}

	c.ParentController.SelectPlaylist(path)
}
