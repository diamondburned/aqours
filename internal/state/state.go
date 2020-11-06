package state

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/diamondburned/aqours/internal/muse/playlist"
	"github.com/gotk3/gotk3/glib"
	"github.com/pkg/errors"
)

var stateDir, stateFile string

func init() {
	stateDir = getConfigDir()
	stateFile = filepath.Join(stateDir, "state.json")
}

func getConfigDir() string {
	d := filepath.Join(glib.GetUserDataDir(), "aqours")

	if err := os.Mkdir(d, os.ModePerm); err != nil && !os.IsExist(err) {
		log.Println("failed to make data directory:", err)
		return ""
	}

	return d
}

type State struct {
	state    jsonState
	playlist *playlist.Playlist // current playlist

	saving chan struct{}
}

type jsonState struct {
	Playlists       []*playlist.Playlist `json:"playlists"`
	CurrentPlaylist string               `json:"current_playlist"`
}

func NewState() *State {
	return &State{
		saving: make(chan struct{}, 1),
	}
}

func ReadFromFile() (*State, error) {
	b, err := ioutil.ReadFile(stateFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read state file")
	}

	return UnmarshalState(b)
}

func UnmarshalState(jsonBytes []byte) (*State, error) {
	var state = NewState()

	if err := json.Unmarshal(jsonBytes, &state.state); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal state JSON")
	}

	// See if we were on any current playlist.
	if state.state.CurrentPlaylist != "" {
		playlist, ok := state.Playlist(state.state.CurrentPlaylist)
		if !ok {
			// Corrupted state. Ignore and move on.
			state.state.CurrentPlaylist = ""
		} else {
			state.playlist = playlist
		}
	}

	return state, nil
}

// Playlist returns a playlist, or nil if none. It also returns a boolean to
// indicate.
func (s *State) Playlist(name string) (*playlist.Playlist, bool) {
	for _, playlist := range s.state.Playlists {
		if playlist.Name == name {
			return playlist, true
		}
	}
	return nil, false
}

// PlaylistFromPath returns a playlist, or nil if none.
func (s *State) PlaylistFromPath(path string) (*playlist.Playlist, bool) {
	for _, playlist := range s.state.Playlists {
		if playlist.Path == path {
			return playlist, true
		}
	}
	return nil, false
}

func (s *State) Playlists() []*playlist.Playlist {
	return s.state.Playlists
}

// SetPlaylist sets a playlist. It does not check for collision.
func (s *State) SetPlaylist(p *playlist.Playlist) {
	for i, playlist := range s.state.Playlists {
		if playlist.Name == p.Name {
			s.state.Playlists[i] = playlist
			return
		}
	}

	s.state.Playlists = append(s.state.Playlists, p)
}

// DeletePlaylist deletes the playlist with the given name.
func (s *State) DeletePlaylist(name string) {
	// TODO: optimize?
	for i, playlist := range s.state.Playlists {
		if playlist.Name == name {
			s.state.Playlists = append(s.state.Playlists[:i], s.state.Playlists[i+1:]...)
			return
		}
	}
}

// SetCurrentPlaylist sets the current playlist.
func (s *State) SetCurrentPlaylist(pl *playlist.Playlist) {
	s.playlist = pl

	if pl == nil {
		s.state.CurrentPlaylist = ""
	} else {
		s.state.CurrentPlaylist = pl.Name
	}
}

// CurrentPlaylist returns the current playlist, or nil if none. It panics if
// the internal states are inconsistent, which should never happen unless the
// playlist pointer was illegally changed. If the path were to be changed, then
// SetCurrentPlaylist should be called again.
func (s *State) CurrentPlaylist() *playlist.Playlist {
	if s.playlist == nil {
		return nil
	}

	if s.playlist.Name != s.state.CurrentPlaylist {
		panic("BUG: s.playlist.Name != s.state.CurrentPlaylist")
	}

	return s.playlist
}

// CurrentPlaylistName returns the current playlist name, or an empty string if
// none.
func (s *State) CurrentPlaylistName() string {
	return s.state.CurrentPlaylist
}

// Save saves the state. It is non-blocking, but the JSON is marshaled in the
// same thread as the caller.
func (s *State) Save() {
	if stateDir == "" {
		return
	}

	// TODO: ignore if not mutated.

	b, err := json.Marshal(s.state)
	if err != nil {
		log.Println("failed to JSON marshal state:", err)
		return
	}

	select {
	case s.saving <- struct{}{}:
		// success
	default:
		return
	}

	go func() {
		if err := ioutil.WriteFile(stateFile, b, os.ModePerm); err != nil {
			log.Println("failed to save JSON state:", err)
		}

		<-s.saving
	}()
}

// ForceSave forces the state to be saved synchronously.
func (s *State) ForceSave() {
	b, err := json.Marshal(s.state)
	if err != nil {
		log.Println("failed to JSON marshal state:", err)
		return
	}

	if err := ioutil.WriteFile(stateFile, b, os.ModePerm); err != nil {
		log.Println("failed to save JSON state:", err)
	}
}
