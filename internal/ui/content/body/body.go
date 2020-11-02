package body

import (
	"time"

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

	RightStack   *gtk.Stack
	TracksScroll *gtk.ScrolledWindow
	TracksView   *tracks.Container
}

func NewContainer(parent ParentController) *Container {
	c := &Container{ParentController: parent}

	c.Sidebar = sidebar.NewContainer(c)
	c.Sidebar.SetHExpand(false)
	c.Sidebar.Show()

	sideSeparator, _ := gtk.SeparatorNew(gtk.ORIENTATION_VERTICAL)
	sideSeparator.Show()

	sideBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	sideBox.PackStart(c.Sidebar, false, false, 0)
	sideBox.PackStart(sideSeparator, false, false, 0)
	sideBox.SetHExpand(false)
	sideBox.Show()

	c.TracksView = tracks.NewContainer(c)
	c.TracksView.SetHExpand(true)
	c.TracksView.Show()

	c.TracksScroll, _ = gtk.ScrolledWindowNew(nil, nil)
	c.TracksScroll.SetPolicy(gtk.POLICY_NEVER, gtk.POLICY_AUTOMATIC)
	c.TracksScroll.SetVExpand(true)
	c.TracksScroll.Add(c.TracksView)
	c.TracksScroll.Show()

	idleIcon, _ := gtk.ImageNewFromIconName("folder-music-symbolic", gtk.ICON_SIZE_DIALOG)
	idleIcon.Show()
	idleBox, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)
	idleBox.Add(idleIcon)
	idleBox.SetVAlign(gtk.ALIGN_CENTER)
	idleBox.SetHAlign(gtk.ALIGN_CENTER)
	idleBox.Show()

	c.RightStack, _ = gtk.StackNew()
	c.RightStack.AddNamed(idleBox, "idle")
	c.RightStack.AddNamed(c.TracksScroll, "tracks")
	c.RightStack.Show()

	c.Leaflet = *handy.LeafletNew()
	c.Leaflet.Add(sideBox)
	c.Leaflet.Add(c.RightStack)
	c.Leaflet.Show()

	leafletOnFold(&c.Leaflet, func(folded bool) {
		if folded {
			sideSeparator.Hide()
		} else {
			sideSeparator.Show()
		}
	})

	return c
}

func (c *Container) SelectPlaylist(path string) {
	if path == "" {
		c.RightStack.SetVisibleChildName("idle")
	} else {
		c.RightStack.SetVisibleChildName("tracks")
	}

	c.ParentController.SelectPlaylist(path)
}

// leafletOnFold binds a callback to a leaflet that would be called when the
// leaflet's folded state changes.
func leafletOnFold(leaflet *handy.Leaflet, foldedFn func(folded bool)) {
	var lastFold = leaflet.GetFolded()
	foldedFn(lastFold)

	// Give each callback a 500ms wait for animations to complete.
	const dt = 500 * time.Millisecond
	var last = time.Now()

	leaflet.ConnectAfter("size-allocate", func() {
		// Ignore if this event is too recent.
		if now := time.Now(); now.Add(-dt).Before(last) {
			return
		} else {
			last = now
		}

		if folded := leaflet.GetFolded(); folded != lastFold {
			lastFold = folded
			foldedFn(folded)
		}
	})
}
