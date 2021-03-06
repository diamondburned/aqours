package sidebar

import (
	"fmt"

	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/diamondburned/handy"
	"github.com/gotk3/gotk3/gtk"
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

	lbox, _ := gtk.ListBoxNew()
	lbox.SetSelectionMode(gtk.SELECTION_BROWSE)
	lbox.SetActivateOnSingleClick(true)
	lbox.Connect("row-activated", func(_ *gtk.ListBox, r *gtk.ListBoxRow) {
		parent.SelectPlaylist(list.Playlists[r.GetIndex()].Name)
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
	l.ListBox.Add(playlist)
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
	l.SelectRow(&pl.ListBoxRow)
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
	handy.ActionRow
	Name  string
	Total int
}

func NewPlaylist(name string, total int) *Playlist {
	arow := handy.ActionRowNew()
	arow.SetActivatable(true)
	arow.Show()

	pl := &Playlist{ActionRow: *arow}
	pl.SetName(name)
	pl.SetTotal(total)

	return pl
}

func (pl *Playlist) SetUnsaved(unsaved bool) {
	if !unsaved {
		pl.SetTitle(pl.Name)
	} else {
		pl.SetTitle(pl.Name + " ●")
	}
}

func (pl *Playlist) SetName(name string) {
	pl.SetTitle(name)
	pl.Name = name
}

func (pl *Playlist) SetTotal(total int) {
	pl.SetSubtitle(fmt.Sprintf("%d songs", total))
	pl.Total = total
}
