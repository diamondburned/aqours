// Package header is the top bar that contains the logo, buttons, and the
// playlist name.
package header

import "github.com/gotk3/gotk3/gtk"

type ParentController interface {
	gtk.IWindow
	AddPlaylist(path string)

	// ParentPlaylistController methods.
	HasPlaylist(name string) bool
	RenamePlaylist(oldName, newName string) bool
}

type Container struct {
	gtk.HeaderBar
	ParentController

	Left  *AppControls
	Info  *PlaylistInfo
	Right *PlaylistControls
}

func NewContainer(parent ParentController) *Container {
	c := &Container{ParentController: parent}

	c.Left = NewAppControls(parent)
	c.Left.Show()

	c.Info = NewPlaylistInfo()
	c.Info.Show()

	c.Right = NewPlaylistControls(c)
	c.Right.Show()

	empty, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	h, _ := gtk.HeaderBarNew()
	h.PackStart(c.Left)
	h.PackStart(c.Info)
	h.PackEnd(c.Right)
	h.SetCustomTitle(empty)
	h.SetShowCloseButton(true)
	h.Show()
	c.HeaderBar = *h

	c.Reset()

	return c
}

func (c *Container) Reset() {
	c.Info.Reset()
	c.Right.SetRevealChild(false)
}

func (c *Container) SetPlaylist(name string) {
	c.Info.SetPlaylist(name)
	c.Right.SetRevealChild(true)
}

// RenamePlaylist calls the parent's RenamePlaylist with the current name.
func (c *Container) RenamePlaylist(newName string) {
	renamed := c.ParentController.RenamePlaylist(c.Info.Playlist, newName)
	if renamed {
		c.Info.SetPlaylist(newName)
	}
}

// PlaylistName returns the current playlist, or an empty string if none.
func (c *Container) PlaylistName() string {
	return c.Info.Playlist
}
