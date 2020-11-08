package header

import (
	"fmt"
	"html"

	"github.com/gotk3/gotk3/gtk"
)

type PlaylistInfo struct {
	gtk.Label // name
	Playlist  string
	Unsaved   bool
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
	info.SetMarkup("<b>Aqours</b>")
	info.Playlist = ""
}

func (info *PlaylistInfo) SetPlaylist(name string) {
	info.SetText(name)
	info.Playlist = name
}

func (info *PlaylistInfo) SetUnsaved(unsaved bool) {
	info.Unsaved = unsaved

	if unsaved {
		info.SetMarkup(fmt.Sprintf("<i>%s</i>", html.EscapeString(info.Playlist)))
	} else {
		info.SetText(info.Playlist)
	}
}
