package ui

import (
	"github.com/diamondburned/aqours/internal/muse/playlist"
	"github.com/diamondburned/aqours/internal/ui/content/body/tracks"
)

type state struct {
	Playlists map[string]*playlist.Playlist
	Playlist  *playlist.Playlist // current playlist
	TrackList *tracks.TrackList
}

func newState() state {
	return state{
		Playlists: map[string]*playlist.Playlist{},
	}
}
