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

type MainWindow struct {
	gtk.ApplicationWindow

	Header  *header.Container
	Content *content.Container

	muse  *muse.Session
	state *state.State
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

	mw.Content = content.NewContainer(mw)
	mw.Content.Show()

	w.Add(mw.Content)

	session.SetHandler(mw)

	return mw, nil
}

// UseState makes the MainWindow use an existing state.
func (w *MainWindow) UseState(s *state.State) {
	w.state = s

	for _, p := range w.state.Playlists() {
		uiPl := w.Content.Body.Sidebar.PlaylistList.AddPlaylist(p.Name)
		uiPl.SetTotal(len(p.Tracks))

		if p.Name == w.state.CurrentPlaylistName() {
			w.Content.Body.Sidebar.PlaylistList.SelectPlaylist(uiPl)
		}
	}
}

func (w *MainWindow) GoBack() { w.Content.Body.SwipeBack() }

func (w *MainWindow) OnPathUpdate(playlistPath, songPath string) {
	playlist, ok := w.state.PlaylistFromPath(playlistPath)
	if !ok {
		log.Println("Playlist not found from path:", playlistPath)
		return
	}

	trackList, ok := w.Content.Body.TracksView.Lists[playlist.Name]
	if !ok {
		log.Println("Track list not found from name:", playlist.Name)
		return
	}

	track, ok := trackList.Tracks[songPath]
	if !ok {
		log.Println("Track not found in track list from path:", songPath)
		return
	}

	trackList.SetPlaying(track)
	w.Content.Bar.NowPlaying.SetTrack(track.Track)
	w.Content.Body.Sidebar.AlbumArt.SetTrack(track.Track)
}

func (w *MainWindow) OnPauseUpdate(pause bool) {
	w.Content.Vis.Drawer.SetPaused(pause)
	w.Content.Bar.Controls.Buttons.Play.SetPlaying(!pause)

	if pause {
		w.Header.SetBitrate(-1)
	}
}

func (w *MainWindow) OnRepeatChange(repeat muse.RepeatMode) {
	// Don't trigger the callback to our methods, as that will cause a feedback
	// loop.
	w.Content.Bar.Controls.Buttons.SetRepeat(repeat, false)
}

func (w *MainWindow) OnBitrateChange(bitrate float64) {
	w.Header.SetBitrate(bitrate)
}

func (w *MainWindow) OnPositionChange(pos, total float64) {
	w.Content.Bar.Controls.Seek.UpdatePosition(pos, total)
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
				log.Println("Duplicated playlist name", p.Name)
				return
			}

			uiPl := w.Content.Body.Sidebar.PlaylistList.AddPlaylist(p.Name)
			uiPl.SetTotal(len(p.Tracks))

			w.state.SetPlaylist(p)
		})
	}()
}

func (w *MainWindow) HasPlaylist(name string) bool {
	_, ok := w.state.Playlist(name)
	return ok
}

// RenamePlaylist renames a playlist. It only works if we're renaming the
// current playlist.
func (w *MainWindow) RenamePlaylist(name, newName string) bool {
	pl, ok := w.state.Playlist(name)
	if !ok {
		log.Println("Playlist not found:", name)
		return false
	}

	// Collision check.
	if _, exists := w.state.Playlist(newName); exists {
		log.Println("Playlist's new name already exists:", newName)
		return false
	}

	pl.Name = newName
	w.state.SetPlaylist(pl)
	w.state.DeletePlaylist(name)

	w.Content.Body.TracksView.DeletePlaylist(name)
	w.Content.Body.Sidebar.PlaylistList.Playlist(name).SetName(newName)
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
	if err := w.muse.Next(); err != nil {
		log.Println("Next failed:", err)
		return
	}
}

func (w *MainWindow) Previous() {
	if err := w.muse.Previous(); err != nil {
		log.Println("Previous failed:", err)
		return
	}
}

func (w *MainWindow) SetPlay(playing bool) {
	if err := w.muse.SetPlay(playing); err != nil {
		log.Println("SetPlay failed:", err)
		return
	}
}

func (w *MainWindow) SetShuffle(shuffle bool) {
	if err := w.muse.SetShuffle(shuffle); err != nil {
		log.Println("SetShuffle failed:", err)
		return
	}
}

func (w *MainWindow) SetRepeat(mode muse.RepeatMode) {
	if err := w.muse.SetRepeat(mode); err != nil {
		log.Println("SetRepeat failed:", err)
	}
}

func (w *MainWindow) PlayTrack(playlistName string, n int) {
	pl, ok := w.state.Playlist(playlistName)
	if !ok {
		log.Println("failed to find playlist from name:", playlistName)
		return
	}

	if err := w.muse.SelectPlaylist(pl.Path); err != nil {
		log.Println("SelectPlaylist failed:", err)
		return
	}

	if err := w.muse.PlayTrackIndex(n); err != nil {
		log.Println("PlayTrackIndex failed:", err)
		return
	}
}

func (w *MainWindow) SelectPlaylist(name string) {
	pl, ok := w.state.Playlist(name)
	if !ok {
		log.Println("Playlist not found:", name)
		return
	}

	w.state.SetCurrentPlaylist(pl)

	tracks := w.Content.Body.TracksView.SelectPlaylist(pl.Name)
	tracks.SetTracks(pl.Tracks)

	w.Header.SetPlaylist(pl)
	w.SetTitle(fmt.Sprintf("%s - Aqours", pl.Name))
}
