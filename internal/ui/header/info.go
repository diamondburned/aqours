package header

import (
	"github.com/diamondburned/aqours/internal/state"
	"github.com/gotk3/gotk3/gtk"
)

type PlaylistInfo struct {
	gtk.Label // name
	Playlist  string
}

func NewPlaylistInfo() *PlaylistInfo {
	name, _ := gtk.LabelNew("")
	name.SetYAlign(0)
	name.SetVAlign(gtk.ALIGN_CENTER)
	name.SetSingleLineMode(true)
	name.Show()

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
