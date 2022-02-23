// Package header is the top bar that contains the logo, buttons, and the
// playlist name.
package header

import (
	"fmt"
	"math"

	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type ParentController interface {
	AddPlaylist(path string)
	// ParentPlaylistController methods.
	GoBack()
	HasPlaylist(name string) bool
	SavePlaylist(pl *state.Playlist)
	RenamePlaylist(pl *state.Playlist, newName string) bool
}

var bitrateCSS = css.PrepareClass("bitrate", `
	.bitrate {
		margin: 0 6px;
	}
`)

type Container struct {
	gtk.HeaderBar
	ParentController

	Left *AppControls
	Info *PlaylistInfo

	RightSide *gtk.Box
	Bitrate   *gtk.Label
	Right     *PlaylistControls

	current *state.Playlist
}

func NewContainer(parent ParentController) *Container {
	c := &Container{ParentController: parent}

	c.Left = NewAppControls(parent)

	c.Info = NewPlaylistInfo()

	c.Bitrate = gtk.NewLabel("")
	c.Bitrate.SetSingleLineMode(true)

	bitrateCSS(c.Bitrate)

	c.Right = NewPlaylistControls(c)

	c.RightSide = gtk.NewBox(gtk.OrientationHorizontal, 0)
	c.RightSide.Append(c.Bitrate)
	c.RightSide.Append(c.Right)

	empty := gtk.NewBox(gtk.OrientationHorizontal, 0)

	h := gtk.NewHeaderBar()
	h.SetTitleWidget(empty)
	h.PackStart(c.Left)
	h.PackStart(c.Info)
	h.PackEnd(c.RightSide)

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
	if bits < 1 {
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
