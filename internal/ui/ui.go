package ui

import (
	"fmt"
	"log"

	"github.com/diamondburned/aqours/internal/muse"
	"github.com/diamondburned/aqours/internal/muse/playlist"
	"github.com/diamondburned/aqours/internal/state"
	"github.com/diamondburned/aqours/internal/ui/content"
	"github.com/diamondburned/aqours/internal/ui/css"
	"github.com/diamondburned/aqours/internal/ui/header"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/pkg/errors"
)

func init() {
	css.LoadGlobal("main", `

	`)
}

func assert(b bool, e string) {
	if !b {
		log.Panicln("BUG: assertion failed:", e)
	}
}

// maxErrorThreshold is the error threshold before the player stops seeking.
// Refer to errCounter.
const maxErrorThreshold = 3

type MainWindow struct {
	gtk.ApplicationWindow
	content.Container

	Header *header.Container

	muse  *muse.Session
	state *state.State

	// errCounter is the counter to print errors before pausing.
	errCounter int
}

func NewMainWindow(a *gtk.Application, session *muse.Session) (*MainWindow, error) {
	w, err := gtk.ApplicationWindowNew(a)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create window")
	}
	w.SetTitle("Aqours")
	w.SetDefaultSize(800, 500)

	mw := &MainWindow{
		ApplicationWindow: *w,
		muse:              session,
		state:             state.NewState(),
	}

	mw.Header = header.NewContainer(mw)
	mw.Header.Show()

	w.SetTitlebar(mw.Header)

	mw.Container = content.NewContainer(mw)
	w.Add(mw.ContentBox)

	session.SetHandler(mw)

	return mw, nil
}

// UseState makes the MainWindow use an existing state.
func (w *MainWindow) UseState(s *state.State) {
	w.state = s

	// Restore the state.
	w.Bar.Controls.Buttons.SetRepeat(w.state.RepeatMode(), false)
	w.Bar.Controls.Buttons.SetShuffle(w.state.IsShuffling())

	var selected bool

	for _, name := range w.state.PlaylistNames() {
		playlist, _ := w.state.Playlist(name)
		uiPl := w.Body.Sidebar.PlaylistList.AddPlaylist(playlist)

		if name == w.state.PlayingPlaylistName() {
			w.Body.Sidebar.PlaylistList.SelectPlaylist(uiPl)
			selected = true
		}
	}

	if !selected {
		w.Body.Sidebar.PlaylistList.SelectFirstPlaylist()
	}
}

func (w *MainWindow) GoBack() { w.Body.SwipeBack() }

func (w *MainWindow) OnSongFinish(err error) {
	if err != nil {
		w.errCounter++

		log.Println("Error playing track:", err)

		if w.errCounter > maxErrorThreshold {
			w.SetPlay(false)
			return
		}
	}

	if w.errCounter > 0 {
		w.errCounter = 0
	}

	// Play the next song.
	track := w.state.AutoNext()
	if track != nil {
		w.playTrack(track)
		return
	}
}

func (w *MainWindow) OnPauseUpdate(pause bool) {
	w.Vis.Drawer.SetPaused(pause)
	w.Bar.Controls.Buttons.Play.SetPlaying(!pause)

	if pause {
		w.Header.SetBitrate(-1)
	}
}

func (w *MainWindow) OnBitrateChange(bitrate float64) {
	w.Header.SetBitrate(bitrate)
}

func (w *MainWindow) OnPositionChange(pos, total float64) {
	w.Bar.Controls.Seek.UpdatePosition(pos, total)
}

func (w *MainWindow) AddPlaylist(path string) {
	w.SetSensitive(false)

	go func() {
		p, err := playlist.ParseFile(path)
		if err != nil {
			log.Println("Failed parsing playlist:", err)
		}

		glib.IdleAdd(func() {
			w.SetSensitive(true)

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
		return
	}
}

func (w *MainWindow) Next() {
	track := w.state.Next()
	if track != nil {
		w.playTrack(track)
	}
}

func (w *MainWindow) Previous() {
	track := w.state.Previous()
	if track != nil {
		w.playTrack(track)
	}
}

func (w *MainWindow) SetPlay(playing bool) {
	if err := w.muse.SetPlay(playing); err != nil {
		log.Println("SetPlay failed:", err)
		return
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
	// If we've updated the current playlist, then we should also refresh the
	// play queue.
	if w.state.PlayingPlaylist() == playlist {
		w.state.RefreshQueue()
	}

	w.Header.SetUnsaved(playlist)
	w.Body.Sidebar.PlaylistList.SetUnsaved(playlist)
}

func (w *MainWindow) playTrack(track *state.Track) {
	if err := w.muse.PlayTrack(track.Filepath); err != nil {
		log.Println("PlayTrack failed:", err)
		return
	}

	playing := w.state.PlayingPlaylist()

	trackList, ok := w.Body.TracksView.Lists[playing.Name]
	assert(ok, "track list not found from name: "+playing.Name)

	trackList.SetPlaying(track)
	w.Bar.NowPlaying.SetTrack(track)
	w.Body.Sidebar.AlbumArt.SetTrack(track)
}

func (w *MainWindow) SelectPlaylist(name string) {
	pl, ok := w.state.Playlist(name)
	if !ok {
		log.Println("Playlist not found:", name)
		return
	}

	// Don't change the state's playing playlist.

	w.Body.TracksView.SelectPlaylist(pl)
	w.Header.SetPlaylist(pl)
	w.SetTitle(fmt.Sprintf("%s - Aqours", pl.Name))
}

func (w *MainWindow) SetVolume(perc float64) {
	if err := w.muse.SetVolume(perc); err != nil {
		log.Println("SetVolume failed:", err)
	}
}

func (w *MainWindow) SetMute(muted bool) {
	if err := w.muse.SetMute(muted); err != nil {
		log.Println("SetVolume failed:", muted)
	}
}
