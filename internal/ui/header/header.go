// Package header is the top bar that contains the logo, buttons, and the
// playlist name.
package header

import (
	"fmt"
	"math"

	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
)

type ParentController interface {
	gtk.IWindow
	AddPlaylist(path string)

	// ParentPlaylistController methods.
	GoBack()
	HasPlaylist(name string) bool
	SavePlaylist(pl *state.Playlist)
	RenamePlaylist(pl *state.Playlist, newName string) bool
}

type Container struct {
	gtk.HeaderBar
	ParentController

	Left *AppControls
	Info *PlaylistInfo

	RightSide *handy.Leaflet
	Bitrate   *gtk.Label
	Right     *PlaylistControls

	current *state.Playlist
}

func NewContainer(parent ParentController) *Container {
	c := &Container{ParentController: parent}

	c.Left = NewAppControls(parent)
	c.Left.Show()

	c.Info = NewPlaylistInfo()
	c.Info.Show()

	c.Bitrate, _ = gtk.LabelNew("")
	c.Bitrate.SetSingleLineMode(true)
	c.Bitrate.Show()

	c.Right = NewPlaylistControls(c)
	c.Right.Show()

	c.RightSide = handy.LeafletNew()
	c.RightSide.SetTransitionType(handy.LeafletTransitionTypeSlide)
	c.RightSide.Add(c.Bitrate)
	c.RightSide.Add(c.Right)
	c.RightSide.SetVisibleChild(c.Right)
	c.RightSide.Show()

	empty, _ := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 0)

	h, _ := gtk.HeaderBarNew()
	h.SetCustomTitle(empty)
	h.PackStart(c.Left)
	h.PackStart(c.Info)
	h.PackEnd(c.RightSide)
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

// SetBitrate sets the bitrate to display. The indicator is empty if bits is
// less than 0.
func (c *Container) SetBitrate(bits float64) {
	if bits < 0 {
		c.Bitrate.SetText("")
		return
	}

	c.Bitrate.SetMarkup(fmt.Sprintf(
		`<span size="small"><i>%g kbits/s</i></span>`, math.Round(bits/1000),
	))
}

// SetUnsaved sets the header info to display the name as unchanged if the
// given playlist is indeed being displayed. It does nothing otherwise.
func (c *Container) SetUnsaved(pl *state.Playlist) {
	if pl == c.current {
		c.Info.SetUnsaved(pl.IsUnsaved())
	}
}

func (c *Container) SetPlaylist(pl *state.Playlist) {
	c.current = pl

	if pl != nil {
		c.Info.SetPlaylist(pl)
		c.Right.SetRevealChild(true)
	} else {
		c.Info.Reset()
		c.Right.SetRevealChild(false)
	}
}

func (c *Container) SaveCurrentPlaylist() {
	c.SavePlaylist(c.current)
}

// RenamePlaylist calls the parent's RenamePlaylist with the current name.
func (c *Container) RenamePlaylist(newName string) {
	renamed := c.ParentController.RenamePlaylist(c.current, newName)
	if renamed {
		c.Info.SetPlaylist(c.current)
	}
}

// PlaylistName returns the current playlist, or an empty string if none.
func (c *Container) PlaylistName() string {
	return c.Info.Playlist
}
