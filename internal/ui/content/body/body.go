package body

import (
	"github.com/diamondburned/aqours/internal/ui/content/body/sidebar"
	"github.com/diamondburned/aqours/internal/ui/content/body/tracks"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
)

type ParentController interface {
	tracks.ParentController
	sidebar.ParentController
}

type Container struct {
	handy.Leaflet
	ParentController

	Sidebar *sidebar.Container

	RightStack *gtk.Stack
	TracksView *tracks.Container
}

func NewContainer(parent ParentController) *Container {
	c := &Container{ParentController: parent}

	c.Sidebar = sidebar.NewContainer(c)
	c.Sidebar.SetHExpand(false)
	c.Sidebar.Show()

	sideSeparator, _ := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	sideSeparator.Show()

	c.TracksView = tracks.NewContainer(c)
	c.TracksView.SetHExpand(true)
	c.TracksView.Show()

	idleIcon, _ := gtk.ImageNewFromIconName("folder-music-symbolic", gtk.ICON_SIZE_DIALOG)
	idleIcon.Show()
	idleBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	idleBox.Add(idleIcon)
	idleBox.SetVAlign(gtk.ALIGN_CENTER)
	idleBox.SetHAlign(gtk.ALIGN_CENTER)
	idleBox.Show()

	c.RightStack, _ = gtk.StackNew()
	c.RightStack.AddNamed(idleBox, "idle")
	c.RightStack.AddNamed(c.TracksView, "tracks")
	c.RightStack.Show()

	c.Leaflet = *handy.LeafletNew()
	c.Leaflet.SetCanSwipeBack(true)
	c.Leaflet.SetCanSwipeForward(true)
	c.Leaflet.SetTransitionType(handy.LeafletTransitionTypeSlide)

	c.Leaflet.Add(c.Sidebar)
	c.Leaflet.Add(sideSeparator)
	c.Leaflet.Add(c.RightStack)

	c.Leaflet.ChildSetProperty(sideSeparator, "navigatable", false)
	c.Leaflet.Show()

	return c
}

func (c *Container) SwipeBack() {
	c.Leaflet.SetVisibleChild(c.Sidebar)
}

func (c *Container) SelectPlaylist(path string) {
	if path == "" {
		c.RightStack.SetVisibleChildName("idle")
	} else {
		c.RightStack.SetVisibleChildName("tracks")
	}

	c.ParentController.SelectPlaylist(path)
}
