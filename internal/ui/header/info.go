package header

import (
	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type PlaylistInfo struct {
	gtk.Label // name
	Playlist  string
}

func NewPlaylistInfo() *PlaylistInfo {
	name := gtk.NewLabel("")
	name.SetYAlign(0)
	name.SetVAlign(gtk.AlignCenter)
	name.SetSingleLineMode(true)

	info := &PlaylistInfo{
		Label: *name,
	}

	info.Reset()

	return info
}

func (info *PlaylistInfo) Reset() {
	info.Playlist = ""
	info.SetMarkup("<b>Aqours</b>")
}

func (info *PlaylistInfo) SetPlaylist(pl *state.Playlist) {
	info.Playlist = pl.Name
	info.SetText(pl.Name)
	info.SetUnsaved(pl.IsUnsaved())
}

func (info *PlaylistInfo) SetUnsaved(unsaved bool) {
	if !unsaved {
		info.SetText(info.Playlist)
	} else {
		info.SetText(info.Playlist + " ●")
	}
}
