package header

import (
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
	info.SetLabel("Aqours")
	info.Playlist = ""
}

func (info *PlaylistInfo) SetPlaylist(name string) {
	info.SetLabel(name)
	info.Playlist = name
}
