package ui

import (
	"fmt"
	"log"
	"time"

	"github.com/diamondburned/aqours/internal/muse"
	"github.com/diamondburned/aqours/internal/muse/playlist"
	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/aqours/internal/ui/content"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/diamondburned/aqours/internal/ui/header"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func init() {
	css.AddGlobal(``)
}

func assert(b bool, e string) {
	if !b {
		log.Panicln("BUG: assertion failed:", e)
	}
}

// TODO
// // MusicPlayer is an interface for a music player backend. All its methods
// // should preferably be non-blocking, and thus should handle error reporting on
// // its own.
// type MusicPlayer interface {
// 	PlayTrack(path string)
// 	Seek(pos float64)
// 	SetPlay(bool)
// 	SetMute(bool)
// 	SetVolume(float64)
// }

// maxErrorThreshold is the error threshold before the player stops seeking.
// Refer to errCounter.
const maxErrorThreshold = 3

const minPlayLength = 250 * time.Millisecond

type MainWindow struct {
	content.Container

	Window *gtk.ApplicationWindow
	Header *header.Container

	muse  *muse.Session
	state *state.State

	lastPlayed time.Time
	skipCount  int
}

func NewMainWindow(
	a *gtk.Application, session *muse.Session, s *state.State) (*MainWindow, error) {

	window := gtk.NewApplicationWindow(a)
	window.SetTitle("Aqours")
	window.SetDefaultSize(800, 500)
	window.ConnectMap(func() {
		surface := window.Surface()
		display := gdk.BaseSurface(surface).Display()
		css.LoadGlobal(display)
	})

	w := &MainWindow{
		Window: window,
		muse:   session,
	}

	w.Header = header.NewContainer(w)
	window.SetTitlebar(w.Header)

	w.Container = content.NewContainer(w)
	window.SetChild(w.ContentBox)

	w.useState(s)

	// Use a low-priority 250ms poller instead of updating live.
	glib.TimeoutAddPriority(250, glib.PriorityDefaultIdle, func() bool {
		pos, rem := session.PlayState.PlayTime()
		w.Bar.Controls.Seek.UpdatePosition(pos, pos+rem)

		w.Header.SetBitrate(session.PlayState.Bitrate())

		return true
	})

	return w, nil
}

// Present shows and focuses the window.
func (w *MainWindow) Present() {
	w.Window.Present()
}

// PlaySession returns the internal playback session.
func (w *MainWindow) PlaySession() *muse.Session {
	return w.muse
}

// State exposes the local state that was passed in.
func (w *MainWindow) State() *state.State {
	return w.state
}

// useState makes the MainWindow use an existing state.
func (w *MainWindow) useState(s *state.State) {
	w.state = s

	// Restore the state. These calls will update the observer.
	w.SetRepeat(w.state.RepeatMode())
	w.SetShuffle(w.state.IsShuffling())
	// These calls will update MainWindow through signals.
	w.Bar.SetMute(w.state.IsMuted())
	w.Bar.Volume.SetVolume(w.state.Volume())

	var selected *state.Playlist

	playlistNames := w.state.PlaylistNames()
	if len(playlistNames) == 0 {
		// First run; can't restore state.
		return
	}

	for _, name := range playlistNames {
		playlist, _ := w.state.Playlist(name)
		uiPl := w.Body.Sidebar.PlaylistList.AddPlaylist(playlist)

		if name == w.state.PlayingPlaylistName() {
			w.Body.Sidebar.PlaylistList.SelectPlaylist(uiPl)
			selected = playlist
		}
	}

	// If there's no active selection, then try the first playlist.
	if selected == nil {
		w.Body.Sidebar.PlaylistList.SelectFirstPlaylist()
		selected, _ = w.state.Playlist(playlistNames[0])

		// Ensure we're selecting the right playlist.
		w.state.SetPlayingPlaylist(selected)
	}

	// If there is finally a selection, then update the track list. This is nil
	// when there is no playlist.
	trackList := w.Body.TracksView.SelectPlaylist(selected)

	// Update the playing track if we have one. NowPlaying should return a track
	// from the given playlist.
	_, track := w.state.NowPlaying()
	if track != nil {
		trackList.SetPlaying(track)
	}
}

func (w *MainWindow) GoBack() { w.Body.SwipeBack() }

// OnSongFinish plays the next song in the playlist. If the error given is not
// nil, then it'll gradually seek to the next song until either no error is
// given anymore or the error counter hits its max.
func (w *MainWindow) OnSongFinish() {
	// Seek the bar back to 0 immediately.
	w.Bar.Controls.Seek.UpdatePosition(0, 0)

	now := time.Now()

	// Are we going too quickly?
	if w.lastPlayed.Add(minPlayLength).After(now) {
		// Increment skip count. If we're over the bound, then stop.
		w.skipCount++
		log.Println("Track too short. Skipped tracks:", w.skipCount)
	} else {
		w.skipCount = 0
	}

	if w.skipCount > maxErrorThreshold {
		log.Println("Skipped tracks over threshold, stopping.")
		return
	}

	w.lastPlayed = now

	// Play the next song.
	_, track := w.state.AutoNext()
	if track != nil {
		w.playTrack(track)
	}
}

func (w *MainWindow) OnPauseUpdate(pause bool) {
	w.Bar.SetPaused(pause)
	if pause {
		w.Header.SetBitrate(-1)
	}
}

func (w *MainWindow) AddPlaylist(path string) {
	w.Window.SetSensitive(false)

	go func() {
		p, err := playlist.ParseFile(path)
		if err != nil {
			log.Println("Failed parsing playlist:", err)
		}

		glib.IdleAdd(func() {
			w.Window.SetSensitive(true)

			if err != nil {
				return
			}

			if _, ok := w.state.Playlist(p.Name); ok {
				// Try and mangle the name.
				var mangle int
				var name string

				for {
					if _, ok := w.state.Playlist(p.Name); !ok {
						break
					}
					mangle++
					p.Name = fmt.Sprintf("%s~%d", name, mangle)
				}
			}

			playlist := w.state.AddPlaylist(p)
			w.Body.Sidebar.PlaylistList.AddPlaylist(playlist)
		})
	}()
}

func (w *MainWindow) HasPlaylist(name string) bool {
	_, ok := w.state.Playlist(name)
	return ok
}

func (w *MainWindow) SaveAllPlaylists() {
	for _, name := range w.state.PlaylistNames() {
		pl, _ := w.state.Playlist(name)
		w.SavePlaylist(pl)
	}
}

func (w *MainWindow) SavePlaylist(pl *state.Playlist) {
	if !pl.IsUnsaved() {
		return
	}

	refresh := func() {
		w.Header.SetUnsaved(pl)
		w.Body.Sidebar.PlaylistList.SetUnsaved(pl)
	}
	// Visually indicate the saved status.
	refresh()

	pl.Save(func(err error) {
		glib.IdleAdd(refresh)
		if err != nil {
			log.Println("failed to save playlist:", err)
		}
	})
}

// RenamePlaylist renames a playlist. It only works if we're renaming the
// current playlist.
func (w *MainWindow) RenamePlaylist(pl *state.Playlist, newName string) bool {
	// Collision check.
	if _, exists := w.state.Playlist(newName); exists {
		return false
	}

	plName := pl.Name
	pl.Name = newName
	w.state.RenamePlaylist(pl, plName)

	w.Body.TracksView.DeletePlaylist(plName)
	w.Body.Sidebar.PlaylistList.Playlist(plName).SetName(newName)
	w.SelectPlaylist(newName)

	return true
}

func (w *MainWindow) Seek(pos float64) {
	if err := w.muse.Seek(pos); err != nil {
		log.Println("Seek failed:", err)
	}
}

func (w *MainWindow) Next() {
	_, track := w.state.Next()
	if track != nil {
		w.playTrack(track)
	}
}

func (w *MainWindow) Previous() {
	_, track := w.state.Previous()
	if track != nil {
		w.playTrack(track)
	}
}

func (w *MainWindow) SetPlay(playing bool) {
	if err := w.muse.SetPlay(playing); err != nil {
		log.Println("SetPlay failed:", err)
	}
}

func (w *MainWindow) SetShuffle(shuffle bool) {
	w.state.SetShuffling(shuffle)
	w.Bar.Controls.Buttons.SetShuffle(shuffle)
}

func (w *MainWindow) SetRepeat(mode state.RepeatMode) {
	w.state.SetRepeatMode(mode)
	w.Bar.Controls.Buttons.SetRepeat(mode, false)
}

func (w *MainWindow) PlayTrack(playlist *state.Playlist, n int) {
	// Change the playing playlist if needed.
	if w.state.PlayingPlaylistName() != playlist.Name {
		w.state.SetPlayingPlaylist(playlist)
	}

	w.playTrack(w.state.Play(n))
}

func (w *MainWindow) UpdateTracks(playlist *state.Playlist) {
	w.Header.SetUnsaved(playlist)
	w.Body.Sidebar.PlaylistList.SetUnsaved(playlist)

	// If we've updated the current playlist, then we should also refresh the
	// play queue.
	if w.state.PlayingPlaylist() == playlist {
		w.state.RefreshQueue()
	}
}

func (w *MainWindow) playTrack(track *state.Track) {
	var nextPath string
	if _, nextTrack := w.state.Peek(); nextTrack != nil {
		nextPath = nextTrack.Filepath
	}

	w.muse.PlayTrack(track.Filepath, nextPath)
	playing := w.state.PlayingPlaylist()

	trackList, ok := w.Body.TracksView.Lists[playing.Name]
	assert(ok, "track list not found from name: "+playing.Name)

	trackList.SetPlaying(track)
	w.Bar.NowPlaying.SetTrack(track)
	w.Body.Sidebar.AlbumArt.SetTrack(track)

	// Save the state asynchronously.
	w.state.SaveState()
}

// SortSelectedTracks sorts the selected tracks.
func (w *MainWindow) SortSelectedTracks() {
	list := w.Body.TracksView.SelectPlaylist(w.state.PlayingPlaylist())
	list.SortSelected()
}

func (w *MainWindow) SelectPlaylist(name string) {
	pl, ok := w.state.Playlist(name)
	if !ok {
		log.Println("Playlist not found:", name)
		return
	}

	w.selectPlaylist(pl)
}

func (w *MainWindow) selectPlaylist(pl *state.Playlist) {
	// Don't change the state's playing playlist.

	trackList := w.Body.TracksView.SelectPlaylist(pl)
	trackList.SelectPlaying()

	w.Header.SetPlaylist(pl)
	w.Window.SetTitle(fmt.Sprintf("%s - Aqours", pl.Name))
}

func (w *MainWindow) ScrollToPlaying() {
	w.selectPlaylist(w.state.PlayingPlaylist())
}

func (w *MainWindow) SetVolume(perc float64) {
	if err := w.muse.SetVolume(perc); err != nil {
		log.Println("SetVolume failed:", err)
		return
	}

	w.state.SetVolume(perc)
}

func (w *MainWindow) SetMute(mute bool) {
	if err := w.muse.SetMute(mute); err != nil {
		log.Println("SetMute failed:", mute)
		return
	}

	w.state.SetMute(mute)
}
