package state

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"github.com/diamondburned/aqours/internal/muse/playlist"
)

type jsonPlaylist struct {
	Name PlaylistName
	Path string
}

type jsonState struct {
	Playlists []jsonPlaylist `json:"playlist_names"`
	Metadata  metadataMap    `json:"metadata"`

	PlayingPlaylist  string `json:"playing_playlist,omitempty"`   // playlist name
	PlayingSongIndex int    `json:"playing_song_index,omitempty"` // song index

	Shuffling bool       `json:"shuffling"`
	Repeating RepeatMode `json:"repeating"`
}

func fileJSONState(file string) (jsonState, error) {
	var jsonState jsonState

	f, err := os.Open(file)
	if err != nil {
		return jsonState, err
	}
	f.SetDeadline(time.Now().Add(10 * time.Second))
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&jsonState); err != nil {
		return jsonState, err
	}

	return jsonState, nil
}

func makeJSONState(s *State) jsonState {
	var playlists = make([]jsonPlaylist, len(s.playlistNames))
	for i, name := range s.playlistNames {
		playlists[i] = jsonPlaylist{
			Name: name,
			Path: s.playlists[name].Path,
		}
	}

	playingPlaylist := s.PlayingPlaylistName()
	playingSongIndex := 0

	if len(s.playing.Queue) > 0 {
		playingSongIndex = s.playing.Queue[s.playing.QueuePos]
	}

	return jsonState{
		Playlists:        playlists,
		Metadata:         s.metadata,
		Shuffling:        s.shuffling,
		Repeating:        s.repeating,
		PlayingPlaylist:  playingPlaylist,
		PlayingSongIndex: playingSongIndex,
	}
}

func makeStateFromJSON(jsonState jsonState) *State {
	state := &State{
		onUpdate:      func(s *State) { s.unsaved = true },
		saving:        make(chan struct{}, 1),
		metadata:      jsonState.Metadata,
		playlists:     make(map[PlaylistName]*Playlist, len(jsonState.Playlists)),
		playlistNames: make([]PlaylistName, 0, len(jsonState.Playlists)),
		shuffling:     jsonState.Shuffling,
		repeating:     jsonState.Repeating,
	}

	// Load playlists multithreaded.
	playlists := make([]*playlist.Playlist, len(jsonState.Playlists))
	waitGroup := sync.WaitGroup{}
	waitGroup.Add(len(jsonState.Playlists))

	for i, pl := range jsonState.Playlists {
		go func(i int, pl jsonPlaylist) {
			p, err := playlist.ParseFile(pl.Path)
			if err != nil {
				log.Printf("Ignoring erroneous playlist at %q, reason: %v\n", pl.Path, err)
			} else {
				p.Name = pl.Name
				playlists[i] = p
			}

			waitGroup.Done()
		}(i, pl)
	}

	waitGroup.Wait()

	for _, pl := range playlists {
		if pl == nil {
			continue
		}

		playlist := convertPlaylist(state, pl)

		state.playlistNames = append(state.playlistNames, playlist.Name)
		state.playlists[playlist.Name] = playlist
	}

	// Drop metadata with no references.
	for k, metadata := range state.metadata {
		if metadata.reference < 1 {
			delete(state.metadata, k)
		}
	}

	// Attempt to restore the currently playing states.

	if jsonState.PlayingPlaylist == "" {
		return state
	}

	pl, ok := state.playlists[jsonState.PlayingPlaylist]
	if !ok {
		return state
	}

	state.SetPlayingPlaylist(pl)

	// Attempt to find the right position by value from the queue position.
	for i, ix := range state.playing.Queue {
		if ix == jsonState.PlayingSongIndex {
			state.playing.QueuePos = i
			break
		}
	}

	return state
}
