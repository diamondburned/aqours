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

func failIf(b bool, e string) {
	if b {
		log.Panicln("BUG: assertion failed:", e)
	}
}

type State struct {
	state jsonState

	playing struct {
		Playlist  *playlist.Playlist
		PlayQueue []*playlist.Track
		QueuePos  int
	}

	saving chan struct{}
}

type jsonState struct {
	Playlists []*playlist.Playlist `json:"playlists"`

	PlayingPlaylist string `json:"playing_playlist"`   // playlist name
	PlayingSongPath string `json:"playing_song_index"` // song path

	Shuffling bool       `json:"shuffling"`
	Repeating RepeatMode `json:"repeating"`
}

func NewState() *State {
	return &State{
		saving: make(chan struct{}, 1),
		state:  jsonState{},
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
	if state.state.PlayingPlaylist != "" {
		playlist, ok := state.Playlist(state.state.PlayingPlaylist)
		if !ok {
			// Corrupted state. Ignore and move on.
			state.state.PlayingPlaylist = ""
			return state, nil
		}

		state.SetPlayingPlaylist(playlist)

		// See if we could restore the track as well.
		if state.state.PlayingSongPath != "" {
			// Search.
			i, track := state.trackFromPath(state.state.PlayingSongPath)
			if track == nil {
				// Corrupted state. Ignore and move on.
				state.state.PlayingSongPath = ""
			} else {
				state.playing.QueuePos = i
			}
		}
	}

	return state, nil
}

// assertCoherentState asserts everything in the JSON state with the helper
// pointers.
func (s *State) assertCoherentState() {
	if s.playing.Playlist != nil {
		failIf(
			s.playing.Playlist.Name != s.state.PlayingPlaylist,
			"playing.Playlist.Name is not equal to state's PlayingPlaylist",
		)
	}
}

// Playlist returns a playlist, or nil if none. It also returns a boolean to
// indicate.
func (s *State) Playlist(name string) (*playlist.Playlist, bool) {
	s.assertCoherentState()

	if s.playing.Playlist != nil && s.playing.Playlist.Name == name {
		return s.playing.Playlist, true
	}

	for _, playlist := range s.state.Playlists {
		if playlist.Name == name {
			return playlist, true
		}
	}

	return nil, false
}

// PlaylistFromPath returns a playlist, or nil if none.
func (s *State) PlaylistFromPath(path string) (*playlist.Playlist, bool) {
	s.assertCoherentState()

	if s.playing.Playlist != nil && s.playing.Playlist.Path == path {
		return s.playing.Playlist, true
	}

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

			if playlist.Name == s.playing.Playlist.Name {
				s.SetPlayingPlaylist(nil)
			}

			return
		}
	}
}

// SetPlayingPlaylist sets the playing playlist.
func (s *State) SetPlayingPlaylist(pl *playlist.Playlist) {
	s.assertCoherentState()

	s.playing.Playlist = pl
	s.state.PlayingSongPath = "" // always

	if pl == nil {
		s.state.PlayingPlaylist = ""
		s.playing.PlayQueue = nil
		return
	}

	s.state.PlayingPlaylist = pl.Name
	s.playing.PlayQueue = make([]*playlist.Track, len(pl.Tracks))
	// Copy all tracks.
	copy(s.playing.PlayQueue, pl.Tracks)
}

// PlayingPlaylist returns the playing playlist, or nil if none. It panics if
// the internal states are inconsistent, which should never happen unless the
// playlist pointer was illegally changed. If the path were to be changed, then
// SetCurrentPlaylist should be called again.
func (s *State) PlayingPlaylist() *playlist.Playlist {
	// coherency check skipped for performance.
	return s.playing.Playlist
}

// PlayingPlaylistName returns the playing playlist name, or an empty string if
// none.
func (s *State) PlayingPlaylistName() string {
	s.assertCoherentState()

	return s.state.PlayingPlaylist
}

// NowPlaying returns the currently playing track. If playingPl is nil, then
// this method returns (-1, nil).
func (s *State) NowPlaying() (int, *playlist.Track) {
	s.assertCoherentState()

	if s.playing.Playlist == nil {
		return -1, nil
	}

	return s.trackFromPath(s.state.PlayingSongPath)
}

// IsShuffling returns true if the list is being shuffled.
func (s *State) IsShuffling() bool {
	return s.state.Shuffling
}

// SetShuffling sets the shuffling mode.
func (s *State) SetShuffling(shuffling bool) {
	s.state.Shuffling = shuffling

	if s.playing.Playlist == nil {
		return
	}

	s.assertCoherentState()

	if shuffling {
		playlist.ShuffleTracks(s.playing.PlayQueue)
		return
	}

	copy(s.playing.PlayQueue, s.playing.Playlist.Tracks)

	// Attempt to renew the QueuePos.
	for i, track := range s.playing.PlayQueue {
		if track.Filepath == s.state.PlayingSongPath {
			s.playing.QueuePos = i
			break
		}
	}
}

// RepeatMode returns the current repeat mode.
func (s *State) RepeatMode() RepeatMode {
	return s.state.Repeating
}

// SetRepeatMode sets the current repeat mode.
func (s *State) SetRepeatMode(mode RepeatMode) {
	s.state.Repeating = mode
}

// ReloadPlayQueue reloads the internal play queue for the currently playing
// playlist. Call this when the playlist's track slice is changed.
func (s *State) ReloadPlayQueue() {
	s.assertCoherentState()

	if s.playing.Playlist == nil {
		return
	}

	if newlen := len(s.playing.Playlist.Tracks); newlen != len(s.playing.PlayQueue) {
		s.playing.PlayQueue = make([]*playlist.Track, newlen)
	}

	copy(s.playing.PlayQueue, s.playing.Playlist.Tracks)

	// Restore shuffling.
	if s.state.Shuffling {
		s.SetShuffling(true)
	}
}

// trackFromPath searches from the scratch list.
func (s *State) trackFromPath(path string) (int, *playlist.Track) {
	for i, track := range s.playing.PlayQueue {
		if track.Filepath == path {
			return i, track
		}
	}
	return -1, nil
}

func (s *State) nowPlayingTrack() *playlist.Track {
	return s.playing.PlayQueue[s.playing.QueuePos]
}

// Play plays the track indexed relative to the actual playlist. This does not
// index relative to the actual play queue, which may be shuffled.
func (s *State) Play(index int) *playlist.Track {
	return s.play(index, false)
}

func (s *State) play(index int, shuffled bool) *playlist.Track {
	// Ensure that we have an active playing playlist.
	failIf(s.playing.Playlist == nil, "playing.Playlist is nil while Play is called")
	// Assert the state after modifying the index.
	s.assertCoherentState()
	// Bound check.
	failIf(index < 0, "index is negative")
	failIf(index >= len(s.playing.PlayQueue), "given index is out of bounds in Play")

	var track *playlist.Track
	// TODO: watch for inconsistencies.
	if shuffled {
		track = s.playing.PlayQueue[index]
	} else {
		track = s.playing.Playlist.Tracks[index]
	}

	s.playing.QueuePos = index
	s.state.PlayingSongPath = track.Filepath

	return track
}

// Previous returns the previous track, similarly to Next. Nil is returned if
// there is no previous track. If shuffling mode is on, then Prev will not
// return the previous track.
func (s *State) Previous() *playlist.Track {
	return s.move(false, true)
}

// Next returns the next track from the currently playing playlist. Nil is
// returned if there is no next track.
func (s *State) Next() *playlist.Track {
	return s.move(true, true)
}

// AutoNext returns the next track, unless we're in RepeatSingle mode, then it
// returns the same track. Use this to cycle across the playlist.
func (s *State) AutoNext() *playlist.Track {
	return s.move(true, false)
}

// move is an abstracted function used by Prev, Next and AutoNext.
func (s *State) move(forward, force bool) *playlist.Track {
	s.assertCoherentState()

	if s.playing.Playlist == nil {
		return nil
	}

	if !force && s.state.Repeating == RepeatSingle {
		return s.nowPlayingTrack()
	}

	next, oob := spinIndex(forward, s.playing.QueuePos, len(s.playing.PlayQueue))

	if oob && s.state.Repeating == RepeatNone {
		return nil
	}

	return s.play(next, true)
}

// spinIndex spins the index. It returns the newly spun index and whether it was
// spun back.
func spinIndex(fwd bool, i, max int) (int, bool) {
	if fwd {
		i++

		if i >= max {
			return 0, true
		}
	} else {
		i--

		if i < 0 {
			return max - 1, true
		}
	}

	return i, false
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
