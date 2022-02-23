package sidebar

import (
	"fmt"

	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/diamondburned/gotk4/pkg/pango"
)

var playlistsCSS = css.PrepareClass("playlists", `
	list.playlists {
		background: @theme_bg_color;
	}
`)

type PlaylistList struct {
	gtk.ListBox
	parent ParentController

	Playlists []*Playlist
}

func NewPlaylistList(parent ParentController) *PlaylistList {
	list := &PlaylistList{parent: parent}

	lbox := gtk.NewListBox()
	lbox.SetSelectionMode(gtk.SelectionBrowse)
	lbox.SetActivateOnSingleClick(true)
	lbox.Connect("row-activated", func(_ *gtk.ListBox, r *gtk.ListBoxRow) {
		parent.SelectPlaylist(list.Playlists[r.Index()].Name)
	})
	playlistsCSS(lbox)

	list.ListBox = *lbox

	return list
}

func (l *PlaylistList) AddPlaylist(pl *state.Playlist) *Playlist {
	for _, playlist := range l.Playlists {
		if playlist.Name == pl.Name {
			return playlist
		}
	}

	playlist := NewPlaylist(pl.Name, len(pl.Tracks))

	l.ListBox.Append(playlist)
	l.Playlists = append(l.Playlists, playlist)

	return playlist
}

// SelectFirstPlaylist selects the first playlist. It does nothing if there are
// no playlists.
func (l *PlaylistList) SelectFirstPlaylist() *Playlist {
	if len(l.Playlists) > 0 {
		l.SelectPlaylist(l.Playlists[0])
		return l.Playlists[0]
	}
	return nil
}

// SelectPlaylist selects the given playlist.
func (l *PlaylistList) SelectPlaylist(pl *Playlist) {
	l.SelectRow(pl.ListBoxRow)
	pl.Activate()
	l.parent.SelectPlaylist(pl.Name)
}

func (l *PlaylistList) SetUnsaved(pl *state.Playlist) {
	if p := l.Playlist(pl.Name); p != nil {
		p.SetUnsaved(pl.IsUnsaved())
	}
}

func (l *PlaylistList) Playlist(name string) *Playlist {
	for _, playlist := range l.Playlists {
		if playlist.Name == name {
			return playlist
		}
	}
	return nil
}

type Playlist struct {
	*gtk.ListBoxRow
	name  *gtk.Label
	total *gtk.Label

	Name  string
	Total int
}

var playlistEntryCSS = css.PrepareClass("playlist-entry", `
	.playlist-entry > box {
		margin: 6px 8px;
	}
	.playlist-entry > box > label:first-child {
		font-size: 1.1em;
	}
	.playlist-entry > box > label:last-child {
		font-size: 0.9em;
		color: alpha(@theme_fg_color, 0.75);
	}
`)

func NewPlaylist(name string, total int) *Playlist {
	pl := Playlist{}
	pl.name = gtk.NewLabel("")
	pl.name.SetXAlign(0)
	pl.name.SetEllipsize(pango.EllipsizeEnd)

	pl.total = gtk.NewLabel("")
	pl.total.SetXAlign(0)

	box := gtk.NewBox(gtk.OrientationVertical, 0)
	box.Append(pl.name)
	box.Append(pl.total)

	pl.ListBoxRow = gtk.NewListBoxRow()
	pl.ListBoxRow.SetChild(box)
	playlistEntryCSS(pl)

	pl.SetName(name)
	pl.SetTotal(total)

	return &pl
}

func (pl *Playlist) SetUnsaved(unsaved bool) {
	if !unsaved {
		pl.name.SetLabel(pl.Name)
	} else {
		pl.name.SetLabel(pl.Name + " ●")
	}
}

func (pl *Playlist) SetName(name string) {
	pl.name.SetLabel(name)
	pl.Name = name
}

func (pl *Playlist) SetTotal(total int) {
	pl.total.SetLabel(fmt.Sprintf("%d songs", total))
	pl.Total = total
}
