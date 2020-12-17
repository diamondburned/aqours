package state

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sync"

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
	// onUpdate is called when a playing track is updated.
	onUpdate func(s *State)

	metadata      metadataMap
	playlistNames []PlaylistName
	playlists     map[PlaylistName]*Playlist

	playing struct {
		Playlist *Playlist
		Queue    []int // list of indices to playlists[playing.Playlist]
		QueuePos int   // relative to Queue
	}

	shuffling bool
	repeating RepeatMode

	saving  chan struct{}
	unsaved bool
}

func NewState() *State {
	return &State{
		onUpdate: func(s *State) { s.unsaved = true },

		saving:    make(chan struct{}, 1),
		metadata:  make(metadataMap),
		playlists: make(map[PlaylistName]*Playlist),
	}
}

func ReadFromFile() (*State, error) {
	s, err := fileJSONState(stateFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read state file")
	}

	return makeStateFromJSON(s), nil
}

// OnTrackUpdate adds into the call stack a callback that is triggered when the
// state is changed.
func (s *State) OnUpdate(fn func(*State)) {
	old := s.onUpdate
	s.onUpdate = func(s *State) {
		old(s)
		fn(s)
	}
}

// RefreshQueue refreshes completely the current play queue.
func (s *State) RefreshQueue() {
	s.SetPlayingPlaylist(s.playing.Playlist)
}

// assertCoherentState asserts everything in the JSON state with the helper
// pointers.
func (s *State) assertCoherentState() {
	// Nothing left. TODO: refactor this out.
}

// Playlist returns a playlist, or nil if none. It also returns a boolean to
// indicate.
func (s *State) Playlist(name string) (*Playlist, bool) {
	s.assertCoherentState()

	pl, ok := s.playlists[name]
	return pl, ok
}

// PlaylistFromPath returns a playlist, or nil if none.
func (s *State) PlaylistFromPath(path string) (*Playlist, bool) {
	s.assertCoherentState()

	if s.playing.Playlist != nil && s.playing.Playlist.Path == path {
		return s.playing.Playlist, true
	}

	for _, playlist := range s.playlists {
		if playlist.Path == path {
			return playlist, true
		}
	}

	return nil, false
}

func (s *State) PlaylistNames() []PlaylistName {
	return s.playlistNames
}

func (s *State) RenamePlaylist(p *Playlist, oldName string) {
	pl, ok := s.playlists[oldName]
	if !ok {
		return
	}

	delete(s.playlists, oldName)
	s.playlists[p.Name] = pl

	for i, name := range s.playlistNames {
		if name == oldName {
			s.playlistNames[i] = p.Name
			break
		}
	}

	pl.Name = p.Name

	s.onUpdate(s)
}

// AddPlaylist adds a playlist. If a playlist with the same name is added, then
// the function does nothing.
func (s *State) AddPlaylist(p *playlist.Playlist) *Playlist {
	if _, ok := s.playlists[p.Name]; ok {
		log.Println("Playlist collision while adding:", p.Name)
		return nil
	}

	playlist := convertPlaylist(s, p)
	playlist.unsaved = 1

	s.playlists[p.Name] = playlist
	s.playlistNames = append(s.playlistNames, p.Name)

	s.onUpdate(s)

	return playlist
}

// DeletePlaylist deletes the playlist with the given name.
func (s *State) DeletePlaylist(name string) {
	// TODO: optimize?
	for i, playlistName := range s.playlistNames {
		if playlistName == name {
			s.playlistNames = append(s.playlistNames[:i], s.playlistNames[i+1:]...)
			delete(s.playlists, name)

			if name == s.playing.Playlist.Name {
				s.SetPlayingPlaylist(nil)
			}

			s.onUpdate(s)

			return
		}
	}
}

// SetPlayingPlaylist sets the playing playlist.
func (s *State) SetPlayingPlaylist(pl *Playlist) {
	s.assertCoherentState()

	defer s.onUpdate(s)

	s.playing.Playlist = pl
	s.playing.QueuePos = 0 // reset QueuePos as well

	if pl == nil {
		s.playing.Queue = nil
		return
	}

	s.ReloadPlayQueue()

	if s.shuffling {
		// Reshuffle.
		playlist.ShuffleQueue(s.playing.Queue)
	}
}

// PlayingPlaylist returns the playing playlist, or nil if none. It panics if
// the internal states are inconsistent, which should never happen unless the
// playlist pointer was illegally changed. If the path were to be changed, then
// SetCurrentPlaylist should be called again.
func (s *State) PlayingPlaylist() *Playlist {
	// coherency check skipped for performance.
	return s.playing.Playlist
}

// PlayingPlaylistName returns the playing playlist name, or an empty string if
// none.
func (s *State) PlayingPlaylistName() string {
	s.assertCoherentState()

	if s.playing.Playlist == nil {
		return ""
	}

	return s.playing.Playlist.Name
}

// NowPlaying returns the currently playing track. If playingPl is nil, then
// this method returns (-1, nil).
func (s *State) NowPlaying() (int, *Track) {
	s.assertCoherentState()

	if s.playing.Playlist == nil {
		return -1, nil
	}

	ix := s.playing.Queue[s.playing.QueuePos]
	return ix, s.playing.Playlist.Tracks[ix]
}

// IsShuffling returns true if the list is being shuffled.
func (s *State) IsShuffling() bool {
	return s.shuffling
}

// SetShuffling sets the shuffling mode.
func (s *State) SetShuffling(shuffling bool) {
	// Do nothing if we're setting the same thing. Helps a bit w/ state
	// inconsistency.
	if s.shuffling == shuffling {
		return
	}

	defer s.onUpdate(s)

	s.shuffling = shuffling

	if s.playing.Playlist == nil {
		return
	}

	s.assertCoherentState()

	if shuffling {
		playlist.ShuffleQueue(s.playing.Queue)
		return
	}

	// Attempt to renew the QueuePos before changing the queue. As Queue holds a
	// list of actual track indices, we could use the queue position as the key
	// to get the actual position, then set that to the queue position.
	s.playing.QueuePos = s.playing.Queue[s.playing.QueuePos]

	playlist.ResetQueue(s.playing.Queue)
}

// RepeatMode returns the current repeat mode.
func (s *State) RepeatMode() RepeatMode {
	return s.repeating
}

// SetRepeatMode sets the current repeat mode.
func (s *State) SetRepeatMode(mode RepeatMode) {
	if s.repeating == mode {
		return
	}

	s.repeating = mode

	s.onUpdate(s)
}

// ReloadPlayQueue reloads the internal play queue for the currently playing
// playlist. Call this when the playlist's track slice is changed.
func (s *State) ReloadPlayQueue() {
	s.assertCoherentState()

	if s.playing.Playlist == nil {
		return
	}

	if newlen := len(s.playing.Playlist.Tracks); newlen != len(s.playing.Queue) {
		s.playing.Queue = make([]int, newlen)
	}

	playlist.ResetQueue(s.playing.Queue)

	// Restore shuffling.
	if s.shuffling {
		playlist.ShuffleQueue(s.playing.Queue)
	}

	s.onUpdate(s)
}

// Play plays the track indexed relative to the actual playlist. This does not
// index relative to the actual play queue, which may be shuffled. It also does
// not update QueuePos if we're shuffling.
func (s *State) Play(index int) *Track {
	// Only update QueuePos if we're not shuffling.
	if !s.shuffling {
		s.playing.QueuePos = index
		defer s.onUpdate(s)
	}

	return s.trackFromPlaylist(index)
}

func (s *State) trackFromQueue(index int) *Track {
	// Bound check.
	failIf(index < 0, "index is negative")
	failIf(index >= len(s.playing.Queue), "given index is out of bounds in Play")

	return s.trackFromPlaylist(s.playing.Queue[index])
}

func (s *State) trackFromPlaylist(index int) *Track {
	// Ensure that we have an active playing playlist.
	failIf(s.playing.Playlist == nil, "playing.Playlist is nil while Play is called")
	// Assert the state after modifying the index.
	s.assertCoherentState()

	// Bound check.
	failIf(index < 0, "index is negative")
	failIf(index >= len(s.playing.Playlist.Tracks), "given index is out of bounds in Play")

	return s.playing.Playlist.Tracks[index]
}

// Previous returns the previous track, similarly to Next. Nil is returned if
// there is no previous track. If shuffling mode is on, then Prev will not
// return the previous track.
func (s *State) Previous() (int, *Track) {
	return s.move(false, true)
}

// Next returns the next track from the currently playing playlist. Nil is
// returned if there is no next track.
func (s *State) Next() (int, *Track) {
	return s.move(true, true)
}

// AutoNext returns the next track, unless we're in RepeatSingle mode, then it
// returns the same track. Use this to cycle across the playlist.
func (s *State) AutoNext() (int, *Track) {
	return s.move(true, false)
}

// Peek returns the next track without changing the state. It basically emulates
// AutoNext.
func (s *State) Peek() (int, *Track) {
	return s.peek(true, false)
}

// move is an abstracted function used by Prev, Next and AutoNext.
func (s *State) move(forward, force bool) (int, *Track) {
	next, track := s.peek(forward, force)
	s.playing.QueuePos = next
	s.onUpdate(s)
	return next, track
}

func (s *State) peek(forward, force bool) (int, *Track) {
	s.assertCoherentState()

	if !force && s.repeating == RepeatSingle {
		return s.NowPlaying()
	}

	next, oob := spinIndex(forward, s.playing.QueuePos, len(s.playing.Queue))

	if oob && s.repeating == RepeatNone {
		return -1, nil
	}

	return next, s.trackFromQueue(next)
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

// SaveState saves the state. It is non-blocking, but the JSON is marshaled in
// the same thread as the caller.
func (s *State) SaveState() {
	if stateDir == "" {
		return
	}

	if !s.unsaved {
		return
	}

	select {
	case s.saving <- struct{}{}:
		// success
	default:
		return
	}

	b, err := json.Marshal(makeJSONState(s))
	if err != nil {
		log.Println("failed to JSON marshal state:", err)
		return
	}

	s.unsaved = false

	go func() {
		if err := ioutil.WriteFile(stateFile, b, os.ModePerm); err != nil {
			log.Println("failed to save JSON state:", err)
		}

		<-s.saving
	}()
}

func (s *State) SaveAll() {
	b, err := json.Marshal(makeJSONState(s))
	if err != nil {
		log.Println("Failed to JSON marshal state:", err)
		return
	}

	s.unsaved = false

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		if err := ioutil.WriteFile(stateFile, b, os.ModePerm); err != nil {
			log.Println("Failed to save JSON state:", err)
		}
		wg.Done()
	}()

	wg.Add(len(s.playlists))

	for _, pl := range s.playlists {
		pl.Save(func(err error) {
			if err != nil {
				log.Println("Failed to save playlist:", err)
			}
			wg.Done()
		})
	}

	wg.Wait()
}
