package ui

import "github.com/diamondburned/aqours/internal/muse/playlist"

type state struct {
	Playlists map[string]*playlist.Playlist
	Playlist  *playlist.Playlist // current playlist
}

func newState() state {
	return state{
		Playlists: map[string]*playlist.Playlist{},
	}
}
